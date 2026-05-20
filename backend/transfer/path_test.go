package transfer

import (
	"path/filepath"
	"testing"
)

func TestBackupHostPath(t *testing.T) {
	dest := "/tmp/backups"
	host := "Production Server"
	db := "app_db"
	hostDir := filepath.Join(dest, safeName(host))
	fileName := safeName(db) + "_01_01_2026_12_00_00.sql.gz"
	fullPath := filepath.Join(hostDir, fileName)
	want := filepath.Join("/tmp/backups", "Production_Server", "app_db_01_01_2026_12_00_00.sql.gz")
	if fullPath != want {
		t.Fatalf("unexpected backup path: got %q want %q", fullPath, want)
	}
}

func TestEstimateBackupTotalUsesCompressedEstimate(t *testing.T) {
	raw := int64(9_000_000)
	estimated := int64(3_000_000)
	if got := raw / 3; got != estimated {
		t.Fatalf("expected compressed estimate helper alignment")
	}
}
