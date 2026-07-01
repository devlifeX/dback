package builder

import (
	"fmt"
	"path/filepath"
	"strings"

	"dback/backend/archiver"
	"dback/backend/shell"
	"dback/models"
)

// BuildTarArchivePlan builds a tar+compress pipeline for a remote/local absolute path.
func BuildTarArchivePlan(rootPath string, excludes []string, compression models.ArchiveCompression, mode shell.ExecutionMode) (shell.ExecutionPlan, error) {
	rootPath = strings.TrimSpace(rootPath)
	if rootPath == "" {
		return shell.ExecutionPlan{}, fmt.Errorf("root path is required")
	}
	if !strings.HasPrefix(rootPath, "/") {
		return shell.ExecutionPlan{}, fmt.Errorf("root path must be absolute: %q", rootPath)
	}
	a, err := archiver.For(compression)
	if err != nil {
		return shell.ExecutionPlan{}, err
	}

	dir := filepath.Dir(rootPath)
	base := filepath.Base(rootPath)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return shell.ExecutionPlan{}, fmt.Errorf("invalid root path: %q", rootPath)
	}

	args := []string{"-cf", "-", "-C", dir}
	for _, ex := range excludes {
		ex = strings.TrimSpace(ex)
		if ex == "" {
			continue
		}
		if err := models.ValidateExcludePattern(ex); err != nil {
			return shell.ExecutionPlan{}, err
		}
		args = append(args, "--exclude", ex)
	}
	args = append(args, "--", base)

	return shell.ExecutionPlan{
		Mode: mode,
		Steps: []shell.Command{
			{Binary: "tar", Args: args},
			a.CompressCommand(),
		},
	}, nil
}

// ArchiveExtension returns the file extension for a compression setting.
func ArchiveExtension(compression models.ArchiveCompression) (string, error) {
	a, err := archiver.For(compression)
	if err != nil {
		return "", err
	}
	return a.Extension(), nil
}
