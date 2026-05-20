package store

import (
	"os"
	"path/filepath"
	"testing"

	"dback/models"
)

func TestLoadProfilesMigratesLegacyFlatShape(t *testing.T) {
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
	if profiles[0].Host != "10.0.0.1" {
		t.Fatalf("expected migrated host field: %#v", profiles[0])
	}
	if profiles[0].ExportSettings != nil || profiles[0].ImportSettings != nil {
		t.Fatalf("flattened profile should not keep nested settings: %#v", profiles[0])
	}
	if profiles[0].Group != "Default" {
		t.Fatalf("expected default group, got %q", profiles[0].Group)
	}
}

func TestFlattenProfileSplitsDifferentImportSettings(t *testing.T) {
	export := models.TransferSettings{
		ConnectionType: models.ConnectionTypeSSH,
		Host:           "source.example.com",
		Port:           "22",
		DBType:         models.DBTypeMySQL,
		TargetDBName:   "source_db",
	}
	importSettings := models.TransferSettings{
		ConnectionType: models.ConnectionTypeSSH,
		Host:           "dest.example.com",
		Port:           "2222",
		DBType:         models.DBTypeMariaDB,
		TargetDBName:   "dest_db",
	}
	profile := models.Profile{
		ID:             "p1",
		Name:           "Production",
		ExportSettings: &export,
		ImportSettings: &importSettings,
	}

	out := flattenProfile(profile)
	if len(out) != 2 {
		t.Fatalf("expected two hosts, got %d: %#v", len(out), out)
	}
	if out[0].Host != "source.example.com" {
		t.Fatalf("unexpected export host: %#v", out[0])
	}
	if out[1].Host != "dest.example.com" || out[1].Name != "Production (import)" {
		t.Fatalf("unexpected import host: %#v", out[1])
	}
}

func TestExportProfilesStripsSecretsByDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.json")
	profiles := []models.Profile{{
		ID:          "p1",
		Name:        "Production",
		SSHPassword: "ssh-secret",
		DBPassword:  "db-secret",
		WPKey:       "wp-secret",
	}}

	if err := New(dir).ExportProfiles(path, profiles, false, ""); err != nil {
		t.Fatal(err)
	}
	imported, err := New(dir).ImportProfilesBundle(path, true, "")
	if err != nil {
		t.Fatal(err)
	}
	got := imported[0]
	if got.SSHPassword != "" || got.DBPassword != "" || got.WPKey != "" {
		t.Fatalf("secrets were not stripped: %#v", got)
	}
}

func TestExportProfilesEncryptedRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.json")
	profiles := []models.Profile{{
		ID:          "p1",
		Name:        "Production",
		SSHPassword: "ssh-secret",
		DBPassword:  "db-secret",
	}}

	s := New(dir)
	if err := s.ExportProfiles(path, profiles, true, "master-pass"); err != nil {
		t.Fatal(err)
	}
	imported, err := s.ImportProfilesBundle(path, true, "master-pass")
	if err != nil {
		t.Fatal(err)
	}
	if imported[0].SSHPassword != "ssh-secret" || imported[0].DBPassword != "db-secret" {
		t.Fatalf("encrypted secrets not restored: %#v", imported[0])
	}
}

func TestLoadTemplatesSeedsDefaults(t *testing.T) {
	dir := t.TempDir()
	templates, err := New(dir).LoadTemplates()
	if err != nil {
		t.Fatal(err)
	}
	if len(templates) < 2 {
		t.Fatalf("expected seed templates, got %#v", templates)
	}
}

func TestDetectProfileConflicts(t *testing.T) {
	existing := []models.Profile{
		{ID: "a", Name: "Alpha"},
		{ID: "b", Name: "Beta"},
	}
	imported := []models.Profile{
		{ID: "a", Name: "Alpha-new"},
		{ID: "c", Name: "Beta"},
	}
	conflicts := DetectProfileConflicts(existing, imported)
	if len(conflicts) != 2 {
		t.Fatalf("expected 2 conflicts, got %d: %#v", len(conflicts), conflicts)
	}
}

func TestMergeProfilesByIDAndName(t *testing.T) {
	existing := []models.Profile{{ID: "a", Name: "Alpha", Host: "old"}}
	imported := []models.Profile{
		{ID: "a", Name: "Alpha", Host: "new"},
		{ID: "b", Name: "Beta", Host: "beta"},
	}
	out := MergeProfiles(existing, imported)
	if len(out) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(out))
	}
	if out[0].Host != "new" {
		t.Fatalf("expected ID merge to update host, got %#v", out[0])
	}
}
