package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Platform abstracts OS-specific behavior so Android can plug in later.
type Platform interface {
	AppDataDir() string
	OpenFolder(path string) error
	OpenURL(url string) error
	IsMobile() bool
}

// DesktopPlatform is the default desktop implementation.
type DesktopPlatform struct{}

func (DesktopPlatform) IsMobile() bool { return false }

func (DesktopPlatform) AppDataDir() string {
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return filepath.Join(dir, "dback")
	}
	return fallbackDataDir()
}

func (DesktopPlatform) OpenFolder(path string) error {
	if path == "" {
		return nil
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("explorer", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}

func (DesktopPlatform) OpenURL(url string) error {
	if strings.TrimSpace(url) == "" {
		return nil
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

func fallbackDataDir() string {
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		return cwd
	}
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		if !strings.Contains(dir, "go-build") && !strings.Contains(dir, "/tmp/") {
			return dir
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return "."
}

func defaultBackupDir(platform Platform) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, "dback", "backups")
	}
	return filepath.Join(platform.AppDataDir(), "backups")
}
