package sqlstore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dback/internal/config"
	"dback/models"
)

func TestVaultFileMigratesToSQLite(t *testing.T) {
	dir := t.TempDir()
	// Simulate legacy vault by creating via old-format isn't needed; use legacy plaintext.
	legacy := `{"profiles":[{"id":"p1","name":"Host","host":"10.0.0.1","port":"22","ssh_user":"root","ssh_password":"secret","auth_type":"Password","db_host":"127.0.0.1","db_port":"3306","db_user":"db","db_password":"dbpass","db_type":"MySQL","target_db_name":"app","destination":"/tmp"}]}`
	if err := os.WriteFile(filepath.Join(dir, "profiles.json"), []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := OpenStore(dir, config.LoadDBConfig(dir))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Unlock("test-master-key"); err != nil {
		t.Fatal(err)
	}
	profiles, err := s.LoadProfiles()
	if err != nil || len(profiles) != 1 || profiles[0].SSHPassword != "secret" {
		t.Fatalf("profiles=%#v err=%v", profiles, err)
	}
	dbPath := filepath.Join(dir, "dback.db")
	raw, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "secret") {
		t.Fatal("sqlite file must not store plaintext secrets")
	}
	if _, err := os.Stat(filepath.Join(dir, "profiles.json")); !os.IsNotExist(err) {
		t.Fatal("legacy file should be removed")
	}
}

func TestMySQLDriverOpen(t *testing.T) {
	if os.Getenv("DBACK_TEST_MYSQL_DSN") == "" {
		t.Skip("set DBACK_TEST_MYSQL_DSN to run MySQL parity test")
	}
	dir := t.TempDir()
	cfg := config.DBConfig{Driver: config.DBDriverMySQL, DSN: os.Getenv("DBACK_TEST_MYSQL_DSN")}
	s, err := OpenStore(dir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CreateVault("test-master-key"); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveProfiles([]models.Profile{{ID: "p1", Name: "MySQL", SSHPassword: "pw"}}); err != nil {
		t.Fatal(err)
	}
	s2, err := OpenStore(dir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := s2.Unlock("test-master-key"); err != nil {
		t.Fatal(err)
	}
	got, err := s2.LoadProfiles()
	if err != nil || len(got) != 1 || got[0].SSHPassword != "pw" {
		t.Fatalf("got=%#v err=%v", got, err)
	}
}

func TestInitialMigrationAvoidsMySQLTextDefaults(t *testing.T) {
	body, err := migrationFS.ReadFile("migrations/001_initial.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToUpper(string(body))
	if strings.Contains(sql, "TEXT NOT NULL DEFAULT") || strings.Contains(sql, "LONGTEXT NOT NULL DEFAULT") {
		t.Fatal("MySQL does not allow portable defaults on TEXT/LONGTEXT columns")
	}
}

func TestOpenStoreRejectsUnsupportedDriver(t *testing.T) {
	if _, err := OpenStore(t.TempDir(), config.DBConfig{Driver: "postgres"}); err == nil {
		t.Fatal("expected unsupported driver error")
	}
}
