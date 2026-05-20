package store

import (
	"os"
	"path/filepath"
	"testing"

	"dback/models"
)

func TestExportProfilesIncludeSecretsRequiresPassphrase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.json")
	profiles := []models.Profile{{ID: "p1", Name: "Production", SSHPassword: "secret"}}
	s := New(dir)
	if err := s.ExportProfiles(path, profiles, true, ""); err != ErrIncludeSecretsNoPassphrase {
		t.Fatalf("expected ErrIncludeSecretsNoPassphrase, got %v", err)
	}
}

func TestExportAppDataIncludeSecretsRequiresPassphrase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.json")
	data := AppImportData{
		Profiles: []models.Profile{{ID: "p1", Name: "Production", SSHPassword: "secret"}},
	}
	s := New(dir)
	if err := s.ExportAppData(path, data, true, ""); err != ErrIncludeSecretsNoPassphrase {
		t.Fatalf("expected ErrIncludeSecretsNoPassphrase, got %v", err)
	}
}

func TestUnlockRemovesLegacyPlaintextWhenVaultExists(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if err := s.CreateVault(testMasterKey); err != nil {
		t.Fatal(err)
	}
	legacy := `{"profiles":[{"id":"p1","name":"Legacy","host":"10.0.0.1"}]}`
	legacyPath := filepath.Join(dir, "profiles.json")
	if err := os.WriteFile(legacyPath, []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}

	s2 := New(dir)
	if err := s2.Unlock(testMasterKey); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatal("legacy plaintext should be removed when vault exists")
	}
}

func TestStoreLockClearsMemory(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	unlockStore(t, s)
	if err := s.SaveProfiles([]models.Profile{{ID: "p1", Name: "Secure", SSHPassword: "secret"}}); err != nil {
		t.Fatal(err)
	}
	s.Lock()
	if s.IsUnlocked() {
		t.Fatal("store should be locked")
	}
	if _, err := s.LoadProfiles(); err != ErrVaultLocked {
		t.Fatalf("expected locked error after Lock(), got %v", err)
	}
}
