package app

import (
	"errors"
	"path/filepath"
	"testing"

	"dback/internal/store"
	"dback/models"
)

const testMasterKey = "test-master-key"

func openApp(t *testing.T, dir string) *App {
	t.Helper()
	a, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if a.HasVault() {
		if err := a.Unlock(testMasterKey); err != nil {
			t.Fatal(err)
		}
	} else {
		if err := a.CreateVault(testMasterKey); err != nil && !errors.Is(err, store.ErrVaultExists) {
			t.Fatal(err)
		}
	}
	if !a.IsUnlocked() {
		if err := a.Unlock(testMasterKey); err != nil {
			t.Fatal(err)
		}
	}
	return a
}

func TestSaveProfilePersistsUnifiedHostSettings(t *testing.T) {
	dir := t.TempDir()
	a := openApp(t, dir)

	profile := models.Profile{
		ID:             "p1",
		Name:           "Production",
		Group:          "Gold",
		ConnectionType: models.ConnectionTypeSSH,
		Host:           "source.example.com",
		Port:           "22",
		AuthType:       models.AuthTypePassword,
		DBType:         models.DBTypeMySQL,
		TargetDBName:   "source_db",
		Destination:    dir,
		PreImportQuery: "SELECT 1;",
	}
	if err := a.SaveProfile(profile); err != nil {
		t.Fatal(err)
	}

	reloaded := openApp(t, dir)
	profiles := reloaded.Profiles()
	if len(profiles) != 1 {
		t.Fatalf("expected one profile, got %d", len(profiles))
	}
	got := profiles[0]
	if got.Host != "source.example.com" {
		t.Fatalf("host not persisted: %#v", got)
	}
	if got.PreImportQuery != "SELECT 1;" {
		t.Fatalf("pre-import query not persisted: %#v", got)
	}
	if got.ExportSettings != nil || got.ImportSettings != nil {
		t.Fatalf("legacy nested settings should not be saved: %#v", got)
	}
}

func TestProfileTransferRoundTrip(t *testing.T) {
	dir := t.TempDir()
	a := openApp(t, dir)
	if err := a.SaveProfile(models.Profile{ID: "p1", Name: "Production", Group: "Gold"}); err != nil {
		t.Fatal(err)
	}

	bundle := filepath.Join(dir, "profiles.json")
	if err := a.ExportProfiles(bundle, false, ""); err != nil {
		t.Fatal(err)
	}

	target := openApp(t, t.TempDir())
	if err := target.ImportProfiles(bundle, false, ""); err != nil {
		t.Fatal(err)
	}
	profiles := target.Profiles()
	if len(profiles) != 1 || profiles[0].Name != "Production" {
		t.Fatalf("unexpected imported profiles: %#v", profiles)
	}
}

func TestProfileTransferEncryptedRoundTrip(t *testing.T) {
	dir := t.TempDir()
	a := openApp(t, dir)
	profile := models.Profile{
		ID:          "p1",
		Name:        "Secure",
		SSHPassword: "ssh-secret",
		DBPassword:  "db-secret",
	}
	if err := a.SaveProfile(profile); err != nil {
		t.Fatal(err)
	}

	bundle := filepath.Join(dir, "encrypted.json")
	if err := a.ExportProfiles(bundle, true, "master-pass"); err != nil {
		t.Fatal(err)
	}

	target := openApp(t, t.TempDir())
	if err := target.ImportProfiles(bundle, true, "master-pass"); err != nil {
		t.Fatal(err)
	}
	profiles := target.Profiles()
	if len(profiles) != 1 {
		t.Fatalf("expected one profile, got %d", len(profiles))
	}
	if profiles[0].SSHPassword != "ssh-secret" || profiles[0].DBPassword != "db-secret" {
		t.Fatalf("encrypted secrets not restored: %#v", profiles[0])
	}
}

func TestProfileImportMergesByID(t *testing.T) {
	dir := t.TempDir()
	a := openApp(t, dir)
	if err := a.SaveProfile(models.Profile{ID: "p1", Name: "Local", Group: "Default"}); err != nil {
		t.Fatal(err)
	}

	bundle := filepath.Join(dir, "bundle.json")
	if err := a.ExportProfiles(bundle, false, ""); err != nil {
		t.Fatal(err)
	}

	target := openApp(t, t.TempDir())
	if err := target.SaveProfile(models.Profile{ID: "p2", Name: "Existing"}); err != nil {
		t.Fatal(err)
	}
	if err := target.ImportProfiles(bundle, false, ""); err != nil {
		t.Fatal(err)
	}
	profiles := target.Profiles()
	if len(profiles) != 2 {
		t.Fatalf("expected merge to keep existing host, got %#v", profiles)
	}
}

func TestPreviewImportProfilesDetectsConflicts(t *testing.T) {
	dir := t.TempDir()
	a := openApp(t, dir)
	if err := a.SaveProfile(models.Profile{ID: "p1", Name: "Local"}); err != nil {
		t.Fatal(err)
	}
	bundle := filepath.Join(dir, "bundle.json")
	if err := a.ExportProfiles(bundle, false, ""); err != nil {
		t.Fatal(err)
	}

	target := openApp(t, t.TempDir())
	if err := target.SaveProfile(models.Profile{ID: "p1", Name: "Existing"}); err != nil {
		t.Fatal(err)
	}
	_, conflicts, err := target.PreviewImportProfiles(bundle, false, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 1 || conflicts[0].Reason != "id" {
		t.Fatalf("expected id conflict, got %#v", conflicts)
	}
}
