package models

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ExportType classifies export history entries.
type ExportType string

const (
	ExportTypeDatabase ExportType = "database"
	ExportTypeFiles    ExportType = "files"
)

// VerificationMethod describes how a backup was verified.
type VerificationMethod string

const (
	VerifyNone     VerificationMethod = "none"
	VerifySHA256   VerificationMethod = "sha256"
	VerifyMetadata VerificationMethod = "metadata"
	VerifyQuick    VerificationMethod = "quick"
	VerifyDeep     VerificationMethod = "deep"
)

// ArchiveCompression selects archive compression for file backups.
type ArchiveCompression string

const (
	ArchiveCompressionZstd ArchiveCompression = "zstd"
	ArchiveCompressionGzip ArchiveCompression = "gzip"
)

// FileBackupPath is one configured remote path to archive.
type FileBackupPath struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	RemotePath   string `json:"remote_path"`
	CanonicalKey string `json:"canonical_key,omitempty"`
}

var slugSanitizer = regexp.MustCompile(`[^a-z0-9]+`)

// SlugCanonical converts a display name to a stable grouping key.
func SlugCanonical(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = slugSanitizer.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if s == "" {
		return "path"
	}
	return s
}

// SafeLabel makes a filesystem-safe label from a slug or name fragment.
func SafeLabel(label string) string {
	s := SlugCanonical(label)
	if len(s) <= 48 {
		return s
	}
	hash := fmt.Sprintf("%x", hashString(s))[:6]
	return s[:41] + "_" + hash
}

func hashString(s string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}

// Normalize validates and derives CanonicalKey.
func (p *FileBackupPath) Normalize() error {
	p.Name = strings.TrimSpace(p.Name)
	p.RemotePath = strings.TrimSpace(p.RemotePath)
	if p.Name == "" {
		return errors.New("file backup path name is required")
	}
	if p.RemotePath == "" {
		return errors.New("file backup remote path is required")
	}
	if !strings.HasPrefix(p.RemotePath, "/") {
		return fmt.Errorf("remote path must be absolute: %q", p.RemotePath)
	}
	p.CanonicalKey = SlugCanonical(p.Name)
	return nil
}

// ValidateExcludePattern rejects unsafe tar exclude patterns.
func ValidateExcludePattern(pattern string) error {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil
	}
	for _, r := range pattern {
		if r == ';' || r == '`' || r == '$' {
			return fmt.Errorf("invalid exclude pattern %q", pattern)
		}
	}
	return nil
}

// NormalizeArchiveCompression returns default zstd when empty.
func NormalizeArchiveCompression(c ArchiveCompression) ArchiveCompression {
	switch c {
	case ArchiveCompressionGzip:
		return ArchiveCompressionGzip
	default:
		return ArchiveCompressionZstd
	}
}

// ValidateFileBackupPaths validates all paths on a profile.
func ValidateFileBackupPaths(paths []FileBackupPath) error {
	keys := map[string]struct{}{}
	pathsSeen := map[string]struct{}{}
	for i := range paths {
		if err := paths[i].Normalize(); err != nil {
			return err
		}
		if paths[i].ID == "" {
			return fmt.Errorf("file backup path %q is missing id", paths[i].Name)
		}
		if _, ok := keys[paths[i].CanonicalKey]; ok {
			return fmt.Errorf("duplicate file backup name %q", paths[i].Name)
		}
		keys[paths[i].CanonicalKey] = struct{}{}
		if _, ok := pathsSeen[paths[i].RemotePath]; ok {
			return fmt.Errorf("duplicate remote path %q", paths[i].RemotePath)
		}
		pathsSeen[paths[i].RemotePath] = struct{}{}
	}
	return nil
}

// EffectiveExportType returns database for legacy records.
func (r ExportRecord) EffectiveExportType() ExportType {
	if r.ExportType == "" {
		return ExportTypeDatabase
	}
	return r.ExportType
}

// SourceDisplay returns the label shown in backup lists.
func (r ExportRecord) SourceDisplay() string {
	if r.SourceLabel != "" {
		return r.SourceLabel
	}
	if r.EffectiveExportType() == ExportTypeFiles {
		if r.SourcePath != "" {
			return r.SourcePath
		}
		return "files"
	}
	if r.DatabaseName != "" {
		return r.DatabaseName
	}
	return "database"
}

// SupportsImport returns true when restore/import applies.
func (r ExportRecord) SupportsImport() bool {
	return r.EffectiveExportType() == ExportTypeDatabase
}

func (p Profile) SupportsFileBackup() bool {
	return !p.UsesWordPress()
}

func (p Profile) FileBackupReady() bool {
	if !p.FileBackupEnabled || !p.SupportsFileBackup() {
		return false
	}
	return len(p.FileBackupPaths) > 0
}

// EffectiveFileBackupDestination resolves the folder for file archives.
func (p Profile) EffectiveFileBackupDestination(defaultDest string) string {
	if strings.TrimSpace(p.FileBackupDestination) != "" {
		return strings.TrimSpace(p.FileBackupDestination)
	}
	if strings.TrimSpace(p.Destination) != "" {
		return strings.TrimSpace(p.Destination)
	}
	return defaultDest
}
