package store

import (
	"testing"

	"dback/models"
)

func TestMigrateRemoteDestinationsOnce(t *testing.T) {
	payload := models.AppVaultPayload{
		Sync: &models.SyncSettings{
			Endpoint: "minio.local:9000", Bucket: "bucket", AccessKeyID: "key", SecretKey: "secret",
		},
	}
	if !migrateRemoteDestinations(&payload) {
		t.Fatal("expected migration to run")
	}
	if len(payload.RemoteDestinations) != 1 {
		t.Fatalf("expected one destination, got %#v", payload.RemoteDestinations)
	}
	if payload.AppSettingsDestinationID == "" {
		t.Fatal("expected app settings destination id")
	}
	if !payload.RemoteDestinationsMigrated {
		t.Fatal("expected migrated flag")
	}
	if migrateRemoteDestinations(&payload) {
		t.Fatal("expected no second migration")
	}
}

func TestMigrateRemoteDestinationsEmptySync(t *testing.T) {
	payload := models.AppVaultPayload{}
	if migrateRemoteDestinations(&payload) {
		t.Fatal("expected no migration without sync settings")
	}
	if !payload.RemoteDestinationsMigrated {
		t.Fatal("expected migrated flag set")
	}
}

func TestSaveRemoteDestinationRejectsDuplicateNameOnUpdate(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if err := s.CreateVault("master-key-12345678"); err != nil {
		t.Fatal(err)
	}
	first := models.RemoteDestination{
		ID:   "d1",
		Name: "First",
		Type: models.RemoteProviderS3,
		S3:   &models.S3DestinationConfig{Endpoint: "s3.local", Bucket: "b1", AccessKeyID: "k", SecretKey: "s"},
	}
	second := models.RemoteDestination{
		ID:   "d2",
		Name: "Second",
		Type: models.RemoteProviderS3,
		S3:   &models.S3DestinationConfig{Endpoint: "s3.local", Bucket: "b2", AccessKeyID: "k", SecretKey: "s"},
	}
	if err := s.SaveRemoteDestination(first); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveRemoteDestination(second); err != nil {
		t.Fatal(err)
	}
	first.Name = "Second"
	if err := s.SaveRemoteDestination(first); err == nil {
		t.Fatal("expected duplicate name error when updating destination")
	}
}

func TestDeleteRemoteDestinationClearsLegacySync(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if err := s.CreateVault("master-key-12345678"); err != nil {
		t.Fatal(err)
	}
	dest := models.RemoteDestination{
		ID:   "d1",
		Name: "Sync",
		Type: models.RemoteProviderS3,
		S3:   &models.S3DestinationConfig{Endpoint: "s3.local", Bucket: "bucket", AccessKeyID: "k", SecretKey: "s"},
	}
	if err := s.SaveRemoteDestination(dest); err != nil {
		t.Fatal(err)
	}
	if err := s.SetAppSettingsDestinationID("d1"); err != nil {
		t.Fatal(err)
	}
	if cfg, err := s.LoadSyncSettings(); err != nil || cfg == nil {
		t.Fatalf("expected legacy sync mirror before delete, cfg=%#v err=%v", cfg, err)
	}
	if err := s.DeleteRemoteDestination("d1", true); err != nil {
		t.Fatal(err)
	}
	if cfg, err := s.LoadSyncSettings(); err != nil || cfg != nil {
		t.Fatalf("expected legacy sync cleared after delete, cfg=%#v err=%v", cfg, err)
	}
}
