package app

import (
	"path/filepath"
	"testing"

	"dback/models"
)

func TestSaveProfilePersistsIndependentTransferSettings(t *testing.T) {
	dir := t.TempDir()
	a, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	exportSettings := models.TransferSettings{
		ConnectionType: models.ConnectionTypeSSH,
		Host:           "source.example.com",
		Port:           "22",
		AuthType:       models.AuthTypePassword,
		DBType:         models.DBTypeMySQL,
		TargetDBName:   "source_db",
		Destination:    dir,
	}
	importSettings := models.TransferSettings{
		ConnectionType: models.ConnectionTypeSSH,
		Host:           "dest.example.com",
		Port:           "2222",
		AuthType:       models.AuthTypeKeyFile,
		DBType:         models.DBTypePostgreSQL,
		TargetDBName:   "dest_db",
	}
	profile := models.Profile{
		ID:             "p1",
		Name:           "Production",
		Group:          "Gold",
		ExportSettings: &exportSettings,
		ImportSettings: &importSettings,
	}
	if err := a.SaveProfile(profile); err != nil {
		t.Fatal(err)
	}

	reloaded, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	profiles := reloaded.Profiles()
	if len(profiles) != 1 {
		t.Fatalf("expected one profile, got %d", len(profiles))
	}
	if profiles[0].EffectiveExport().Host != "source.example.com" {
		t.Fatalf("export settings not persisted: %#v", profiles[0].EffectiveExport())
	}
	if profiles[0].EffectiveImport().Host != "dest.example.com" {
		t.Fatalf("import settings not persisted: %#v", profiles[0].EffectiveImport())
	}
}

func TestProfileTransferRoundTrip(t *testing.T) {
	dir := t.TempDir()
	a, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.SaveProfile(models.Profile{ID: "p1", Name: "Production", Group: "Gold"}); err != nil {
		t.Fatal(err)
	}

	bundle := filepath.Join(dir, "profiles.json")
	if err := a.ExportProfiles(bundle, false); err != nil {
		t.Fatal(err)
	}

	target, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := target.ImportProfiles(bundle, false); err != nil {
		t.Fatal(err)
	}
	profiles := target.Profiles()
	if len(profiles) != 1 || profiles[0].Name != "Production" {
		t.Fatalf("unexpected imported profiles: %#v", profiles)
	}
}
