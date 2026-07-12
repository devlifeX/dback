package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"dback/internal/app"
	"dback/internal/audit"
	"dback/internal/config"
	"dback/internal/controlplane"
	"dback/internal/daemon"
	"dback/internal/event"
	"dback/internal/metrics"
	"dback/internal/notify"
	"dback/internal/operation"
	"dback/internal/store"
	"dback/internal/trigger"

	apiv1 "dback/internal/api/v1"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "serve":
		os.Exit(runServe(os.Args[2:]))
	case "unlock-status":
		os.Exit(runUnlockStatus(os.Args[2:]))
	case "user":
		os.Exit(runUser(os.Args[2:]))
	case "run":
		os.Exit(runCommand(os.Args[2:]))
	case "task":
		os.Exit(runTask(os.Args[2:]))
	case "notify":
		os.Exit(runNotify(os.Args[2:]))
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		printUsage()
		os.Exit(2)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage:
  dback serve
  dback unlock-status
  dback user create --phone <mobile> --password <pass> [--name <name>]
  dback run operation <kind> --profile <id>
  dback task list
  dback task run <id> [--profile <id>]
  dback task enable <id>
  dback task disable <id>
  dback notify test --channel <id>

Operation kinds: backup_db, backup_files, upload, url_checker
`)
}

func runServe(args []string) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	application, unlock, err := openApp(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vault: %v\n", err)
		return 1
	}
	if user, created, err := application.EnsureDefaultAdmin(cfg.DefaultAdminPhone, cfg.DefaultAdminPassword, cfg.DefaultAdminName); err != nil {
		fmt.Fprintf(os.Stderr, "default admin: %v\n", err)
		return 1
	} else if created {
		fmt.Printf("default admin created phone=%s name=%s\n", user.Phone, user.Name)
	}
	cp := controlplane.NewService(application, cfg.QueueCapacity, cfg.MaxConcurrent)

	var metricsCollector *metrics.Collector
	if cfg.MetricsEnabled {
		metricsCollector = metrics.NewCollector()
		metricsCollector.Start(cp.Bus)
	}

	auditWriter := audit.NewWriter(cfg.AuditCap)
	auditWriter.Start(cp.Bus)

	notifyStore, notifyNamer := application.NotifyDeps()
	notifyRouter := notify.NewRouter(cp.Bus, notifyStore, notify.NewRegistry(), notifyNamer)
	if metricsCollector != nil {
		notifyRouter.SetMetrics(metricsCollector)
	}
	notifyRouter.Start()
	taskRunner := controlplane.NewTaskRunner(application, cp.Dispatcher, cp.Bus)
	registry := trigger.NewRegistry(trigger.AppStore(application), taskRunner, trigger.RealClock{}, 10*time.Second)
	if err := registry.Start(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "triggers: %v\n", err)
		return 1
	}

	apiHandler := &apiv1.Handler{
		App:      application,
		CP:       cp,
		Triggers: registry,
		Notify:   notifyRouter,
		Audit:    auditWriter,
		Metrics:  metricsCollector,
		Cfg:      cfg,
	}
	server := daemon.NewServer(cfg, cp, registry, notifyRouter, apiHandler, unlock)
	server.SetShutdownHooks(auditWriter.Stop, func() {
		if metricsCollector != nil {
			metricsCollector.Stop()
		}
	})
	fmt.Printf("dback serve listening on %s (data dir: %s)\n", cfg.Listen, cfg.DataDir)
	if err := server.Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "serve: %v\n", err)
		return 1
	}
	return 0
}

func runTask(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: dback task list|run|enable|disable ...")
		return 2
	}
	switch args[0] {
	case "list":
		return taskList(args[1:])
	case "run":
		return taskRun(args[1:])
	case "enable":
		return taskSetEnabled(args[1:], true)
	case "disable":
		return taskSetEnabled(args[1:], false)
	default:
		fmt.Fprintf(os.Stderr, "unknown task subcommand %q\n", args[0])
		return 2
	}
}

func taskList(args []string) int {
	fs := flag.NewFlagSet("task list", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	application, unlock, err := openApp(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vault: %v\n", err)
		return 1
	}
	defer unlock()

	tasks, err := application.ListTasks()
	if err != nil {
		fmt.Fprintf(os.Stderr, "list tasks: %v\n", err)
		return 1
	}
	if len(tasks) == 0 {
		fmt.Println("no tasks")
		return 0
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tENABLED\tTRIGGER\tNEXT_RUN\tPROFILES")
	for _, t := range tasks {
		next := "-"
		if !t.State.NextRunAt.IsZero() {
			next = t.State.NextRunAt.Format(time.RFC3339)
		}
		fmt.Fprintf(w, "%s\t%s\t%t\t%s\t%s\t%d\n",
			t.ID, t.Name, t.Enabled, t.Trigger.Type, next, len(t.ProfileIDs))
	}
	_ = w.Flush()
	return 0
}

func taskRun(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: dback task run <id> [--profile <id>]")
		return 2
	}
	taskID := args[0]
	fs := flag.NewFlagSet("task run", flag.ContinueOnError)
	profileID := fs.String("profile", "", "run for a single profile only")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	application, unlock, err := openApp(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vault: %v\n", err)
		return 1
	}
	defer unlock()

	task, err := application.GetTask(taskID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "task: %v\n", err)
		return 1
	}
	cp := controlplane.NewService(application, cfg.QueueCapacity, cfg.MaxConcurrent)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = cp.Dispatcher.Shutdown(ctx)
	}()
	runner := controlplane.NewTaskRunner(application, cp.Dispatcher, cp.Bus)
	registry := trigger.NewRegistry(trigger.AppStore(application), runner, trigger.RealClock{}, time.Minute)
	_ = registry.Resync(context.Background())

	var profileIDs []string
	if *profileID != "" {
		profileIDs = []string{*profileID}
	}
	if err := registry.FireNow(context.Background(), taskID, profileIDs); err != nil {
		fmt.Fprintf(os.Stderr, "task run failed: %v\n", err)
		return 1
	}
	fmt.Printf("task %q (%s) finished\n", task.Name, taskID)
	return 0
}

func taskSetEnabled(args []string, enabled bool) int {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "usage: dback task %s <id>\n", map[bool]string{true: "enable", false: "disable"}[enabled])
		return 2
	}
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	application, unlock, err := openApp(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vault: %v\n", err)
		return 1
	}
	defer unlock()

	if err := application.SetTaskEnabled(args[0], enabled); err != nil {
		fmt.Fprintf(os.Stderr, "task: %v\n", err)
		return 1
	}
	fmt.Printf("task %s %sd\n", args[0], map[bool]string{true: "enable", false: "disable"}[enabled])
	return 0
}

func runUnlockStatus(args []string) int {
	fs := flag.NewFlagSet("unlock-status", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	application, err := app.NewWithOptions(store.Options{BaseDir: cfg.DataDir, DB: cfg.DB})
	if err != nil {
		fmt.Fprintf(os.Stderr, "app: %v\n", err)
		return 1
	}
	switch {
	case !application.HasVault() && !application.HasLegacyPlaintext():
		fmt.Println("no vault")
	case application.IsUnlocked():
		fmt.Println("unlocked")
	default:
		fmt.Println("locked")
	}
	return 0
}

func runCommand(args []string) int {
	if len(args) < 2 || args[0] != "operation" {
		fmt.Fprintf(os.Stderr, "usage: dback run operation <kind> --profile <id>\n")
		return 2
	}
	kind := operation.Kind(args[1])
	fs := flag.NewFlagSet("run operation", flag.ContinueOnError)
	profileID := fs.String("profile", "", "host profile id")
	triggerRef := fs.String("trigger", "cli", "trigger reference")
	if err := fs.Parse(args[2:]); err != nil {
		return 2
	}
	if *profileID == "" {
		fmt.Fprintln(os.Stderr, "--profile is required")
		return 2
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	application, unlock, err := openApp(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vault: %v\n", err)
		return 1
	}
	defer unlock()

	cp := controlplane.NewService(application, cfg.QueueCapacity, cfg.MaxConcurrent)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = cp.Dispatcher.Shutdown(ctx)
	}()

	spec, err := specForKind(kind)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 2
	}
	spec.ProfileID = *profileID
	spec.TriggerRef = *triggerRef

	ctx := context.Background()
	rec, err := cp.Dispatcher.Submit(ctx, spec)
	if rec != nil {
		fmt.Printf("operation_id=%s status=%s\n", rec.ID, rec.Status)
		if rec.Error != "" {
			fmt.Printf("error=%s\n", rec.Error)
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "operation failed: %v\n", err)
		return 1
	}
	if rec != nil && rec.Status != operation.StatusSucceeded {
		return 1
	}
	return 0
}

func specForKind(kind operation.Kind) (operation.Spec, error) {
	switch kind {
	case operation.KindBackupDB:
		return operation.Spec{Kind: kind, Params: operation.BackupDBParams{}}, nil
	case operation.KindBackupFiles:
		return operation.Spec{Kind: kind, Params: operation.BackupFilesParams{}}, nil
	case operation.KindUpload:
		return operation.Spec{
			Kind:   kind,
			Params: operation.UploadParams{StalePolicy: operation.UploadStaleNewOnly},
		}, nil
	case operation.KindUrlChecker:
		return operation.Spec{Kind: kind, Params: operation.UrlCheckerParams{}}, nil
	default:
		return operation.Spec{}, fmt.Errorf("unsupported operation kind %q (supported: %s)", kind, strings.Join([]string{
			string(operation.KindBackupDB),
			string(operation.KindBackupFiles),
			string(operation.KindUpload),
			string(operation.KindUrlChecker),
		}, ", "))
	}
}

func runNotify(args []string) int {
	if len(args) == 0 || args[0] != "test" {
		fmt.Fprintln(os.Stderr, "usage: dback notify test --channel <id>")
		return 2
	}
	fs := flag.NewFlagSet("notify test", flag.ContinueOnError)
	channelID := fs.String("channel", "", "notification channel id")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if *channelID == "" {
		fmt.Fprintln(os.Stderr, "--channel is required")
		return 2
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	application, unlock, err := openApp(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vault: %v\n", err)
		return 1
	}
	defer unlock()

	bus := event.NewMemoryBus(8)
	notifyStore, notifyNamer := application.NotifyDeps()
	router := notify.NewRouter(bus, notifyStore, notify.NewRegistry(), notifyNamer)
	if err := router.Test(context.Background(), *channelID); err != nil {
		fmt.Fprintf(os.Stderr, "notify test failed: %v\n", err)
		return 1
	}
	fmt.Printf("notify test sent via channel %s\n", *channelID)
	return 0
}

func runUser(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: dback user create --phone <mobile> --password <pass>")
		return 2
	}
	switch args[0] {
	case "create":
		return userCreate(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown user subcommand %q\n", args[0])
		return 2
	}
}

func userCreate(args []string) int {
	fs := flag.NewFlagSet("user create", flag.ContinueOnError)
	phone := fs.String("phone", "", "user mobile phone (09XXXXXXXXX)")
	password := fs.String("password", "", "user password")
	name := fs.String("name", "Admin", "display name")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *phone == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "--phone and --password are required")
		return 2
	}
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	application, unlock, err := openApp(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vault: %v\n", err)
		return 1
	}
	defer unlock()
	user, err := application.CreateUser(*phone, *password, *name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create user: %v\n", err)
		return 1
	}
	fmt.Printf("user created id=%s phone=%s name=%s\n", user.ID, user.Phone, user.Name)
	return 0
}

func openApp(cfg config.Config) (*app.App, func(), error) {
	application, err := app.NewWithOptions(store.Options{BaseDir: cfg.DataDir, DB: cfg.DB})
	if err != nil {
		return nil, nil, err
	}
	unlock := func() {
		application.Lock()
	}
	if application.IsUnlocked() {
		return application, unlock, nil
	}
	if cfg.Passphrase == "" {
		return nil, nil, fmt.Errorf("vault locked; set DBACK_PASSPHRASE or DBACK_PASSPHRASE_FILE")
	}
	if application.HasVault() || application.HasLegacyPlaintext() {
		if err := application.Unlock(cfg.Passphrase); err != nil {
			return nil, nil, err
		}
		return application, unlock, nil
	}
	if err := application.CreateVault(cfg.Passphrase); err != nil {
		return nil, nil, err
	}
	return application, unlock, nil
}
