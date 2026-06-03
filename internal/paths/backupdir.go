package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// DefaultBackupDestination returns the OS-specific default folder for backup files.
// Linux and other non-Windows systems use {Home}/dback/backups.
// Windows uses {Home}/Documents/dback.
func DefaultBackupDestination() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if runtime.GOOS == "windows" {
			return filepath.Join(home, "Documents", "dback")
		}
		return filepath.Join(home, "dback", "backups")
	}
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return filepath.Join(dir, "dback", "backups")
	}
	return "./backups"
}

// EffectiveBackupDestination returns the folder to use for backups.
// Empty or stale hardcoded defaults are replaced with DefaultBackupDestination().
func EffectiveBackupDestination(stored string) string {
	stored = strings.TrimSpace(stored)
	if stored == "" || isStaleDefaultBackupDestination(stored) {
		return DefaultBackupDestination()
	}
	return stored
}

// MigrateBackupDestination rewrites legacy hardcoded destinations saved in host profiles.
// The second return value is true when stored should be updated in the vault.
func MigrateBackupDestination(stored string) (string, bool) {
	stored = strings.TrimSpace(stored)
	if stored == "" {
		return "", false
	}
	if !isStaleDefaultBackupDestination(stored) {
		return stored, false
	}
	return DefaultBackupDestination(), true
}

func isStaleDefaultBackupDestination(dest string) bool {
	dest = filepath.Clean(dest)
	slash := strings.ToLower(filepath.ToSlash(dest))

	// Legacy developer hardcode from older builds.
	if strings.Contains(slash, "/home/mjavad/") {
		return true
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return false
	}
	home = filepath.Clean(home)

	if runtime.GOOS == "windows" {
		expected := filepath.Join(home, "Documents", "dback")
		if strings.EqualFold(dest, expected) {
			return false
		}
		suffix := filepath.Join("Documents", "dback")
		if strings.HasSuffix(strings.ToLower(dest), strings.ToLower(suffix)) {
			otherProfile := filepath.Dir(filepath.Dir(dest))
			if !strings.EqualFold(otherProfile, home) && strings.Contains(otherProfile, `\Users\`) {
				return true
			}
		}
		return false
	}

	expected := filepath.Join(home, "dback", "backups")
	if dest == expected {
		return false
	}
	suffix := filepath.Join("dback", "backups")
	if strings.HasSuffix(dest, suffix) {
		otherHome := filepath.Dir(filepath.Dir(dest))
		if otherHome != home && looksLikeUnixHomeDir(otherHome) {
			return true
		}
	}
	return false
}

func looksLikeUnixHomeDir(p string) bool {
	p = filepath.ToSlash(p)
	return strings.HasPrefix(p, "/home/") || p == "/root"
}
