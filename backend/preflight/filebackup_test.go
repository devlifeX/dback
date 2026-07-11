package preflight

import (
	"strings"
	"testing"

	"dback/models"
)

func TestValidateFileBackupOutputRequiresTarAndZstd(t *testing.T) {
	out := strings.Join([]string{
		"===OS===",
		"Linux",
		"===TOOLS===",
		"tar (GNU tar) 1.34",
		"zstd 1.5.5",
		"===RESULT===",
		"fail=0",
	}, "\n")
	if err := ValidateFileBackupOutput(out, models.ArchiveCompressionZstd); err != nil {
		t.Fatal(err)
	}
}

func TestValidateFileBackupOutputMissingTar(t *testing.T) {
	out := strings.Join([]string{
		"===OS===",
		"Linux",
		"===TOOLS===",
		"no-tar",
		"zstd 1.5.5",
	}, "\n")
	if err := ValidateFileBackupOutput(out, models.ArchiveCompressionZstd); err == nil {
		t.Fatal("expected missing tar error")
	}
}

func TestValidateFileBackupOutputRequiresGzip(t *testing.T) {
	out := strings.Join([]string{
		"===OS===",
		"Linux",
		"===TOOLS===",
		"tar (GNU tar) 1.34",
		"no-gzip",
	}, "\n")
	if err := ValidateFileBackupOutput(out, models.ArchiveCompressionGzip); err == nil {
		t.Fatal("expected missing gzip error")
	}
}
