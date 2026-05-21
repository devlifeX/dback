package transfer

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestValidateBackupIntegrityValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.sql.gz")
	out, err := exec.Command("bash", "-c", "echo 'SELECT 1;' | gzip -1").Output()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, out, 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateBackupIntegrity(path); err != nil {
		t.Fatalf("expected valid gzip: %v", err)
	}
}

func TestValidateBackupIntegrityTruncated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.sql.gz")
	out, err := exec.Command("bash", "-c", "echo 'SELECT 1;' | gzip -1").Output()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, out[:len(out)/2], 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateBackupIntegrity(path); err == nil {
		t.Fatal("expected truncated gzip to fail validation")
	}
}
