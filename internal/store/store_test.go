package store

import (
	"os"
	"path/filepath"
	"testing"

	"dback/models"
)

func TestLoadProfilesMigratesLegacyShape(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")
	legacy := `{
  "profiles": [{
    "id": "p1",
    "name": "Production",
    "host": "10.0.0.1",
    "port": "22",
    "ssh_user": "root",
    "ssh_password": "secret",
    "auth_type": "Password",
    "db_host": "127.0.0.1",
    "db_port": "3306",
    "db_user": "db",
    "db_password": "db-secret",
    "db_type": "MySQL",
    "target_db_name": "app",
    "destination": "/tmp"
  }]
}`
	if err := os.WriteFile(path, []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}

	profiles, err := New(dir).LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 {
		t.Fatalf("expected one profile, got %d", len(profiles))
	}
	if profiles[0].ExportSettings == nil || profiles[0].ImportSettings == nil {
		t.Fatalf("expected migrated export/import settings: %#v", profiles[0])
	}
	if profiles[0].Group != "Default" {
		t.Fatalf("expected default group, got %q", profiles[0].Group)
	}
}

func TestExportProfilesStripsSecretsByDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.json")
	settings := models.TransferSettings{
		ConnectionType: models.ConnectionTypeSSH,
		SSHPassword:    "ssh-secret",
		DBPassword:     "db-secret",
		WPKey:          "wp-secret",
	}
	profiles := []models.Profile{{
		ID:             "p1",
		Name:           "Production",
		SSHPassword:    "ssh-secret",
		DBPassword:     "db-secret",
		WPKey:          "wp-secret",
		ExportSettings: &settings,
		ImportSettings: &settings,
	}}

	if err := New(dir).ExportProfiles(path, profiles, false); err != nil {
		t.Fatal(err)
	}
	imported, err := New(dir).ImportProfiles(path, true)
	if err != nil {
		t.Fatal(err)
	}
	got := imported[0]
	if got.SSHPassword != "" || got.DBPassword != "" || got.WPKey != "" {
		t.Fatalf("top-level secrets were not stripped: %#v", got)
	}
	if got.ExportSettings.SSHPassword != "" || got.ExportSettings.DBPassword != "" || got.ExportSettings.WPKey != "" {
		t.Fatalf("export settings secrets were not stripped: %#v", got.ExportSettings)
	}
	if got.ImportSettings.SSHPassword != "" || got.ImportSettings.DBPassword != "" || got.ImportSettings.WPKey != "" {
		t.Fatalf("import settings secrets were not stripped: %#v", got.ImportSettings)
	}
}
