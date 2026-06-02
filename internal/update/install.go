package update

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type ProgressFunc func(stage string)

type CommandRunner func(ctx context.Context, name string, args ...string) *exec.Cmd

func defaultCommandRunner(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

func Apply(ctx context.Context, info Info, progress ProgressFunc, runner CommandRunner) error {
	if !info.Available {
		return fmt.Errorf("no update available")
	}
	if info.Asset.URL == "" {
		return fmt.Errorf("update asset is missing")
	}
	if runner == nil {
		runner = defaultCommandRunner
	}

	destDir := filepath.Join(os.TempDir(), "dback-update")
	path, err := Download(ctx, nil, userAgent(info.CurrentVersion), info.Asset, destDir, progress)
	if err != nil {
		return err
	}

	switch runtime.GOOS {
	case "linux":
		return installDeb(ctx, path, progress, runner)
	case "windows":
		return installWindows(path, progress)
	default:
		return fmt.Errorf("install is not supported on %s", runtime.GOOS)
	}
}

func userAgent(currentVersion string) string {
	v := NormalizeVersion(currentVersion)
	return "DBack/" + v
}

func installDeb(ctx context.Context, debPath string, progress ProgressFunc, runner CommandRunner) error {
	if progress != nil {
		progress("Installing package (admin password may be required)…")
	}
	cmd := runner(ctx, "pkexec", "env", "DEBIAN_FRONTEND=noninteractive", "apt-get", "install", "-y", debPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := string(output)
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("package install failed: %s", truncate(msg, 500))
	}
	return nil
}
