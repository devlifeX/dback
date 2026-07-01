package app

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"sync"
	"time"

	"dback/internal/remote"
	"dback/internal/store"
	syncpkg "dback/internal/sync"
	"dback/models"
)

var (
	ErrRemoteUploadRunning       = errors.New("remote upload already running for this host")
	ErrRemoteUploadNotConfigured = errors.New("no remote upload destinations configured for this host")
)

type RemoteUploadProgress struct {
	ProfileID     string
	RecordID      string
	DestinationID string
	RemoteKey     string
	Status        models.RemoteUploadStatus
	Error         string
}

type RemoteUploadProgressFunc func(RemoteUploadProgress)

type PendingUploadDetail struct {
	RecordID   string
	FilePath   string
	ProfileID  string
	MissingFor []string
}

type remoteUploadLocks struct {
	mu    sync.Mutex
	byKey map[string]struct{}
}

func (l *remoteUploadLocks) tryAcquire(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.byKey == nil {
		l.byKey = map[string]struct{}{}
	}
	if _, ok := l.byKey[key]; ok {
		return false
	}
	l.byKey[key] = struct{}{}
	return true
}

func (l *remoteUploadLocks) release(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.byKey, key)
}

var profileUploadLocks remoteUploadLocks

func (a *App) ListRemoteDestinations() ([]models.RemoteDestination, error) {
	return a.store.LoadRemoteDestinations()
}

func (a *App) SaveRemoteDestination(dest models.RemoteDestination) error {
	if dest.ID == "" {
		dest.ID = newID()
	}
	if dest.S3 != nil {
		dest.S3.Endpoint = syncpkg.NormalizeEndpoint(dest.S3.Endpoint)
	}
	return a.store.SaveRemoteDestination(dest)
}

func (a *App) DeleteRemoteDestination(id string, force bool) error {
	if err := a.store.DeleteRemoteDestination(id, force); err != nil {
		return err
	}
	return a.Reload()
}

func (a *App) DestinationUsage(id string) (store.DestinationUsage, error) {
	return a.store.DestinationUsage(id)
}

func (a *App) AppSettingsDestinationID() (string, error) {
	return a.store.AppSettingsDestinationID()
}

func (a *App) SetAppSettingsDestinationID(id string) error {
	return a.store.SetAppSettingsDestinationID(id)
}

func (a *App) RemoteDestinationByID(id string) (models.RemoteDestination, error) {
	return a.store.RemoteDestinationByID(id)
}

func (a *App) TestRemoteDestination(ctx context.Context, dest models.RemoteDestination) error {
	if dest.S3 != nil {
		dest.S3.Endpoint = syncpkg.NormalizeEndpoint(dest.S3.Endpoint)
	}
	return syncpkg.TestDestinationConnection(ctx, dest)
}

func (a *App) PendingUploads(profileID string) (int, []PendingUploadDetail, error) {
	profile, err := a.profileByID(profileID)
	if err != nil {
		return 0, nil, err
	}
	destinations, err := a.activeUploadDestinations(profile)
	if err != nil {
		return 0, nil, err
	}
	if len(destinations) == 0 {
		return 0, nil, nil
	}
	history := a.History()
	var details []PendingUploadDetail
	for _, rec := range history {
		if rec.ProfileID != profileID {
			continue
		}
		missing := pendingDestinationsForRecord(rec, destinations)
		if len(missing) == 0 {
			continue
		}
		details = append(details, PendingUploadDetail{
			RecordID:   rec.ID,
			FilePath:   rec.FilePath,
			ProfileID:  profileID,
			MissingFor: missing,
		})
	}
	return len(details), details, nil
}

func pendingDestinationsForRecord(rec models.ExportRecord, destinationIDs []string) []string {
	if rec.FilePath == "" {
		return nil
	}
	if _, err := os.Stat(rec.FilePath); err != nil {
		return nil
	}
	var missing []string
	for _, destID := range destinationIDs {
		if !models.RemoteUploadDoneForDestination(rec.RemoteUploads, destID) {
			missing = append(missing, destID)
		}
	}
	return missing
}

func (a *App) profileByID(id string) (models.Profile, error) {
	for _, p := range a.Profiles() {
		if p.ID == id {
			return p, nil
		}
	}
	return models.Profile{}, fmt.Errorf("profile not found")
}

func (a *App) activeUploadDestinations(profile models.Profile) ([]string, error) {
	ids := profile.RemoteUploadDestinationIDs
	if len(ids) == 0 {
		return nil, nil
	}
	all, err := a.store.LoadRemoteDestinations()
	if err != nil {
		return nil, err
	}
	exists := map[string]bool{}
	for _, d := range all {
		exists[d.ID] = true
	}
	var active []string
	for _, id := range ids {
		if exists[id] {
			active = append(active, id)
		}
	}
	return active, nil
}

func backupRemoteKey(profileID, recordID, filePath string) string {
	base := filepath.Base(filePath)
	return fmt.Sprintf("dback/backups/%s/%s/%s", profileID, recordID, base)
}

func (a *App) UploadProfileBackups(ctx context.Context, profileID string, recordIDs []string, progress RemoteUploadProgressFunc) error {
	lockKey := profileID
	if !profileUploadLocks.tryAcquire(lockKey) {
		return ErrRemoteUploadRunning
	}
	defer profileUploadLocks.release(lockKey)

	profile, err := a.profileByID(profileID)
	if err != nil {
		return err
	}
	destIDs, err := a.activeUploadDestinations(profile)
	if err != nil {
		return err
	}
	if len(destIDs) == 0 {
		return ErrRemoteUploadNotConfigured
	}

	records, err := a.recordsForUpload(profileID, recordIDs)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}

	destByID := map[string]models.RemoteDestination{}
	allDests, err := a.store.LoadRemoteDestinations()
	if err != nil {
		return err
	}
	for _, d := range allDests {
		destByID[d.ID] = d
	}

	var failures []error
	for _, rec := range records {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		missing := pendingDestinationsForRecord(rec, destIDs)
		if len(missing) == 0 {
			continue
		}
		for _, destID := range missing {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			dest, ok := destByID[destID]
			if !ok {
				continue
			}
			if err := a.uploadRecordToDestination(ctx, profileID, &rec, dest, progress); err != nil {
				failures = append(failures, fmt.Errorf("%s: %w", destID, err))
				if progress != nil {
					progress(RemoteUploadProgress{
						ProfileID:     profileID,
						RecordID:      rec.ID,
						DestinationID: destID,
						Status:        models.RemoteUploadFailed,
						Error:         err.Error(),
					})
				}
			}
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("remote upload completed with %d failure(s): %w", len(failures), errors.Join(failures...))
	}
	return nil
}

func (a *App) recordsForUpload(profileID string, recordIDs []string) ([]models.ExportRecord, error) {
	history := a.History()
	if len(recordIDs) == 0 {
		var out []models.ExportRecord
		for _, rec := range history {
			if rec.ProfileID == profileID {
				out = append(out, rec)
			}
		}
		return out, nil
	}
	byID := map[string]models.ExportRecord{}
	for _, rec := range history {
		byID[rec.ID] = rec
	}
	var out []models.ExportRecord
	for _, id := range recordIDs {
		rec, ok := byID[id]
		if !ok || rec.ProfileID != profileID {
			return nil, fmt.Errorf("backup record not found: %s", id)
		}
		out = append(out, rec)
	}
	return out, nil
}

func (a *App) uploadRecordToDestination(ctx context.Context, profileID string, rec *models.ExportRecord, dest models.RemoteDestination, progress RemoteUploadProgressFunc) error {
	if rec.FilePath == "" {
		return fmt.Errorf("backup file path is empty")
	}
	info, err := os.Stat(rec.FilePath)
	if err != nil {
		return fmt.Errorf("local backup file missing: %w", err)
	}

	remoteKey := backupRemoteKey(profileID, rec.ID, rec.FilePath)
	uploading := models.RemoteUploadState{
		DestinationID: dest.ID,
		Status:        models.RemoteUploadUploading,
		RemoteKey:     remoteKey,
	}
	rec.RemoteUploads = upsertRemoteUploadState(rec.RemoteUploads, uploading)
	if err := a.UpdateHistoryRecord(*rec); err != nil {
		return err
	}
	if progress != nil {
		progress(RemoteUploadProgress{
			ProfileID:     profileID,
			RecordID:      rec.ID,
			DestinationID: dest.ID,
			RemoteKey:     remoteKey,
			Status:        models.RemoteUploadUploading,
		})
	}

	provider, err := remote.NewProvider(dest)
	if err != nil {
		a.markUploadFailed(rec, dest.ID, remoteKey, err)
		return err
	}

	putCtx, cancel := context.WithTimeout(ctx, remote.PutObjectTimeout)
	defer cancel()

	f, err := os.Open(rec.FilePath)
	if err != nil {
		a.markUploadFailed(rec, dest.ID, remoteKey, err)
		return err
	}
	defer f.Close()

	contentType := mime.TypeByExtension(filepath.Ext(rec.FilePath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	etag, err := provider.PutObject(putCtx, remoteKey, f, info.Size(), contentType)
	if err != nil {
		a.markUploadFailed(rec, dest.ID, remoteKey, err)
		return err
	}

	done := models.RemoteUploadState{
		DestinationID: dest.ID,
		Status:        models.RemoteUploadDone,
		RemoteKey:     remoteKey,
		UploadedAt:    time.Now(),
		ETag:          etag,
		SizeBytes:     info.Size(),
	}
	rec.RemoteUploads = upsertRemoteUploadState(rec.RemoteUploads, done)
	if err := a.UpdateHistoryRecord(*rec); err != nil {
		return err
	}
	if progress != nil {
		progress(RemoteUploadProgress{
			ProfileID:     profileID,
			RecordID:      rec.ID,
			DestinationID: dest.ID,
			RemoteKey:     remoteKey,
			Status:        models.RemoteUploadDone,
		})
	}
	return nil
}

func (a *App) markUploadFailed(rec *models.ExportRecord, destID, remoteKey string, err error) {
	state := models.RemoteUploadState{
		DestinationID: destID,
		Status:        models.RemoteUploadFailed,
		RemoteKey:     remoteKey,
		Error:         err.Error(),
	}
	rec.RemoteUploads = upsertRemoteUploadState(rec.RemoteUploads, state)
	_ = a.UpdateHistoryRecord(*rec)
}

func upsertRemoteUploadState(states []models.RemoteUploadState, update models.RemoteUploadState) []models.RemoteUploadState {
	for i, s := range states {
		if s.DestinationID == update.DestinationID {
			states[i] = update
			return states
		}
	}
	return append(states, update)
}

func (a *App) MaybeAutoUploadAfterBackup(ctx context.Context, profile models.Profile, records []models.ExportRecord) error {
	if len(profile.RemoteUploadDestinationIDs) == 0 {
		return nil
	}
	uploadDB := profile.RemoteAutoUploadDB
	uploadFiles := profile.RemoteAutoUploadFiles
	if !uploadDB && !uploadFiles {
		return nil
	}
	var ids []string
	for _, rec := range records {
		exportType := rec.ExportType
		if exportType == "" {
			exportType = models.ExportTypeDatabase
		}
		if exportType == models.ExportTypeDatabase && uploadDB {
			ids = append(ids, rec.ID)
		}
		if exportType == models.ExportTypeFiles && uploadFiles {
			ids = append(ids, rec.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	return a.UploadProfileBackups(ctx, profile.ID, ids, nil)
}
