package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dback/models"
)

const testMasterKey = "test-master-key"

func unlockStore(t *testing.T, s *Store) {
	t.Helper()
	if s.HasVault() {
		if err := s.Unlock(testMasterKey); err != nil {
			t.Fatal(err)
		}
		return
	}
	if err := s.CreateVault(testMasterKey); err != nil {
		t.Fatal(err)
	}
}

func TestCreateVaultAndUnlock(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if s.HasVault() {
		t.Fatal("expected no vault initially")
	}
	if err := s.CreateVault(testMasterKey); err != nil {
		t.Fatal(err)
	}
	if !s.HasVault() || !s.IsUnlocked() {
		t.Fatal("vault should exist and be unlocked")
	}

	profiles := []models.Profile{{ID: "p1", Name: "Secure", SSHPassword: "secret"}}
	if err := s.SaveProfiles(profiles); err != nil {
		t.Fatal(err)
	}

	s2 := New(dir)
	if s2.IsUnlocked() {
		t.Fatal("new store instance should start locked")
	}
	if err := s2.Unlock(testMasterKey); err != nil {
		t.Fatal(err)
	}
	got, err := s2.LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].SSHPassword != "secret" {
		t.Fatalf("expected decrypted profile secret, got %#v", got)
	}
	raw, err := os.ReadFile(s.VaultPath())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "secret") {
		t.Fatal("vault file must not contain plaintext secrets")
	}
}

func TestUnlockWrongMasterKey(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if err := s.CreateVault(testMasterKey); err != nil {
		t.Fatal(err)
	}
	s2 := New(dir)
	if err := s2.Unlock("wrong-key"); err != ErrWrongMasterKey {
		t.Fatalf("expected wrong master key, got %v", err)
	}
}

func TestLegacyPlaintextMigration(t *testing.T) {
	dir := t.TempDir()
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
	if err := os.WriteFile(filepath.Join(dir, "profiles.json"), []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}

	s := New(dir)
	if !s.HasLegacyPlaintext() {
		t.Fatal("expected legacy plaintext")
	}
	if err := s.Unlock(testMasterKey); err != nil {
		t.Fatal(err)
	}
	profiles, err := s.LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 || profiles[0].Host != "10.0.0.1" {
		t.Fatalf("migration failed: %#v", profiles)
	}
	if _, err := os.Stat(filepath.Join(dir, "profiles.json")); !os.IsNotExist(err) {
		t.Fatal("legacy profiles.json should be archived")
	}
	if _, err := os.Stat(filepath.Join(dir, "profiles.json.legacy")); err != nil {
		t.Fatal("expected profiles.json.legacy archive")
	}
}

func TestLoadWhileLockedFails(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if _, err := s.LoadProfiles(); err != ErrVaultLocked {
		t.Fatalf("expected locked error, got %v", err)
	}
}

func TestVaultPersistsTemplatesHistoryLogs(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	unlockStore(t, s)

	now := models.SQLTemplate{ID: "t1", Name: "Reset", Body: "SELECT 1"}
	if err := s.SaveTemplates([]models.SQLTemplate{now}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveHistory([]models.ExportRecord{{ID: "h1", ProfileID: "p1"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveLogs([]models.LogEntry{{ID: "l1", ProfileID: "p1", Details: "ok"}}); err != nil {
		t.Fatal(err)
	}

	s2 := New(dir)
	if err := s2.Unlock(testMasterKey); err != nil {
		t.Fatal(err)
	}
	templates, err := s2.LoadTemplates()
	if err != nil || len(templates) != 1 {
		t.Fatalf("templates not persisted: %#v err=%v", templates, err)
	}
	history, err := s2.LoadHistory()
	if err != nil || len(history) != 1 {
		t.Fatalf("history not persisted: %#v err=%v", history, err)
	}
	logs, err := s2.LoadLogs()
	if err != nil || len(logs) != 1 {
		t.Fatalf("logs not persisted: %#v err=%v", logs, err)
	}
}
