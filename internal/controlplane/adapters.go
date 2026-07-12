package controlplane

import (
	"context"
	"fmt"

	"dback/internal/app"
	"dback/internal/operation"
	"dback/models"
)

func RegisterAppHandlers(registry *Registry, application *app.App) {
	registry.Register(operation.KindBackupDB, backupDBHandler(application))
	registry.Register(operation.KindBackupFiles, backupFilesHandler(application))
	registry.Register(operation.KindUpload, uploadHandler(application))
	registry.Register(operation.KindRestore, restoreHandler(application))
	registry.Register(operation.KindDeepVerify, deepVerifyHandler(application))
	registry.Register(operation.KindUrlChecker, urlCheckerHandler(application))
}

func backupDBHandler(application *app.App) Handler {
	return func(ctx context.Context, spec operation.Spec, publishProgress func(message string, current, total int64)) (*operation.Result, error) {
		if _, ok := spec.Params.(operation.BackupDBParams); !ok {
			return nil, fmt.Errorf("invalid params for backup_db")
		}
		profile, err := profileByID(application, spec.ProfileID)
		if err != nil {
			return nil, err
		}
		record, err := application.BackupWithOperationID(ctx, spec.ID, profile, func(message string, current, total int64) {
			if publishProgress != nil {
				publishProgress(message, current, total)
			}
		})
		if err != nil {
			return &operation.Result{
				OperationID: spec.ID,
				Kind:        operation.KindBackupDB,
				Status:      operation.StatusFailed,
				Error:       err.Error(),
			}, err
		}
		details := autoUploadDetails(ctx, application, spec.ID, profile, []models.ExportRecord{record})
		return &operation.Result{
			OperationID: spec.ID,
			Kind:        operation.KindBackupDB,
			Status:      operation.StatusSucceeded,
			Details:     details,
			Artifacts: []operation.Artifact{{
				Type: operation.ArtifactExportRecord,
				ID:   record.ID,
				Path: record.FilePath,
			}},
		}, nil
	}
}

func backupFilesHandler(application *app.App) Handler {
	return func(ctx context.Context, spec operation.Spec, publishProgress func(message string, current, total int64)) (*operation.Result, error) {
		if _, ok := spec.Params.(operation.BackupFilesParams); !ok {
			return nil, fmt.Errorf("invalid params for backup_files")
		}
		profile, err := profileByID(application, spec.ProfileID)
		if err != nil {
			return nil, err
		}
		result, err := application.BackupFilesWithOperationID(ctx, spec.ID, profile, func(prog app.FileBackupProgress) {
			if publishProgress == nil {
				return
			}
			total := int64(prog.PathTotal)
			current := int64(prog.PathIndex - 1)
			if prog.BytesTotal > 0 {
				current = int64(prog.PathIndex-1)*prog.BytesTotal + prog.BytesDone
				total = int64(prog.PathTotal) * prog.BytesTotal
			}
			publishProgress(prog.PathName, current, total)
		})
		if err != nil {
			return &operation.Result{
				OperationID: spec.ID,
				Kind:        operation.KindBackupFiles,
				Status:      operation.StatusFailed,
				Error:       err.Error(),
				Artifacts:   artifactsFromRecords(result.Records),
			}, err
		}
		details := autoUploadDetails(ctx, application, spec.ID, profile, result.Records)
		return &operation.Result{
			OperationID: spec.ID,
			Kind:        operation.KindBackupFiles,
			Status:      operation.StatusSucceeded,
			Details:     details,
			Artifacts:   artifactsFromRecords(result.Records),
		}, nil
	}
}

func uploadHandler(application *app.App) Handler {
	return func(ctx context.Context, spec operation.Spec, publishProgress func(message string, current, total int64)) (*operation.Result, error) {
		params, ok := spec.Params.(operation.UploadParams)
		if !ok {
			return nil, fmt.Errorf("invalid params for upload")
		}
		recordIDs := params.RecordIDs
		if params.UploadAll {
			recordIDs = nil
		}
		uploadResult, err := application.UploadProfileBackupsWithOperationID(ctx, spec.ID, spec.ProfileID, recordIDs, func(prog app.RemoteUploadProgress) {
			if publishProgress == nil {
				return
			}
			publishProgress(string(prog.Status), int64(prog.Current), int64(prog.Total))
		})
		if err != nil {
			return &operation.Result{
				OperationID: spec.ID,
				Kind:        operation.KindUpload,
				Status:      operation.StatusFailed,
				Error:       err.Error(),
			}, err
		}
		if uploadResult.FailedRecords > 0 {
			msg := fmt.Sprintf("upload completed with %d failed record(s)", uploadResult.FailedRecords)
			return &operation.Result{
				OperationID: spec.ID,
				Kind:        operation.KindUpload,
				Status:      operation.StatusFailed,
				Error:       msg,
			}, fmt.Errorf("%s", msg)
		}
		return &operation.Result{
			OperationID: spec.ID,
			Kind:        operation.KindUpload,
			Status:      operation.StatusSucceeded,
		}, nil
	}
}

func profileByID(application *app.App, profileID string) (models.Profile, error) {
	for _, p := range application.Profiles() {
		if p.ID == profileID {
			return p, nil
		}
	}
	return models.Profile{}, fmt.Errorf("profile %q not found", profileID)
}

func autoUploadDetails(ctx context.Context, application *app.App, operationID string, profile models.Profile, records []models.ExportRecord) string {
	if len(records) == 0 {
		return ""
	}
	result, err := application.MaybeAutoUploadAfterBackup(ctx, operationID, profile, records)
	if err != nil {
		return "Auto-upload failed: " + err.Error()
	}
	if result.UploadedRecords == 0 && result.FailedRecords == 0 && result.SkippedRecords == 0 {
		return ""
	}
	return app.FormatRemoteUploadResultMessage(result)
}

func artifactsFromRecords(records []models.ExportRecord) []operation.Artifact {
	out := make([]operation.Artifact, 0, len(records))
	for _, rec := range records {
		out = append(out, operation.Artifact{
			Type: operation.ArtifactExportRecord,
			ID:   rec.ID,
			Path: rec.FilePath,
		})
	}
	return out
}

func restoreHandler(application *app.App) Handler {
	return func(ctx context.Context, spec operation.Spec, publishProgress func(message string, current, total int64)) (*operation.Result, error) {
		params, ok := spec.Params.(operation.RestoreParams)
		if !ok {
			return nil, fmt.Errorf("invalid params for restore")
		}
		record, err := historyRecordByID(application, params.RecordID)
		if err != nil {
			return nil, err
		}
		destination, err := profileByID(application, params.DestinationProfileID)
		if err != nil {
			return nil, err
		}
		progress := func(message string, current, total int64) {
			if publishProgress != nil {
				publishProgress(message, current, total)
			}
		}
		if err := application.RestoreWithOperationID(ctx, spec.ID, record, destination, progress); err != nil {
			return &operation.Result{
				OperationID: spec.ID,
				Kind:        operation.KindRestore,
				Status:      operation.StatusFailed,
				Error:       err.Error(),
			}, err
		}
		return &operation.Result{
			OperationID: spec.ID,
			Kind:        operation.KindRestore,
			Status:      operation.StatusSucceeded,
			Artifacts: []operation.Artifact{{
				Type: operation.ArtifactExportRecord,
				ID:   record.ID,
				Path: record.FilePath,
			}},
		}, nil
	}
}

func deepVerifyHandler(application *app.App) Handler {
	return func(ctx context.Context, spec operation.Spec, publishProgress func(message string, current, total int64)) (*operation.Result, error) {
		params, ok := spec.Params.(operation.DeepVerifyParams)
		if !ok {
			return nil, fmt.Errorf("invalid params for deep_verify")
		}
		destination, err := profileByID(application, params.DestinationProfileID)
		if err != nil {
			return nil, err
		}
		progress := func(message string, current, total int64) {
			if publishProgress != nil {
				publishProgress(message, current, total)
			}
		}
		last, err := application.DeepVerifyWithOperationID(ctx, spec.ID, params.RecordID, destination, progress)
		if err != nil {
			return &operation.Result{
				OperationID: spec.ID,
				Kind:        operation.KindDeepVerify,
				Status:      operation.StatusFailed,
				Error:       err.Error(),
			}, err
		}
		status := operation.StatusSucceeded
		errMsg := ""
		if !last.Passed {
			status = operation.StatusFailed
			errMsg = "deep verify found table row mismatches"
		}
		return &operation.Result{
			OperationID: spec.ID,
			Kind:        operation.KindDeepVerify,
			Status:      status,
			Error:       errMsg,
			Artifacts: []operation.Artifact{{
				Type: operation.ArtifactExportRecord,
				ID:   params.RecordID,
			}},
		}, nil
	}
}

func historyRecordByID(application *app.App, recordID string) (models.ExportRecord, error) {
	for _, rec := range application.History() {
		if rec.ID == recordID {
			return rec, nil
		}
	}
	return models.ExportRecord{}, fmt.Errorf("backup record %q not found", recordID)
}

func urlCheckerHandler(application *app.App) Handler {
	return func(ctx context.Context, spec operation.Spec, publishProgress func(message string, current, total int64)) (*operation.Result, error) {
		params, ok := spec.Params.(operation.UrlCheckerParams)
		if !ok {
			return nil, fmt.Errorf("invalid params for url_checker")
		}
		if publishProgress != nil {
			publishProgress("Checking URLs", 0, 1)
		}
		outcomes, checkErr := application.CheckURLs(ctx, spec.ProfileID, spec.ID, params)
		details, _ := application.FormatURLCheckNotifyReport(spec.ProfileID, outcomes)
		if checkErr != nil {
			return &operation.Result{
				OperationID: spec.ID,
				Kind:        operation.KindUrlChecker,
				Status:      operation.StatusFailed,
				Error:       checkErr.Error(),
				Details:     details,
			}, checkErr
		}
		if publishProgress != nil {
			publishProgress("URL checks passed", 1, 1)
		}
		return &operation.Result{
			OperationID: spec.ID,
			Kind:        operation.KindUrlChecker,
			Status:      operation.StatusSucceeded,
			Details:     details,
		}, nil
	}
}
