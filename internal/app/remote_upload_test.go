package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"dback/internal/store"
	"dback/models"
)

func TestPendingUploadsNotDoneForAllActiveDestinations(t *testing.T) {
	dir := t.TempDir()
	a, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.CreateVault("master-key-12345678"); err != nil {
		t.Fatal(err)
	}

	dest1 := models.RemoteDestination{
		ID:   "d1",
		Name: "One",
		Type: models.RemoteProviderS3,
		S3: &models.S3DestinationConfig{
			Endpoint: "s3.local", Bucket: "b1", AccessKeyID: "k", SecretKey: "s",
		},
	}
	dest2 := models.RemoteDestination{
		ID:   "d2",
		Name: "Two",
		Type: models.RemoteProviderS3,
		S3: &models.S3DestinationConfig{
			Endpoint: "s3.local", Bucket: "b2", AccessKeyID: "k", SecretKey: "s",
		},
	}
	if err := a.SaveRemoteDestination(dest1); err != nil {
		t.Fatal(err)
	}
	if err := a.SaveRemoteDestination(dest2); err != nil {
		t.Fatal(err)
	}

	profile := models.Profile{
		ID:                         "p1",
		Name:                       "Host",
		Group:                      "Default",
		RemoteUploadDestinationIDs: []string{"d1", "d2"},
	}
	if err := a.SaveProfile(profile); err != nil {
		t.Fatal(err)
	}

	filePath := filepath.Join(dir, "backup.sql.gz")
	if err := os.WriteFile(filePath, []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	record := models.ExportRecord{
		ID:          "r1",
		ProfileID:   "p1",
		ProfileName: "Host",
		FilePath:    filePath,
		ExportDate:  time.Now(),
		RemoteUploads: []models.RemoteUploadState{
			{DestinationID: "d1", Status: models.RemoteUploadDone},
		},
	}
	a.mu.Lock()
	a.history = []models.ExportRecord{record}
	a.mu.Unlock()
	if err := a.store.SaveHistory([]models.ExportRecord{record}); err != nil {
		t.Fatal(err)
	}

	count, details, err := a.PendingUploads("p1")
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected 1 pending record, got %d", count)
	}
	if len(details[0].MissingFor) != 1 || details[0].MissingFor[0] != "d2" {
		t.Fatalf("expected missing d2, got %#v", details[0].MissingFor)
	}
}

func TestPendingUploadsNewDestination(t *testing.T) {
	dir := t.TempDir()
	a, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.CreateVault("master-key-12345678"); err != nil {
		t.Fatal(err)
	}

	dest1 := models.RemoteDestination{ID: "d1", Name: "One", Type: models.RemoteProviderS3, S3: &models.S3DestinationConfig{Endpoint: "s3.local", Bucket: "b1", AccessKeyID: "k", SecretKey: "s"}}
	if err := a.SaveRemoteDestination(dest1); err != nil {
		t.Fatal(err)
	}

	profile := models.Profile{ID: "p1", Name: "Host", Group: "Default", RemoteUploadDestinationIDs: []string{"d1"}}
	if err := a.SaveProfile(profile); err != nil {
		t.Fatal(err)
	}

	filePath := filepath.Join(dir, "backup.sql.gz")
	if err := os.WriteFile(filePath, []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	record := models.ExportRecord{
		ID: "r1", ProfileID: "p1", ProfileName: "Host", FilePath: filePath, ExportDate: time.Now(),
		RemoteUploads: []models.RemoteUploadState{{DestinationID: "d1", Status: models.RemoteUploadDone}},
	}
	a.mu.Lock()
	a.history = []models.ExportRecord{record}
	a.mu.Unlock()
	if err := a.store.SaveHistory([]models.ExportRecord{record}); err != nil {
		t.Fatal(err)
	}

	count, _, err := a.PendingUploads("p1")
	if err != nil || count != 0 {
		t.Fatalf("expected no pending before adding destination, got %d err=%v", count, err)
	}

	profile.RemoteUploadDestinationIDs = []string{"d1", "d2"}
	if err := a.SaveProfile(profile); err != nil {
		t.Fatal(err)
	}
	dest2 := models.RemoteDestination{ID: "d2", Name: "Two", Type: models.RemoteProviderS3, S3: &models.S3DestinationConfig{Endpoint: "s3.local", Bucket: "b2", AccessKeyID: "k", SecretKey: "s"}}
	if err := a.SaveRemoteDestination(dest2); err != nil {
		t.Fatal(err)
	}

	count, _, err = a.PendingUploads("p1")
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected pending after adding new destination, got %d", count)
	}
}

func TestRemoteUploadLock(t *testing.T) {
	dir := t.TempDir()
	a, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.CreateVault("master-key-12345678"); err != nil {
		t.Fatal(err)
	}

	dest := models.RemoteDestination{ID: "d1", Name: "One", Type: models.RemoteProviderS3, S3: &models.S3DestinationConfig{Endpoint: "s3.local", Bucket: "b1", AccessKeyID: "k", SecretKey: "s"}}
	if err := a.SaveRemoteDestination(dest); err != nil {
		t.Fatal(err)
	}
	profile := models.Profile{ID: "p1", Name: "Host", Group: "Default", RemoteUploadDestinationIDs: []string{"d1"}}
	if err := a.SaveProfile(profile); err != nil {
		t.Fatal(err)
	}

	if !profileUploadLocks.tryAcquire("p1") {
		t.Fatal("expected first acquire to succeed")
	}
	err = a.UploadProfileBackups(t.Context(), "p1", nil, nil)
	if err != ErrRemoteUploadRunning {
		t.Fatalf("expected ErrRemoteUploadRunning, got %v", err)
	}
	profileUploadLocks.release("p1")
}

func TestRemoteDestinationsMigrationOnce(t *testing.T) {
	dir := t.TempDir()
	s := store.New(dir)
	if err := s.CreateVault("master-key-12345678"); err != nil {
		t.Fatal(err)
	}

	settings := models.SyncSettings{
		Endpoint: "minio.local:9000", Bucket: "dback-sync", AccessKeyID: "access", SecretKey: "secret",
	}
	if err := s.SaveSyncSettings(settings); err != nil {
		t.Fatal(err)
	}
	s.Lock()

	s2 := store.New(dir)
	if err := s2.Unlock("master-key-12345678"); err != nil {
		t.Fatal(err)
	}
	dests, err := s2.LoadRemoteDestinations()
	if err != nil {
		t.Fatal(err)
	}
	if len(dests) != 1 || dests[0].Name != "Default" {
		t.Fatalf("expected migrated destination, got %#v", dests)
	}
	appID, err := s2.AppSettingsDestinationID()
	if err != nil || appID == "" {
		t.Fatalf("expected app settings destination id, got %q err=%v", appID, err)
	}
	if err := s2.DeleteRemoteDestination(dests[0].ID, true); err != nil {
		t.Fatal(err)
	}

	s3 := store.New(dir)
	if err := s3.Unlock("master-key-12345678"); err != nil {
		t.Fatal(err)
	}
	dests, err = s3.LoadRemoteDestinations()
	if err != nil {
		t.Fatal(err)
	}
	if len(dests) != 0 {
		t.Fatalf("expected no remigration after delete, got %#v", dests)
	}
}

func TestDeleteRemoteDestinationForceReloadsProfileRefs(t *testing.T) {
	dir := t.TempDir()
	a, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.CreateVault("master-key-12345678"); err != nil {
		t.Fatal(err)
	}
	dest := models.RemoteDestination{
		ID:   "d1",
		Name: "One",
		Type: models.RemoteProviderS3,
		S3:   &models.S3DestinationConfig{Endpoint: "s3.local", Bucket: "b1", AccessKeyID: "k", SecretKey: "s"},
	}
	if err := a.SaveRemoteDestination(dest); err != nil {
		t.Fatal(err)
	}
	if err := a.SetAppSettingsDestinationID("d1"); err != nil {
		t.Fatal(err)
	}
	if err := a.SaveProfile(models.Profile{
		ID:                         "p1",
		Name:                       "Host",
		Group:                      "Default",
		RemoteUploadDestinationIDs: []string{"d1"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.DeleteRemoteDestination("d1", true); err != nil {
		t.Fatal(err)
	}
	profiles := a.Profiles()
	if len(profiles) != 1 {
		t.Fatalf("expected one profile, got %d", len(profiles))
	}
	if len(profiles[0].RemoteUploadDestinationIDs) != 0 {
		t.Fatalf("expected profile refs cleared after force delete, got %#v", profiles[0].RemoteUploadDestinationIDs)
	}
	if _, err := a.appSettingsDestination(); err != store.ErrSyncNotConfigured {
		t.Fatalf("expected app settings sync to require destination after delete, got %v", err)
	}
}
