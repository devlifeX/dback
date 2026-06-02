package update

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func installWindows(newExePath string, progress ProgressFunc) error {
	targetExe, err := os.Executable()
	if err != nil {
		return err
	}
	targetExe, err = filepath.Abs(targetExe)
	if err != nil {
		return err
	}

	scriptDir := filepath.Join(os.TempDir(), "dback-update")
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		return err
	}

	scriptPath := filepath.Join(scriptDir, "apply-update.bat")
	script := fmt.Sprintf(`@echo off
setlocal
set PID=%d
set SRC=%s
set DST=%s
:wait
tasklist /FI "PID eq %%PID%%" 2>NUL | find /I "%%PID%%" >NUL
if not errorlevel 1 (
  timeout /t 1 /nobreak >NUL
  goto wait
)
copy /Y "%%SRC%%" "%%DST%%"
if errorlevel 1 exit /b 1
start "" "%%DST%%"
del "%%~f0"
`, os.Getpid(), quoteWindowsPath(newExePath), quoteWindowsPath(targetExe))

	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		return err
	}

	if progress != nil {
		progress("Applying update and restarting…")
	}

	cmd := exec.Command("cmd", "/C", "start", "", scriptPath)
	cmd.Dir = scriptDir
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start update helper: %w", err)
	}

	// Give the helper a moment to start before exiting.
	time.Sleep(300 * time.Millisecond)
	os.Exit(0)
	return nil
}

func quoteWindowsPath(path string) string {
	return `"` + path + `"`
}
