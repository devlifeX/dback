package models

import "testing"

func TestFileBackupPathNormalize(t *testing.T) {
	p := FileBackupPath{Name: "Uploads", RemotePath: "/var/www/uploads"}
	if err := p.Normalize(); err != nil {
		t.Fatal(err)
	}
	if p.CanonicalKey != "uploads" {
		t.Fatalf("expected uploads, got %q", p.CanonicalKey)
	}
}

func TestFileBackupPathNameRequired(t *testing.T) {
	p := FileBackupPath{RemotePath: "/var/www/html"}
	if err := p.Normalize(); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestValidateFileBackupPathsDuplicateKey(t *testing.T) {
	paths := []FileBackupPath{
		{ID: "1", Name: "Uploads", RemotePath: "/var/a"},
		{ID: "2", Name: "uploads", RemotePath: "/var/b"},
	}
	if err := ValidateFileBackupPaths(paths); err == nil {
		t.Fatal("expected duplicate canonical key error")
	}
}

func TestSafeLabelTruncation(t *testing.T) {
	long := "this_is_a_very_long_name_that_should_be_truncated_for_filesystem_use"
	got := SafeLabel(long)
	if len(got) > 48 {
		t.Fatalf("label too long: %q", got)
	}
}

func TestExportRecordSourceDisplay(t *testing.T) {
	rec := ExportRecord{
		ExportType:  ExportTypeFiles,
		SourceLabel: "Website",
	}
	if rec.SourceDisplay() != "Website" {
		t.Fatalf("got %q", rec.SourceDisplay())
	}
}

func TestExportRecordEffectiveExportTypeLegacy(t *testing.T) {
	rec := ExportRecord{}
	if rec.EffectiveExportType() != ExportTypeDatabase {
		t.Fatalf("expected database default")
	}
}
