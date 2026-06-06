package verify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestQuickCheck_Match(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backup.sql.gz")
	if err := os.WriteFile(path, []byte("test backup payload"), 0600); err != nil {
		t.Fatal(err)
	}
	sum, err := ChecksumFile(path)
	if err != nil {
		t.Fatal(err)
	}
	result, err := QuickCheck(path, sum)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed {
		t.Fatalf("expected pass, got %#v", result)
	}
}

func TestQuickCheck_Mismatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backup.sql.gz")
	if err := os.WriteFile(path, []byte("payload"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := QuickCheck(path, "deadbeef")
	if err != nil {
		t.Fatal(err)
	}
	if result.Passed {
		t.Fatal("expected mismatch")
	}
}

func TestQuickCheck_MissingChecksum(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backup.sql.gz")
	if err := os.WriteFile(path, []byte("payload"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := QuickCheck(path, "")
	if err == nil {
		t.Fatal("expected error for missing checksum")
	}
}
