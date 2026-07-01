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

func TestCountUploadWork(t *testing.T) {
	destIDs := []string{"d1", "d2"}
	filePath := filepath.Join(t.TempDir(), "backup.sql.gz")
	if err := os.WriteFile(filePath, []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}

	records := []models.ExportRecord{
		{
			ID:       "r1",
			FilePath: filePath,
			RemoteUploads: []models.RemoteUploadState{
				{DestinationID: "d1", Status: models.RemoteUploadDone},
			},
		},
		{
			ID:       "r2",
			FilePath: filePath,
		},
	}

	totalWork, pendingRecords := countUploadWork(records, destIDs)
	if totalWork != 3 {
		t.Fatalf("expected totalWork=3, got %d", totalWork)
	}
	if pendingRecords != 2 {
		t.Fatalf("expected pendingRecords=2, got %d", pendingRecords)
	}
}

func TestComputeUploadResult(t *testing.T) {
	destIDs := []string{"d1", "d2"}
	filePath := filepath.Join(t.TempDir(), "backup.sql.gz")
	if err := os.WriteFile(filePath, []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}

	records := []models.ExportRecord{
		{
			ID:       "r1",
			FilePath: filePath,
			RemoteUploads: []models.RemoteUploadState{
				{DestinationID: "d1", Status: models.RemoteUploadDone},
				{DestinationID: "d2", Status: models.RemoteUploadDone},
			},
		},
		{
			ID:       "r2",
			FilePath: filePath,
			RemoteUploads: []models.RemoteUploadState{
				{DestinationID: "d1", Status: models.RemoteUploadDone},
				{DestinationID: "d2", Status: models.RemoteUploadFailed},
			},
		},
		{ID: "r3", FilePath: filePath},
	}

	hadWork := map[string]bool{"r2": true, "r3": true}
	result := computeUploadResult(records, destIDs, hadWork)
	if result.SkippedRecords != 1 {
		t.Fatalf("expected 1 skipped record, got %d", result.SkippedRecords)
	}
	if result.UploadedRecords != 0 {
		t.Fatalf("expected 0 uploaded records, got %d", result.UploadedRecords)
	}
	if result.FailedRecords != 2 {
		t.Fatalf("expected 2 failed records, got %d", result.FailedRecords)
	}
}

func TestUploadProfileBackupsInitialProgress(t *testing.T) {
	dir := t.TempDir()
	a, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.CreateVault("master-key-12345678"); err != nil {
		t.Fatal(err)
	}

	dest := models.RemoteDestination{
		ID: "d1", Name: "One", Type: models.RemoteProviderS3,
		S3: &models.S3DestinationConfig{Endpoint: "s3.local", Bucket: "b1", AccessKeyID: "k", SecretKey: "s"},
	}
	if err := a.SaveRemoteDestination(dest); err != nil {
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
	}
	a.mu.Lock()
	a.history = []models.ExportRecord{record}
	a.mu.Unlock()
	if err := a.store.SaveHistory([]models.ExportRecord{record}); err != nil {
		t.Fatal(err)
	}

	var callbacks []RemoteUploadProgress
	result, err := a.UploadProfileBackups(t.Context(), "p1", nil, func(p RemoteUploadProgress) {
		callbacks = append(callbacks, p)
	})
	if len(callbacks) == 0 {
		t.Fatal("expected at least one progress callback")
	}
	first := callbacks[0]
	if first.Current != 0 || first.Total != 1 {
		t.Fatalf("expected initial progress 0/1, got %d/%d", first.Current, first.Total)
	}
	if first.RecordTotal != 1 {
		t.Fatalf("expected record total 1, got %d", first.RecordTotal)
	}
	if result.FailedRecords != 1 {
		t.Fatalf("expected 1 failed record after upload error, got uploaded=%d failed=%d skipped=%d err=%v",
			result.UploadedRecords, result.FailedRecords, result.SkippedRecords, err)
	}
	if err == nil {
		t.Fatal("expected upload error against fake S3 endpoint")
	}
}

func TestFormatRemoteUploadResultMessage(t *testing.T) {
	cases := []struct {
		result  RemoteUploadResult
		message string
	}{
		{RemoteUploadResult{UploadedRecords: 4}, "4 backups uploaded"},
		{RemoteUploadResult{UploadedRecords: 1}, "1 backup uploaded"},
		{RemoteUploadResult{UploadedRecords: 3, FailedRecords: 1}, "3 backups uploaded, 1 failed"},
		{RemoteUploadResult{SkippedRecords: 2}, "All backups already uploaded"},
	}
	for _, tc := range cases {
		if got := FormatRemoteUploadResultMessage(tc.result); got != tc.message {
			t.Fatalf("FormatRemoteUploadResultMessage(%#v) = %q, want %q", tc.result, got, tc.message)
		}
	}
}

func TestComputeUploadResultUsesFreshHistory(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "backup.sql.gz")
	if err := os.WriteFile(filePath, []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	destIDs := []string{"d1"}
	stale := models.ExportRecord{
		ID:       "r1",
		FilePath: filePath,
		RemoteUploads: []models.RemoteUploadState{
			{DestinationID: "d1", Status: models.RemoteUploadFailed, Error: "network error"},
		},
	}
	fresh := models.ExportRecord{
		ID:       "r1",
		FilePath: filePath,
		RemoteUploads: []models.RemoteUploadState{
			{DestinationID: "d1", Status: models.RemoteUploadDone},
		},
	}
	hadWork := map[string]bool{"r1": true}
	staleResult := computeUploadResult([]models.ExportRecord{stale}, destIDs, hadWork)
	if staleResult.FailedRecords != 1 || staleResult.UploadedRecords != 0 {
		t.Fatalf("stale snapshot should count failed, got %#v", staleResult)
	}
	freshResult := computeUploadResult([]models.ExportRecord{fresh}, destIDs, hadWork)
	if freshResult.UploadedRecords != 1 || freshResult.FailedRecords != 0 {
		t.Fatalf("fresh history should count uploaded, got %#v", freshResult)
	}
	if FormatRemoteUploadResultMessage(freshResult) != "1 backup uploaded" {
		t.Fatalf("unexpected message: %q", FormatRemoteUploadResultMessage(freshResult))
	}
}

func TestRecordsNeedRemoteVerify(t *testing.T) {
	rec := models.ExportRecord{
		RemoteUploads: []models.RemoteUploadState{
			{DestinationID: "d1", Status: models.RemoteUploadFailed},
		},
	}
	if recordsNeedRemoteVerify([]models.ExportRecord{rec}, []string{"d1"}) {
		t.Fatal("failed uploads should not trigger remote verify")
	}
	rec.RemoteUploads = []models.RemoteUploadState{
		{DestinationID: "d1", Status: models.RemoteUploadDone},
	}
	if !recordsNeedRemoteVerify([]models.ExportRecord{rec}, []string{"d1"}) {
		t.Fatal("done uploads should trigger remote verify")
	}
}

func TestResetInterruptedUploads(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "backup.sql.gz")
	if err := os.WriteFile(filePath, []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	rec := models.ExportRecord{
		ID:       "r1",
		FilePath: filePath,
		RemoteUploads: []models.RemoteUploadState{
			{DestinationID: "d1", Status: models.RemoteUploadUploading},
		},
	}
	if !resetInterruptedUploads(&rec) {
		t.Fatal("expected interrupted upload to be reset")
	}
	if rec.RemoteUploads[0].Status != models.RemoteUploadFailed {
		t.Fatalf("expected failed status, got %s", rec.RemoteUploads[0].Status)
	}
	missing := pendingDestinationsForRecord(rec, []string{"d1"})
	if len(missing) != 1 || missing[0] != "d1" {
		t.Fatalf("expected pending destination d1, got %#v", missing)
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
	_, err = a.UploadProfileBackups(t.Context(), "p1", nil, nil)
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
