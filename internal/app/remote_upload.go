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
	Current       int
	Total         int
	RecordIndex   int
	RecordTotal   int
}

type RemoteUploadProgressFunc func(RemoteUploadProgress)

type RemoteUploadResult struct {
	UploadedRecords int
	FailedRecords   int
	SkippedRecords  int
}

type RemoteUploadPlan struct {
	RetryRecordIDs       []string
	StaleRemoteRecordIDs []string
	LatestStaleRecordID  string
}

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
		state, ok := models.RemoteUploadStateForDestination(rec.RemoteUploads, destID)
		if ok && state.Status == models.RemoteUploadDone {
			continue
		}
		missing = append(missing, destID)
	}
	return missing
}

func resetInterruptedUploads(rec *models.ExportRecord) bool {
	changed := false
	for i := range rec.RemoteUploads {
		if rec.RemoteUploads[i].Status == models.RemoteUploadUploading {
			rec.RemoteUploads[i].Status = models.RemoteUploadFailed
			rec.RemoteUploads[i].Error = "upload interrupted"
			changed = true
		}
	}
	return changed
}

func removeRemoteUploadState(states []models.RemoteUploadState, destID string) []models.RemoteUploadState {
	var out []models.RemoteUploadState
	for _, s := range states {
		if s.DestinationID != destID {
			out = append(out, s)
		}
	}
	return out
}

func (a *App) remoteObjectExists(ctx context.Context, provider remote.Provider, key string) (bool, error) {
	checkCtx, cancel := context.WithTimeout(ctx, remote.StatObjectTimeout)
	defer cancel()
	return provider.ObjectExists(checkCtx, key)
}

func recordsNeedRemoteVerify(records []models.ExportRecord, destIDs []string) bool {
	for _, rec := range records {
		for _, destID := range destIDs {
			state, ok := models.RemoteUploadStateForDestination(rec.RemoteUploads, destID)
			if ok && state.Status == models.RemoteUploadDone {
				return true
			}
		}
	}
	return false
}

func providersForDestinations(destByID map[string]models.RemoteDestination, destIDs []string) (map[string]remote.Provider, error) {
	providers := make(map[string]remote.Provider, len(destIDs))
	for _, destID := range destIDs {
		dest, ok := destByID[destID]
		if !ok {
			continue
		}
		if _, ok := providers[destID]; ok {
			continue
		}
		provider, err := remote.NewProvider(dest)
		if err != nil {
			return nil, err
		}
		providers[destID] = provider
	}
	return providers, nil
}

func (a *App) reconcileRecordsBeforeUpload(ctx context.Context, profileID string, records []models.ExportRecord, destIDs []string, destByID map[string]models.RemoteDestination) (map[string]bool, error) {
	profileName := profileID
	if profile, err := a.profileByID(profileID); err == nil {
		profileName = profile.Name
	}
	remoteUploadLog("Reconcile", "start", fmt.Sprintf("records=%d destinations=%d", len(records), len(destIDs)), profileName, "")

	staleSet := map[string]bool{}
	for i := range records {
		rec := records[i]
		if !resetInterruptedUploads(&rec) {
			continue
		}
		remoteUploadLog("Reconcile", "reset_interrupted", fmt.Sprintf("record=%s file=%q", rec.ID, filepath.Base(rec.FilePath)), profileName, "")
		if err := a.UpdateHistoryRecord(rec); err != nil {
			return nil, err
		}
	}

	if !recordsNeedRemoteVerify(records, destIDs) {
		remoteUploadLog("Reconcile", "skip_remote_verify", "no locally-done backups to verify on remote", profileName, "")
		return staleSet, nil
	}

	providers, err := providersForDestinations(destByID, destIDs)
	if err != nil {
		remoteUploadLog("Reconcile", "provider_error", "", profileName, err.Error())
		return nil, err
	}
	for destID, provider := range providers {
		dest := destByID[destID]
		remoteUploadLog("Reconcile", "provider_ready", destinationDebugDetails(dest)+" type="+string(provider.Type()), profileName, "")
	}

	for _, rec := range records {
		updated := rec
		changed := false
		for _, destID := range destIDs {
			state, ok := models.RemoteUploadStateForDestination(updated.RemoteUploads, destID)
			if !ok || state.Status != models.RemoteUploadDone {
				continue
			}
			provider, ok := providers[destID]
			if !ok {
				continue
			}
			remoteKey := state.RemoteKey
			if remoteKey == "" {
				remoteKey = backupRemoteKey(profileID, updated.ID, updated.FilePath)
			}
			dest := destByID[destID]
			remoteUploadLog("Reconcile", "stat_start", fmt.Sprintf("record=%s file=%q key=%q %s", updated.ID, filepath.Base(updated.FilePath), remoteKey, destinationDebugDetails(dest)), profileName, "")
			start := time.Now()
			exists, err := a.remoteObjectExists(ctx, provider, remoteKey)
			if err != nil {
				remoteUploadLogTimed(profileName, "Reconcile", "stat_failed", fmt.Sprintf("record=%s key=%q", updated.ID, remoteKey), start, err)
				return nil, fmt.Errorf("check remote backup %s: %w", filepath.Base(updated.FilePath), err)
			}
			if exists {
				remoteUploadLogTimed(profileName, "Reconcile", "stat_ok", fmt.Sprintf("record=%s key=%q exists=true", updated.ID, remoteKey), start, nil)
				continue
			}
			remoteUploadLogTimed(profileName, "Reconcile", "stat_missing", fmt.Sprintf("record=%s key=%q clearing local done state", updated.ID, remoteKey), start, nil)
			updated.RemoteUploads = removeRemoteUploadState(updated.RemoteUploads, destID)
			staleSet[updated.ID] = true
			changed = true
		}
		if changed {
			if err := a.UpdateHistoryRecord(updated); err != nil {
				return nil, err
			}
		}
	}
	remoteUploadLog("Reconcile", "done", fmt.Sprintf("stale_records=%d", len(staleSet)), profileName, "")
	return staleSet, nil
}

func (a *App) PrepareProfileUpload(ctx context.Context, profileID string, recordIDs []string) (RemoteUploadPlan, error) {
	profile, err := a.profileByID(profileID)
	if err != nil {
		return RemoteUploadPlan{}, err
	}
	destIDs, err := a.activeUploadDestinations(profile)
	if err != nil {
		return RemoteUploadPlan{}, err
	}
	if len(destIDs) == 0 {
		return RemoteUploadPlan{}, ErrRemoteUploadNotConfigured
	}

	records, err := a.recordsForUpload(profileID, recordIDs)
	if err != nil {
		return RemoteUploadPlan{}, err
	}
	remoteUploadLog("Prepare", "start", fmt.Sprintf("filter_records=%d history_records=%d", len(recordIDs), len(records)), profile.Name, "")

	destByID := map[string]models.RemoteDestination{}
	allDests, err := a.store.LoadRemoteDestinations()
	if err != nil {
		return RemoteUploadPlan{}, err
	}
	for _, d := range allDests {
		destByID[d.ID] = d
	}

	staleSet, err := a.reconcileRecordsBeforeUpload(ctx, profileID, records, destIDs, destByID)
	if err != nil {
		remoteUploadLog("Prepare", "failed", "", profile.Name, err.Error())
		return RemoteUploadPlan{}, err
	}
	records, err = a.recordsForUpload(profileID, recordIDs)
	if err != nil {
		return RemoteUploadPlan{}, err
	}

	var plan RemoteUploadPlan
	retrySet := map[string]bool{}
	for _, rec := range records {
		if len(pendingDestinationsForRecord(rec, destIDs)) == 0 {
			continue
		}
		if staleSet[rec.ID] {
			plan.StaleRemoteRecordIDs = append(plan.StaleRemoteRecordIDs, rec.ID)
		} else {
			retrySet[rec.ID] = true
		}
	}
	for id := range retrySet {
		plan.RetryRecordIDs = append(plan.RetryRecordIDs, id)
	}
	if len(plan.StaleRemoteRecordIDs) > 0 {
		byID := map[string]models.ExportRecord{}
		for _, rec := range records {
			byID[rec.ID] = rec
		}
		latest := plan.StaleRemoteRecordIDs[0]
		latestTime := byID[latest].ExportDate
		for _, id := range plan.StaleRemoteRecordIDs[1:] {
			rec := byID[id]
			if rec.ExportDate.After(latestTime) {
				latest = id
				latestTime = rec.ExportDate
			}
		}
		plan.LatestStaleRecordID = latest
	}
	remoteUploadLog(
		"Prepare",
		"done",
		fmt.Sprintf("retry=%d stale=%d filter_records=%d history_records=%d destinations=%d", len(plan.RetryRecordIDs), len(plan.StaleRemoteRecordIDs), len(recordIDs), len(records), len(destIDs)),
		profile.Name,
		"",
	)
	return plan, nil
}

func recordFullyUploaded(rec models.ExportRecord, destinationIDs []string) bool {
	return len(pendingDestinationsForRecord(rec, destinationIDs)) == 0
}

func countUploadWork(records []models.ExportRecord, destIDs []string) (totalWork int, pendingRecords int) {
	for _, rec := range records {
		missing := pendingDestinationsForRecord(rec, destIDs)
		if len(missing) == 0 {
			continue
		}
		pendingRecords++
		totalWork += len(missing)
	}
	return totalWork, pendingRecords
}

func computeUploadResult(records []models.ExportRecord, destIDs []string, hadWork map[string]bool) RemoteUploadResult {
	var result RemoteUploadResult
	for _, rec := range records {
		if !hadWork[rec.ID] {
			result.SkippedRecords++
			continue
		}
		if recordFullyUploaded(rec, destIDs) {
			result.UploadedRecords++
		} else {
			result.FailedRecords++
		}
	}
	return result
}

func (a *App) uploadResultFromHistory(profileID string, recordIDs []string, destIDs []string, hadWork map[string]bool) (RemoteUploadResult, error) {
	records, err := a.recordsForUpload(profileID, recordIDs)
	if err != nil {
		return RemoteUploadResult{}, err
	}
	return computeUploadResult(records, destIDs, hadWork), nil
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

func (a *App) UploadProfileBackups(ctx context.Context, profileID string, recordIDs []string, progress RemoteUploadProgressFunc) (RemoteUploadResult, error) {
	lockKey := profileID
	if !profileUploadLocks.tryAcquire(lockKey) {
		remoteUploadLog("Upload", "lock_busy", fmt.Sprintf("profile_id=%s", profileID), profileID, ErrRemoteUploadRunning.Error())
		return RemoteUploadResult{}, ErrRemoteUploadRunning
	}
	defer profileUploadLocks.release(lockKey)

	profile, err := a.profileByID(profileID)
	if err != nil {
		return RemoteUploadResult{}, err
	}
	destIDs, err := a.activeUploadDestinations(profile)
	if err != nil {
		return RemoteUploadResult{}, err
	}
	if len(destIDs) == 0 {
		return RemoteUploadResult{}, ErrRemoteUploadNotConfigured
	}

	records, err := a.recordsForUpload(profileID, recordIDs)
	if err != nil {
		return RemoteUploadResult{}, err
	}
	if len(records) == 0 {
		remoteUploadLog("Upload", "no_records", fmt.Sprintf("filter_records=%d", len(recordIDs)), profile.Name, "")
		return RemoteUploadResult{}, nil
	}
	remoteUploadLog("Upload", "start", fmt.Sprintf("filter_records=%d history_records=%d destinations=%d", len(recordIDs), len(records), len(destIDs)), profile.Name, "")

	destByID := map[string]models.RemoteDestination{}
	allDests, err := a.store.LoadRemoteDestinations()
	if err != nil {
		return RemoteUploadResult{}, err
	}
	for _, d := range allDests {
		destByID[d.ID] = d
	}

	if _, err := a.reconcileRecordsBeforeUpload(ctx, profileID, records, destIDs, destByID); err != nil {
		remoteUploadLog("Upload", "reconcile_failed", "", profile.Name, err.Error())
		return RemoteUploadResult{}, err
	}
	records, err = a.recordsForUpload(profileID, recordIDs)
	if err != nil {
		return RemoteUploadResult{}, err
	}

	totalWork, recordTotal := countUploadWork(records, destIDs)
	remoteUploadLog("Upload", "work", fmt.Sprintf("total_work=%d record_total=%d", totalWork, recordTotal), profile.Name, "")
	hadWork := map[string]bool{}
	for _, rec := range records {
		if len(pendingDestinationsForRecord(rec, destIDs)) > 0 {
			hadWork[rec.ID] = true
		}
	}

	current := 0
	emit := func(base RemoteUploadProgress) {
		if progress == nil {
			return
		}
		base.ProfileID = profileID
		base.Current = current
		base.Total = totalWork
		base.RecordTotal = recordTotal
		progress(base)
	}

	if progress != nil {
		emit(RemoteUploadProgress{Status: models.RemoteUploadPending})
	}

	var failures []error
	recordIndex := 0
	for i := range records {
		rec := records[i]
		select {
		case <-ctx.Done():
			result, resErr := a.uploadResultFromHistory(profileID, recordIDs, destIDs, hadWork)
			if resErr != nil {
				return RemoteUploadResult{}, resErr
			}
			remoteUploadLog("Upload", "canceled", fmt.Sprintf("current=%d total=%d", current, totalWork), profile.Name, ctx.Err().Error())
			return result, ctx.Err()
		default:
		}
		missing := pendingDestinationsForRecord(rec, destIDs)
		if len(missing) == 0 {
			continue
		}
		recordIndex++
		for _, destID := range missing {
			select {
			case <-ctx.Done():
				result, resErr := a.uploadResultFromHistory(profileID, recordIDs, destIDs, hadWork)
				if resErr != nil {
					return RemoteUploadResult{}, resErr
				}
				remoteUploadLog("Upload", "canceled", fmt.Sprintf("current=%d total=%d", current, totalWork), profile.Name, ctx.Err().Error())
				return result, ctx.Err()
			default:
			}
			dest, ok := destByID[destID]
			if !ok {
				continue
			}
			remoteUploadLog(
				"Upload",
				"destination_start",
				fmt.Sprintf("record=%s file=%q dest_id=%s %s", rec.ID, filepath.Base(rec.FilePath), destID, destinationDebugDetails(dest)),
				profile.Name,
				"",
			)
			start := time.Now()
			uploadErr := a.uploadRecordToDestination(ctx, profileID, &rec, dest, func(p RemoteUploadProgress) {
				p.RecordIndex = recordIndex
				emit(p)
			})
			records[i] = rec
			current++
			status := models.RemoteUploadDone
			errMsg := ""
			if uploadErr != nil {
				status = models.RemoteUploadFailed
				errMsg = uploadErr.Error()
				failures = append(failures, fmt.Errorf("%s: %w", destID, uploadErr))
				remoteUploadLogTimed(profile.Name, "Upload", "destination_failed", fmt.Sprintf("record=%s dest_id=%s key=%q", rec.ID, destID, backupRemoteKey(profileID, rec.ID, rec.FilePath)), start, uploadErr)
			} else {
				remoteUploadLogTimed(profile.Name, "Upload", "destination_done", fmt.Sprintf("record=%s dest_id=%s", rec.ID, destID), start, nil)
			}
			emit(RemoteUploadProgress{
				RecordID:      rec.ID,
				DestinationID: destID,
				Status:        status,
				Error:         errMsg,
				RecordIndex:   recordIndex,
			})
		}
	}

	result, err := a.uploadResultFromHistory(profileID, recordIDs, destIDs, hadWork)
	if err != nil {
		return RemoteUploadResult{}, err
	}
	if len(failures) > 0 {
		remoteUploadLog("Upload", "done_with_errors", fmt.Sprintf("uploaded=%d failed=%d skipped=%d failures=%d", result.UploadedRecords, result.FailedRecords, result.SkippedRecords, len(failures)), profile.Name, errors.Join(failures...).Error())
		return result, fmt.Errorf("remote upload completed with %d failure(s): %w", len(failures), errors.Join(failures...))
	}
	remoteUploadLog("Upload", "done", fmt.Sprintf("uploaded=%d failed=%d skipped=%d", result.UploadedRecords, result.FailedRecords, result.SkippedRecords), profile.Name, "")
	return result, nil
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
	remoteUploadLog("Put", "start", fmt.Sprintf("record=%s file=%q size=%d key=%q %s", rec.ID, filepath.Base(rec.FilePath), info.Size(), remoteKey, destinationDebugDetails(dest)), "", "")
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

func (a *App) MaybeAutoUploadAfterBackup(ctx context.Context, profile models.Profile, records []models.ExportRecord) (RemoteUploadResult, error) {
	if len(profile.RemoteUploadDestinationIDs) == 0 {
		return RemoteUploadResult{}, nil
	}
	uploadDB := profile.RemoteAutoUploadDB
	uploadFiles := profile.RemoteAutoUploadFiles
	if !uploadDB && !uploadFiles {
		return RemoteUploadResult{}, nil
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
		return RemoteUploadResult{}, nil
	}
	return a.UploadProfileBackups(ctx, profile.ID, ids, nil)
}

func FormatRemoteUploadResultMessage(result RemoteUploadResult) string {
	if result.UploadedRecords == 0 && result.FailedRecords == 0 && result.SkippedRecords > 0 {
		return "All backups already uploaded"
	}
	if result.UploadedRecords == 0 && result.FailedRecords == 0 {
		return "No backups to upload"
	}
	if result.FailedRecords == 0 {
		if result.UploadedRecords == 1 {
			return "1 backup uploaded"
		}
		return fmt.Sprintf("%d backups uploaded", result.UploadedRecords)
	}
	if result.UploadedRecords == 0 {
		if result.FailedRecords == 1 {
			return "1 backup failed to upload"
		}
		return fmt.Sprintf("%d backups failed to upload", result.FailedRecords)
	}
	return fmt.Sprintf("%d backups uploaded, %d failed", result.UploadedRecords, result.FailedRecords)
}
