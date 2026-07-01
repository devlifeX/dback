package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"dback/backend/db"
	"dback/backend/ssh"
	"dback/backend/transfer"
	"dback/backend/wordpress"
	"dback/internal/debug"
	"dback/internal/paths"
	"dback/internal/store"
	"dback/models"
)

type ProgressFunc = transfer.ProgressFunc

type App struct {
	store *store.Store

	mu        sync.Mutex
	profiles  []models.Profile
	templates []models.SQLTemplate
	history   []models.ExportRecord
	logs      []models.LogEntry
}

func New(baseDir string) (*App, error) {
	knownHostsPath := filepath.Join(baseDir, "ssh_known_hosts")
	log.Printf("app.New: baseDir=%q knownHostsFile=%q", baseDir, knownHostsPath)
	ssh.SetKnownHostsFile(knownHostsPath)
	return &App{store: store.New(baseDir)}, nil
}

func (a *App) HasVault() bool {
	return a.store.HasVault()
}

func (a *App) HasLegacyPlaintext() bool {
	return a.store.HasLegacyPlaintext()
}

func (a *App) IsUnlocked() bool {
	return a.store.IsUnlocked()
}

func (a *App) DataRevision() uint64 {
	return a.store.Revision()
}

func (a *App) CreateVault(passphrase string) error {
	log.Printf("app.CreateVault: creating vault")
	if err := a.store.CreateVault(passphrase); err != nil {
		log.Printf("app.CreateVault: store.CreateVault failed: %v", err)
		return err
	}
	log.Printf("app.CreateVault: vault created, reloading")
	return a.Reload()
}

func (a *App) Unlock(passphrase string) error {
	log.Printf("app.Unlock: unlocking vault")
	if err := a.store.Unlock(passphrase); err != nil {
		log.Printf("app.Unlock: store.Unlock failed: %v", err)
		return err
	}
	log.Printf("app.Unlock: vault unlocked, reloading")
	return a.Reload()
}

func (a *App) Lock() {
	a.mu.Lock()
	a.profiles = nil
	a.templates = nil
	a.history = nil
	a.logs = nil
	a.mu.Unlock()
	a.store.Lock()
}

func (a *App) Reload() error {
	log.Printf("app.Reload: loading profiles")
	profiles, err := a.store.LoadProfiles()
	if err != nil {
		log.Printf("app.Reload: LoadProfiles failed: %v", err)
		return err
	}
	migratedProfiles := false
	for i := range profiles {
		if newDest, ok := paths.MigrateBackupDestination(profiles[i].Destination); ok {
			log.Printf("app.Reload: migrated backup destination for profile %q: %q -> %q", profiles[i].Name, profiles[i].Destination, newDest)
			profiles[i].Destination = newDest
			migratedProfiles = true
		}
	}
	log.Printf("app.Reload: loading templates")
	templates, err := a.store.LoadTemplates()
	if err != nil {
		log.Printf("app.Reload: LoadTemplates failed: %v", err)
		return err
	}
	log.Printf("app.Reload: loading history")
	history, err := a.store.LoadHistory()
	if err != nil {
		log.Printf("app.Reload: LoadHistory failed: %v", err)
		return err
	}
	log.Printf("app.Reload: loading logs")
	logs, err := a.store.LoadLogs()
	if err != nil {
		log.Printf("app.Reload: LoadLogs failed: %v", err)
		return err
	}
	a.mu.Lock()
	a.profiles = profiles
	a.templates = templates
	a.history = history
	a.logs = logs
	a.mu.Unlock()

	if migratedProfiles {
		if err := a.store.SaveProfiles(profiles); err != nil {
			log.Printf("app.Reload: SaveProfiles after destination migration failed: %v", err)
			return err
		}
	}
	return nil
}

func (a *App) Profiles() []models.Profile {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]models.Profile(nil), a.profiles...)
}

func (a *App) Templates() []models.SQLTemplate {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]models.SQLTemplate(nil), a.templates...)
}

func (a *App) History() []models.ExportRecord {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]models.ExportRecord(nil), a.history...)
}

func (a *App) Logs() []models.LogEntry {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]models.LogEntry(nil), a.logs...)
}

func (a *App) SaveProfile(profile models.Profile) error {
	if profile.ID == "" {
		profile.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	if profile.Group == "" {
		profile.Group = "Default"
	}
	if profile.UsesWordPress() {
		if err := ensureWordPressAPIKey(&profile); err != nil {
			return err
		}
	}
	if err := normalizeProfileFileBackup(&profile); err != nil {
		return err
	}
	profile.ExportSettings = nil
	profile.ImportSettings = nil

	a.mu.Lock()
	found := false
	for i := range a.profiles {
		if a.profiles[i].ID == profile.ID {
			a.profiles[i] = profile
			found = true
			break
		}
	}
	if !found {
		a.profiles = append(a.profiles, profile)
	}
	profiles := append([]models.Profile(nil), a.profiles...)
	a.mu.Unlock()

	return a.store.SaveProfiles(profiles)
}

func (a *App) DeleteProfile(id string) error {
	a.mu.Lock()
	for i := range a.profiles {
		if a.profiles[i].ID == id {
			a.profiles = append(a.profiles[:i], a.profiles[i+1:]...)
			break
		}
	}
	profiles := append([]models.Profile(nil), a.profiles...)
	a.mu.Unlock()
	return a.store.SaveProfiles(profiles)
}

func (a *App) SaveTemplate(t models.SQLTemplate) error {
	if t.ID == "" {
		t.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	now := time.Now()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now

	a.mu.Lock()
	found := false
	for i := range a.templates {
		if a.templates[i].ID == t.ID {
			t.CreatedAt = a.templates[i].CreatedAt
			a.templates[i] = t
			found = true
			break
		}
	}
	if !found {
		a.templates = append(a.templates, t)
	}
	templates := append([]models.SQLTemplate(nil), a.templates...)
	a.mu.Unlock()
	return a.store.SaveTemplates(templates)
}

func (a *App) ReplaceTemplateInProfiles(oldBody, newBody string, profileIDs map[string]struct{}) error {
	if len(profileIDs) == 0 {
		return nil
	}
	a.mu.Lock()
	a.profiles = models.ReplaceTemplateInProfiles(a.profiles, oldBody, newBody, profileIDs)
	profiles := append([]models.Profile(nil), a.profiles...)
	a.mu.Unlock()
	return a.store.SaveProfiles(profiles)
}

func (a *App) DeleteTemplate(id string) error {
	a.mu.Lock()
	for i := range a.templates {
		if a.templates[i].ID == id {
			a.templates = append(a.templates[:i], a.templates[i+1:]...)
			break
		}
	}
	templates := append([]models.SQLTemplate(nil), a.templates...)
	a.mu.Unlock()
	return a.store.SaveTemplates(templates)
}

func (a *App) ExportProfiles(path string, includeSecrets bool, passphrase string) error {
	return a.store.ExportProfiles(path, a.Profiles(), includeSecrets, passphrase)
}

func (a *App) PreviewImportProfiles(path string, includeSecrets bool, passphrase string) ([]models.Profile, []store.ProfileConflict, error) {
	imported, err := a.store.ImportProfilesBundle(path, includeSecrets, passphrase)
	if err != nil {
		return nil, nil, err
	}
	conflicts := store.DetectProfileConflicts(a.Profiles(), imported)
	return imported, conflicts, nil
}

func (a *App) ImportProfiles(path string, includeSecrets bool, passphrase string) error {
	imported, err := a.store.ImportProfilesBundle(path, includeSecrets, passphrase)
	if err != nil {
		return err
	}
	a.mu.Lock()
	a.profiles = store.MergeProfiles(a.profiles, imported)
	profiles := append([]models.Profile(nil), a.profiles...)
	a.mu.Unlock()
	return a.store.SaveProfiles(profiles)
}

func (a *App) ExportAppData(path string, includeSecrets bool, passphrase string) error {
	destinations, _ := a.store.LoadRemoteDestinations()
	appDestID, _ := a.store.AppSettingsDestinationID()
	syncSettings, _ := a.SyncSettings()
	return a.store.ExportAppData(path, store.AppImportData{
		Profiles:                 a.Profiles(),
		Templates:                a.Templates(),
		History:                  a.History(),
		Logs:                     a.Logs(),
		Sync:                     syncSettings,
		RemoteDestinations:       destinations,
		AppSettingsDestinationID: appDestID,
	}, includeSecrets, passphrase)
}

func (a *App) PreviewImportAppData(path string, includeSecrets bool, passphrase string) (store.AppImportData, []store.ProfileConflict, []store.TemplateConflict, error) {
	imported, err := a.store.ImportAppDataBundle(path, includeSecrets, passphrase)
	if err != nil {
		return store.AppImportData{}, nil, nil, err
	}
	profileConflicts := store.DetectProfileConflicts(a.Profiles(), imported.Profiles)
	templateConflicts := store.DetectTemplateConflicts(a.Templates(), imported.Templates)
	return imported, profileConflicts, templateConflicts, nil
}

func (a *App) ImportAppData(path string, includeSecrets bool, passphrase string) error {
	imported, err := a.store.ImportAppDataBundle(path, includeSecrets, passphrase)
	if err != nil {
		return err
	}
	return a.applyImportedAppData(imported)
}

func (a *App) Backup(ctx context.Context, profile models.Profile, progress ProgressFunc) (models.ExportRecord, error) {
	operationID := newID()
	started := time.Now()
	dest := paths.EffectiveBackupDestination(profile.Destination)
	if err := os.MkdirAll(dest, 0755); err != nil {
		return models.ExportRecord{}, err
	}

	logger := a.newOpLogger(operationID, &profile)
	a.logPhase(operationID, &profile, "Export", "start", "", 0, "Starting backup", "Info", "Started", "")

	var fullPath string
	var size int64
	var err error

	var result transfer.BackupResult
	var backupErr error
	if profile.UsesWordPress() {
		result, backupErr = transfer.BackupWordPress(ctx, transfer.BackupRequest{
			Profile:     profile,
			OperationID: operationID,
			Destination: dest,
			Logger:      logger,
			Progress:    progress,
		})
	} else {
		result, backupErr = transfer.BackupSSH(ctx, transfer.BackupRequest{
			Profile:     profile,
			OperationID: operationID,
			Destination: dest,
			Logger:      logger,
			Progress:    progress,
		})
	}
	fullPath, size, err = result.Path, result.Size, backupErr

	if err != nil {
		if errors.Is(err, context.Canceled) {
			a.logPhase(operationID, &profile, "Export", "cancel", "", 0, "Backup canceled", "Info", "Canceled", "")
			return models.ExportRecord{}, err
		}
		a.logPhase(operationID, &profile, "Export", "failure", "", 0, "Backup failed", "Error", "Failed", err.Error())
		return models.ExportRecord{}, err
	}

	if size < 128 {
		_ = os.Remove(fullPath)
		err = fmt.Errorf("backup file too small (%d bytes)", size)
		a.logPhase(operationID, &profile, "Export", "validation", "", 0, err.Error(), "Error", "Failed", err.Error())
		return models.ExportRecord{}, err
	}

	record := models.ExportRecord{
		ID:             newID(),
		OperationID:    operationID,
		ProfileID:      profile.ID,
		ProfileName:    profile.Name,
		DatabaseName:   profile.TargetDBName,
		ExportType:     models.ExportTypeDatabase,
		ExportDate:     time.Now(),
		FilePath:       fullPath,
		FileSize:       formatSize(size),
		FileSizeBytes:  size,
		ConnectionType: profile.ConnectionType,
	}
	if profile.UsesWordPress() && strings.TrimSpace(record.DatabaseName) == "" {
		record.DatabaseName = "wordpress"
	}
	a.mu.Lock()
	a.history = append(a.history, record)
	history := append([]models.ExportRecord(nil), a.history...)
	a.mu.Unlock()
	if err := a.store.SaveHistory(history); err != nil {
		return record, err
	}

	if progress != nil {
		progress("Capturing fingerprint...", size, size)
	}
	sha256, fingerprint := a.captureBackupMetadata(ctx, profile, fullPath, profile.TargetDBName)
	record.Sha256 = sha256
	record.Fingerprint = fingerprint

	if progress != nil {
		progress("Verifying backup integrity...", size, size)
	}
	applyAutoQuickVerify(&record)
	if record.VerificationMethod == "" {
		if record.QuickVerified != nil && record.QuickVerified.Passed {
			record.VerificationMethod = models.VerifyQuick
		} else if record.Sha256 != "" {
			record.VerificationMethod = models.VerifySHA256
		}
	}
	if err := a.UpdateHistoryRecord(record); err != nil {
		return record, err
	}

	a.logPhaseWithFile(operationID, profile, "Export", "complete", "", 0, fmt.Sprintf("Backup completed in %s", time.Since(started).Round(time.Millisecond)), "Info", "Succeeded", "", fullPath, size)
	if progress != nil {
		progress("Backup completed", size, size)
	}
	return record, nil
}

func (a *App) RunImportQuery(ctx context.Context, profile models.Profile, query string, connectDB bool) (db.QueryResult, error) {
	if err := ctx.Err(); err != nil {
		return db.QueryResult{}, err
	}
	if !profile.SupportsSQLQuery() {
		return db.QueryResult{}, fmt.Errorf("SQL query is not supported for this host")
	}

	if profile.UsesWordPress() {
		client, err := wordpress.NewClient(profile)
		if err != nil {
			return db.QueryResult{}, err
		}
		database := ""
		if connectDB {
			database = db.WordPressImportDatabase(profile)
		}
		result, err := client.Query(ctx, query, database)
		if err != nil {
			return db.QueryResult{
				Columns: []string{"Error"},
				Rows:    [][]string{{truncateTestOutput(err.Error())}},
				Message: err.Error(),
			}, err
		}
		return result, nil
	}

	cmd, err := db.BuildQueryCommand(profile, query, connectDB)
	if err != nil {
		return db.QueryResult{}, err
	}
	debug.Log("Debug", "Query", "Started", db.MaskCommand(cmd), profile.Name, "", "")

	client, err := ssh.NewExecutor(profile)
	if err != nil {
		return db.QueryResult{}, err
	}
	defer client.Close()

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = client.Close()
		case <-done:
		}
	}()
	defer close(done)

	output, err := client.RunCommand(cmd)
	if ctx.Err() != nil {
		return db.QueryResult{}, ctx.Err()
	}
	output = strings.TrimSpace(output)
	if err != nil {
		return db.QueryResult{
			Columns: []string{"Error"},
			Rows:    [][]string{{output}},
			Message: err.Error(),
		}, fmt.Errorf("%w: %s", err, output)
	}
	result := db.ParseMySQLBatchOutput(output)
	result.Message = output
	return result, nil
}

func (a *App) Restore(ctx context.Context, record models.ExportRecord, destination models.Profile, progress ProgressFunc) error {
	if !destination.AllowsImport() {
		return fmt.Errorf("host %q is protected from import", destination.Name)
	}
	operationID := newID()
	started := time.Now()
	if info, err := os.Stat(record.FilePath); err != nil {
		return err
	} else if info.Size() < 128 {
		return fmt.Errorf("backup file too small (%d bytes)", info.Size())
	}
	logger := a.newOpLogger(operationID, &destination)
	a.logPhaseWithFile(operationID, destination, "Import", "start", "", 0, "Starting restore", "Info", "Started", "", record.FilePath, record.FileSizeBytes)

	if err := a.runPreImportQueryPhase(ctx, operationID, destination, record.FileSizeBytes, progress); err != nil {
		return err
	}

	var err error
	if destination.UsesWordPress() {
		err = transfer.RestoreWordPress(ctx, transfer.RestoreRequest{
			Profile:     destination,
			OperationID: operationID,
			LocalPath:   record.FilePath,
			FileSize:    record.FileSizeBytes,
			Logger:      logger,
			Progress:    progress,
		})
	} else {
		err = transfer.RestoreSSH(ctx, transfer.RestoreRequest{
			Profile:     destination,
			OperationID: operationID,
			LocalPath:   record.FilePath,
			FileSize:    record.FileSizeBytes,
			Logger:      logger,
			Progress:    progress,
		})
	}

	if err != nil {
		if errors.Is(err, context.Canceled) {
			a.logPhase(operationID, &destination, "Import", "cancel", "", 0, "Restore canceled", "Info", "Canceled", "")
			return err
		}
		a.logPhase(operationID, &destination, "Import", "failure", "", 0, "Restore failed", "Error", "Failed", err.Error())
		return err
	}

	a.logPhaseWithFile(operationID, destination, "Import", "complete", "", 0, fmt.Sprintf("Restore completed in %s", time.Since(started).Round(time.Millisecond)), "Info", "Succeeded", "", record.FilePath, record.FileSizeBytes)
	if progress != nil {
		progress("Restore completed", record.FileSizeBytes, record.FileSizeBytes)
	}

	if err := a.runPostImportQueryPhase(ctx, operationID, destination); err != nil {
		return err
	}
	return nil
}

func (a *App) TestConnection(profile models.Profile) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := a.TestServerConnection(ctx, profile); err != nil {
		return err
	}
	return a.TestDatabaseConnection(ctx, profile)
}

type opLogger struct {
	app         *App
	operationID string
	profile     *models.Profile
}

func (a *App) newOpLogger(operationID string, profile *models.Profile) *opLogger {
	return &opLogger{app: a, operationID: operationID, profile: profile}
}

func (l *opLogger) Phase(action, phase, strategy string, attempt int, details, status, errStr string) {
	l.app.logPhase(l.operationID, l.profile, action, phase, strategy, attempt, details, "Info", status, errStr)
}

func (a *App) logPhase(operationID string, profile *models.Profile, action, phase, strategy string, attempt int, details, level, status, errStr string) {
	a.logPhaseWithFile(operationID, profileValue(profile), action, phase, strategy, attempt, details, level, status, errStr, "", 0)
}

func (a *App) logPhaseWithFile(operationID string, profile models.Profile, action, phase, strategy string, attempt int, details, level, status, errStr, filePath string, fileSize int64) {
	entry := models.LogEntry{
		ID:          newID(),
		OperationID: operationID,
		Timestamp:   time.Now(),
		Action:      action,
		Phase:       phase,
		Strategy:    strategy,
		Attempt:     attempt,
		Level:       level,
		Details:     details,
		FilePath:    filePath,
		Status:      status,
		Error:       errStr,
	}
	if fileSize > 0 {
		entry.FileSize = formatSize(fileSize)
	}
	if profile.ID != "" {
		entry.ProfileID = profile.ID
		entry.ProfileName = profile.Name
	}
	a.mu.Lock()
	a.logs = append(a.logs, entry)
	logs := append([]models.LogEntry(nil), a.logs...)
	a.mu.Unlock()
	_ = a.store.SaveLogs(logs)

	profileName := profile.Name
	debug.Log(level, action+"."+phase, status, details, profileName, operationID, errStr)
}

func profileValue(p *models.Profile) models.Profile {
	if p == nil {
		return models.Profile{}
	}
	return *p
}

func formatQueryResultSummary(result db.QueryResult) string {
	if result.Message != "" && len(result.Rows) == 0 {
		return result.Message
	}
	if len(result.Rows) == 0 {
		return "Query executed successfully"
	}
	if len(result.Columns) == 1 && result.Columns[0] == "Statements" && len(result.Rows) == 1 {
		return result.Message
	}
	return fmt.Sprintf("%d row(s) returned", len(result.Rows))
}

func newID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func safeName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "backup"
	}
	replacer := strings.NewReplacer(" ", "_", "/", "_", "\\", "_", ":", "_")
	return replacer.Replace(s)
}

func formatSize(size int64) string {
	return fmt.Sprintf("%.2f MB", float64(size)/1024/1024)
}

func percent(current, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(current) / float64(total) * 100
}
