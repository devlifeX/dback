package builder

import (
	"strings"
	"testing"

	"dback/backend/shell"
	"dback/models"
)

func TestBuildTarArchivePlan(t *testing.T) {
	plan, err := BuildTarArchivePlan("/var/www/html", []string{"*.log"}, models.ArchiveCompressionZstd, shell.ModeRemoteToLocalPipe)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Mode != shell.ModeRemoteToLocalPipe {
		t.Fatalf("unexpected mode %v", plan.Mode)
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(plan.Steps))
	}
	tar := plan.Steps[0]
	if tar.Binary != "tar" {
		t.Fatalf("expected tar, got %q", tar.Binary)
	}
	foundExclude := false
	for i, arg := range tar.Args {
		if arg == "--exclude" && i+1 < len(tar.Args) && tar.Args[i+1] == "*.log" {
			foundExclude = true
		}
	}
	if !foundExclude {
		t.Fatalf("exclude not in args: %v", tar.Args)
	}
	if plan.Steps[1].Binary != "zstd" {
		t.Fatalf("expected zstd compressor")
	}
}

func TestBuildTarArchivePlanRejectsRelativePath(t *testing.T) {
	_, err := BuildTarArchivePlan("relative/path", nil, models.ArchiveCompressionZstd, shell.ModeLocalPipe)
	if err == nil {
		t.Fatal("expected error for relative path")
	}
}

func TestBuildTarArchivePlanRejectsUnsafeExclude(t *testing.T) {
	_, err := BuildTarArchivePlan("/var/www/html", []string{"; rm -rf /"}, models.ArchiveCompressionZstd, shell.ModeLocalPipe)
	if err == nil {
		t.Fatal("expected unsafe exclude rejection")
	}
}

func TestArchiveExtension(t *testing.T) {
	ext, err := ArchiveExtension(models.ArchiveCompressionZstd)
	if err != nil {
		t.Fatal(err)
	}
	if ext != ".tar.zst" {
		t.Fatalf("got %q", ext)
	}
}

func TestBuildTarArchivePlanNoPipelineInArgs(t *testing.T) {
	plan, err := BuildTarArchivePlan("/var/www/html", nil, models.ArchiveCompressionGzip, shell.ModeLocalPipe)
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range plan.Steps {
		joined := strings.Join(step.Args, " ")
		if strings.Contains(joined, "|") {
			t.Fatal("pipeline char in args")
		}
	}
}
