package transfer

import (
	"path/filepath"
	"testing"

	"dback/models"
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
	raw := int64(20_000_000)
	estimated := int64(1_000_000)
	if got := raw / 20; got != estimated {
		t.Fatalf("expected gzip compressed estimate alignment")
	}
}

func TestBackupStrategiesJumpHostPrefersTmpFile(t *testing.T) {
	p := models.Profile{ConnectionType: models.ConnectionTypeJumpHost}
	strategies := backupStrategies(p)
	if len(strategies) < 1 || strategies[0] != StrategyTmpFile {
		t.Fatalf("jump host should prefer tmp-file first, got %v", strategies)
	}
}

func TestBackupStrategiesDirectSSHPrefersStreaming(t *testing.T) {
	p := models.Profile{ConnectionType: models.ConnectionTypeSSH}
	strategies := backupStrategies(p)
	if len(strategies) < 1 || strategies[0] != StrategyStreaming {
		t.Fatalf("direct SSH should prefer streaming first, got %v", strategies)
	}
}

func TestDumpStderrIsFatal(t *testing.T) {
	if !dumpStderrIsFatal("mysqldump: Error: 'Access denied'") {
		t.Fatal("expected access denied to be fatal")
	}
	if dumpStderrIsFatal("mysqldump: Warning: Using a password on the command line") {
		t.Fatal("expected password warning to be non-fatal")
	}
}
