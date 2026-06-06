package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"dback/backend/db"
	"dback/backend/transfer"
	"dback/backend/verify"
	"dback/models"
)

type appQueryRunner struct {
	app *App
}

func (r appQueryRunner) RunQuery(ctx context.Context, profile models.Profile, query string, connectDB bool) (db.QueryResult, error) {
	return r.app.RunImportQuery(ctx, profile, query, connectDB)
}

func (a *App) findHistoryRecord(id string) (models.ExportRecord, int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i, rec := range a.history {
		if rec.ID == id {
			return rec, i, nil
		}
	}
	return models.ExportRecord{}, -1, fmt.Errorf("backup record not found")
}

// UpdateHistoryRecord persists a single updated backup history entry.
func (a *App) UpdateHistoryRecord(record models.ExportRecord) error {
	a.mu.Lock()
	updated := false
	for i, rec := range a.history {
		if rec.ID == record.ID {
			a.history[i] = record
			updated = true
			break
		}
	}
	if !updated {
		a.mu.Unlock()
		return fmt.Errorf("backup record not found")
	}
	history := append([]models.ExportRecord(nil), a.history...)
	a.mu.Unlock()
	return a.store.SaveHistory(history)
}

func (a *App) captureBackupMetadata(ctx context.Context, profile models.Profile, filePath, databaseName string) (sha256 string, fingerprint *models.BackupFingerprint) {
	sum, err := verify.ChecksumFile(filePath)
	if err == nil {
		sha256 = sum
	}
	runner := appQueryRunner{app: a}
	fp, err := verify.CaptureFingerprint(ctx, runner, profile, databaseName, verify.ModeFast)
	if err == nil {
		fingerprint = &fp
	}
	return sha256, fingerprint
}

func applyAutoQuickVerify(record *models.ExportRecord) {
	if record.Sha256 == "" {
		record.QuickVerified = &models.LastVerified{
			VerifiedAt: time.Now().UTC(),
			Method:     "quick",
			Passed:     false,
		}
		return
	}
	result, err := verify.QuickCheck(record.FilePath, record.Sha256)
	passed := err == nil && result.Passed
	record.QuickVerified = &models.LastVerified{
		VerifiedAt: time.Now().UTC(),
		Method:     "quick",
		Passed:     passed,
	}
}

// BackupVerifyStatus returns quick verify display state: verifying, done, failed, or none.
func BackupVerifyStatus(record models.ExportRecord) string {
	if record.QuickVerified != nil {
		if record.QuickVerified.Passed {
			return "done"
		}
		return "failed"
	}
	if record.LastVerified != nil && record.LastVerified.Method != "deep" {
		if record.LastVerified.Passed {
			return "done"
		}
		return "failed"
	}
	if record.Sha256 == "" && strings.TrimSpace(record.FilePath) != "" {
		return "verifying"
	}
	if record.Sha256 == "" {
		return "none"
	}
	result, err := verify.QuickCheck(record.FilePath, record.Sha256)
	if err != nil || !result.Passed {
		return "failed"
	}
	return "done"
}

// BackupDeepVerifyStatus returns deep verify display state:
// matched (SHA256 OK and rows match), row_diff (SHA256 OK but row counts differ), or none.
func BackupDeepVerifyStatus(record models.ExportRecord) string {
	if record.DeepVerified != nil {
		if record.DeepVerified.Passed {
			return "matched"
		}
		return "row_diff"
	}
	if record.LastVerified != nil && record.LastVerified.Method == "deep" {
		if record.LastVerified.Passed {
			return "matched"
		}
		return "row_diff"
	}
	return "none"
}

// QuickVerify checks local file SHA256 against the stored checksum.
func (a *App) QuickVerify(ctx context.Context, recordID string) (models.LastVerified, error) {
	if err := ctx.Err(); err != nil {
		return models.LastVerified{}, err
	}
	record, _, err := a.findHistoryRecord(recordID)
	if err != nil {
		return models.LastVerified{}, err
	}
	result, err := verify.QuickCheck(record.FilePath, record.Sha256)
	if err != nil {
		return models.LastVerified{}, err
	}
	last := models.LastVerified{
		VerifiedAt: time.Now().UTC(),
		Method:     "quick",
		Passed:     result.Passed,
	}
	record.QuickVerified = &last
	if err := a.UpdateHistoryRecord(record); err != nil {
		return last, err
	}
	if !result.Passed {
		return last, fmt.Errorf("file is corrupted or has been modified")
	}
	return last, nil
}

// DeepVerify restores the backup to a temporary database and compares row counts.
func (a *App) DeepVerify(ctx context.Context, recordID string, destination models.Profile, progress ProgressFunc) (models.LastVerified, error) {
	if err := ctx.Err(); err != nil {
		return models.LastVerified{}, err
	}
	if !destination.AllowsImport() {
		return models.LastVerified{}, fmt.Errorf("host %q is protected from import", destination.Name)
	}
	record, _, err := a.findHistoryRecord(recordID)
	if err != nil {
		return models.LastVerified{}, err
	}
	if record.Fingerprint == nil {
		return models.LastVerified{}, fmt.Errorf("no fingerprint available; re-create this backup to enable deep verify")
	}
	quick, err := verify.QuickCheck(record.FilePath, record.Sha256)
	if err != nil {
		return models.LastVerified{}, err
	}
	if !quick.Passed {
		return models.LastVerified{}, fmt.Errorf("file integrity check failed; file is corrupted or has been modified")
	}

	tempDB := verify.TempDBName()
	if !strings.HasPrefix(tempDB, "dback_verify_") {
		return models.LastVerified{}, fmt.Errorf("invalid temp verify database name")
	}
	originalDB := strings.TrimSpace(destination.TargetDBName)
	dropped := false
	defer func() {
		if dropped {
			return
		}
		dropCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		_ = a.dropVerifyDatabase(dropCtx, destination, tempDB)
	}()

	if destination.UsesWordPress() {
		if err := a.prepareVerifyDatabase(ctx, destination, tempDB); err != nil {
			return models.LastVerified{}, err
		}
	}

	operationID := newID()
	logger := a.newOpLogger(operationID, &destination)
	restoreReq := transfer.RestoreRequest{
		Profile:          destination,
		OperationID:      operationID,
		LocalPath:        record.FilePath,
		FileSize:         record.FileSizeBytes,
		Logger:           logger,
		Progress:         progress,
		TargetDBOverride: tempDB,
	}
	var restoreErr error
	if destination.UsesWordPress() {
		restoreErr = transfer.RestoreWordPress(ctx, restoreReq)
	} else {
		restoreErr = transfer.RestoreSSH(ctx, restoreReq)
	}
	if restoreErr != nil {
		return models.LastVerified{}, restoreErr
	}

	tables := make([]string, 0, len(record.Fingerprint.Tables))
	for name := range record.Fingerprint.Tables {
		tables = append(tables, name)
	}
	runner := appQueryRunner{app: a}
	actual, err := verify.CountTablesExact(ctx, runner, destination, tempDB, tables)
	if err != nil {
		return models.LastVerified{}, err
	}

	report, passed := verify.BuildTableReport(record.Fingerprint, actual)
	last := models.LastVerified{
		VerifiedAt: time.Now().UTC(),
		Method:     "deep",
		Passed:     passed,
		Report:     report,
	}
	record.DeepVerified = &last
	if err := a.UpdateHistoryRecord(record); err != nil {
		return last, err
	}

	if err := a.dropVerifyDatabase(ctx, destination, tempDB); err != nil {
		return last, fmt.Errorf("verify completed but failed to drop temp database %q: %w", tempDB, err)
	}
	dropped = true

	if originalDB != "" && originalDB == tempDB {
		return last, fmt.Errorf("safety check failed: temp database must not match production database")
	}

	if !passed {
		return last, fmt.Errorf("deep verify found table row mismatches")
	}
	return last, nil
}

func (a *App) prepareVerifyDatabase(ctx context.Context, profile models.Profile, tempDB string) error {
	sql := db.BuildDropDatabaseCommand(tempDB) + " " + fmt.Sprintf(
		"CREATE DATABASE %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;",
		db.SQLIdent(tempDB),
	)
	_, err := a.RunImportQuery(ctx, profile, sql, false)
	return err
}

func (a *App) dropVerifyDatabase(ctx context.Context, profile models.Profile, tempDB string) error {
	_, err := a.RunImportQuery(ctx, profile, db.BuildDropDatabaseCommand(tempDB), false)
	return err
}
