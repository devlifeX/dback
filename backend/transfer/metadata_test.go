package transfer

import (
	"os"
	"testing"
)

func TestValidateLocalFileSize(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/dump.sql.gz"
	if err := os.WriteFile(path, []byte("hello world backup content"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateLocalFile(path, 26, ""); err != nil {
		t.Fatalf("expected valid file, got %v", err)
	}
	if err := validateLocalFile(path, 999, ""); err == nil {
		t.Fatal("expected size mismatch")
	}
}

func TestLocalResumeOffset(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/partial"
	if err := os.WriteFile(path, make([]byte, 100), 0600); err != nil {
		t.Fatal(err)
	}
	meta := FileMeta{Offset: 50}
	if got := localResumeOffset(path, meta, 200); got != 100 {
		t.Fatalf("expected resume at 100, got %d", got)
	}
}

func TestSafeName(t *testing.T) {
	if got := safeName(" prod/db:1 "); got != "prod_db_1" {
		t.Fatalf("unexpected safe name: %q", got)
	}
}
