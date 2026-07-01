package ui

import (
	"context"
	"errors"
	"fmt"

	"dback/backend/verify"
	coreapp "dback/internal/app"
	"dback/models"
)

func recordQuickVerified(record models.ExportRecord) *models.LastVerified {
	if record.QuickVerified != nil {
		return record.QuickVerified
	}
	if record.LastVerified != nil && record.LastVerified.Method != "deep" {
		return record.LastVerified
	}
	return nil
}

func recordDeepVerified(record models.ExportRecord) *models.LastVerified {
	if record.DeepVerified != nil {
		return record.DeepVerified
	}
	if record.LastVerified != nil && record.LastVerified.Method == "deep" {
		return record.LastVerified
	}
	return nil
}

func (u *UI) backupQuickVerifyStatus(record models.ExportRecord) string {
	if u.isBackupVerifyActive(record) {
		return "Verifying..."
	}
	switch coreapp.BackupVerifyStatus(record) {
	case "done":
		return "Done"
	case "failed":
		return "Failed"
	case "verifying":
		return "Verifying..."
	default:
		return "—"
	}
}

func (u *UI) backupDetailFileVerifyLabel(record models.ExportRecord) string {
	if u.isBackupVerifyActive(record) {
		return "Verifying file integrity..."
	}
	switch coreapp.BackupVerifyStatus(record) {
	case "done":
		return "Verified — SHA256 matched, file integrity OK"
	case "failed":
		return "Failed — file may be corrupted or modified"
	case "verifying":
		return "Verifying file integrity..."
	default:
		return "Not verified yet"
	}
}

func (u *UI) backupDetailDeepVerifyLabel(record models.ExportRecord) string {
	if u.isDeepVerifyJobActive(record.ID) {
		return "Deep verify in progress..."
	}
	switch coreapp.BackupDeepVerifyStatus(record) {
	case "matched":
		return "Done — SHA256 matched, row counts match"
	case "row_diff":
		return "Done — SHA256 matched, some row counts differ"
	default:
		return "Not run yet"
	}
}

func (u *UI) backupDeepVerifyStatus(record models.ExportRecord) string {
	if u.isDeepVerifyJobActive(record.ID) {
		return "Verifying..."
	}
	switch coreapp.BackupDeepVerifyStatus(record) {
	case "matched":
		return "SHA256 OK · rows match"
	case "row_diff":
		return "SHA256 OK · rows differ"
	default:
		return "—"
	}
}

func (u *UI) isBackupVerifyActive(record models.ExportRecord) bool {
	u.jobsMu.Lock()
	defer u.jobsMu.Unlock()
	for _, job := range u.jobs {
		if job.Done || job.Kind != "Backup" {
			continue
		}
		if job.RecordID != "" && job.RecordID == record.ID {
			return job.VerifyPhase
		}
		if job.RecordID == "" && job.ProfileName == record.ProfileName && job.VerifyPhase {
			return true
		}
	}
	return false
}

func (u *UI) isDeepVerifyJobActive(recordID string) bool {
	u.jobsMu.Lock()
	defer u.jobsMu.Unlock()
	for _, job := range u.jobs {
		if job.Done || job.Kind != "Deep verify" {
			continue
		}
		if job.RecordID == recordID {
			return true
		}
	}
	return false
}

func deepVerifyDialogTitle(passed bool) string {
	if passed {
		return "Deep verify — SHA256 matched, rows match"
	}
	return "Deep verify — SHA256 matched, rows differ"
}

func deepVerifyJobResultMessage(passed bool) string {
	if passed {
		return "Deep verify done — SHA256 matched, rows match"
	}
	return "Deep verify done — SHA256 matched, rows differ"
}

func verifySummaryMessage(passed bool, summary verify.ReportSummary, fingerprintMode string) string {
	if passed {
		return fmt.Sprintf(
			"Your backup file is fine. SHA256 matched and row counts match across all %d tables.",
			summary.Total,
		)
	}
	msg := fmt.Sprintf(
		"Your backup file is fine. SHA256 matched, but %d of %d tables have more or fewer rows than recorded at backup time.",
		summary.Mismatched,
		summary.Total,
	)
	if fingerprintMode == verify.ModeFast || fingerprintMode == "" {
		msg += "\n\nThis is common and usually not a problem. Counts at backup time are fast estimates, and busy tables (scheduler, logs, analytics) change often."
	} else {
		msg += "\n\nReview the tables below if you want to double-check before importing."
	}
	return msg
}

func (u *UI) runDeepVerifyPrompt(record models.ExportRecord) {
	if !record.SupportsImport() {
		u.showInfo("Deep verify unavailable", "Deep verify is only available for database backups.")
		return
	}
	if record.Fingerprint == nil {
		u.showInfo("Deep verify unavailable", "No fingerprint available. Re-create this backup to enable deep verify.")
		return
	}
	quick := recordQuickVerified(record)
	if quick != nil && !quick.Passed {
		u.showError(fmt.Errorf("file integrity check failed; fix or re-create the backup before deep verify"))
		return
	}
	if record.Sha256 == "" {
		if coreapp.BackupVerifyStatus(record) == "verifying" {
			u.showInfo("Backup in progress", "Wait for backup verify to finish before running deep verify.")
			return
		}
		u.showInfo("Deep verify unavailable", "No checksum stored for this backup. Create a new backup first.")
		return
	}
	hosts := importableProfiles(u.core.Profiles())
	if len(hosts) == 0 {
		u.showError(fmt.Errorf("no import destinations available for deep verify"))
		return
	}
	u.deepVerifySelect.Value = defaultDeepVerifyHostID(hosts)
	rec := record
	fpMode := ""
	if rec.Fingerprint != nil {
		fpMode = rec.Fingerprint.Mode
	}
	u.showDialog(DialogState{
		Kind:    DialogDeepVerifyConfirm,
		Title:   "Deep verify",
		Message: "Restore runs only in a temporary database (dback_verify_*). Your production database is not modified. The temp database is deleted when verify finishes.",
		OnOK: func() {
			dest, ok := profileByID(hosts, u.deepVerifySelect.Value)
			if !ok {
				u.showError(fmt.Errorf("select a host for deep verify"))
				return
			}
			u.runDeepVerify(rec, dest, fpMode)
		},
		OnCancel: func() {},
	})
}

func (u *UI) runDeepVerify(record models.ExportRecord, dest models.Profile, fingerprintMode string) {
	ctx, cancel := context.WithCancel(context.Background())
	job := u.addJob("Deep verify", dest.Name, cancel)
	job.RecordID = record.ID
	u.backupTab = 1
	u.invalidateBackupCache()
	go func() {
		defer cancel()
		last, err := u.core.DeepVerify(ctx, record.ID, dest, func(message string, current int64, total int64) {
			progress := float64(0)
			if total > 0 {
				progress = float64(current) / float64(total)
			}
			u.updateJob(job.ID, message, progress, "")
		})
		u.invalidateBackupCache()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				u.finishJob(job.ID, "Deep verify canceled", nil)
				return
			}
			if len(last.Report) > 0 {
				u.finishJob(job.ID, deepVerifyJobResultMessage(last.Passed), nil)
				u.showVerifyReport(last.Passed, last.Report, fingerprintMode)
				return
			}
			u.finishJob(job.ID, "Deep verify failed", err)
			u.showError(err)
			return
		}
		u.finishJob(job.ID, deepVerifyJobResultMessage(true), nil)
		u.showVerifyReport(true, last.Report, fingerprintMode)
	}()
}

func (u *UI) showVerifyReport(passed bool, report []models.TableVerifyResult, fingerprintMode string) {
	summary, _, _ := verify.PartitionReport(report)
	u.showDialog(DialogState{
		Kind:                  DialogVerifyReport,
		Title:                 deepVerifyDialogTitle(passed),
		Message:               verifySummaryMessage(passed, summary, fingerprintMode),
		VerifyReport:          report,
		VerifyPassed:          passed,
		VerifyFingerprintMode: fingerprintMode,
	})
}
