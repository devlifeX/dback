package app

import (
	"context"
	"os"
	"sort"

	"dback/internal/remote"
	"dback/models"
)

func (a *App) ApplyRetention(profileID string) error {
	profile, err := a.profileByID(profileID)
	if err != nil {
		return err
	}
	if !profile.BackupPolicy.Enabled {
		return nil
	}
	policy := profile.BackupPolicy

	if err := a.applyLocalRetention(profileID, models.ExportTypeDatabase, policy.DBLocalKeep); err != nil {
		return err
	}
	if err := a.applyLocalRetention(profileID, models.ExportTypeFiles, policy.FilesLocalKeep); err != nil {
		return err
	}
	if err := a.applyRemoteRetention(profile, models.ExportTypeDatabase, policy.DBRemoteKeep); err != nil {
		return err
	}
	return a.applyRemoteRetention(profile, models.ExportTypeFiles, policy.FilesRemoteKeep)
}

func (a *App) applyLocalRetention(profileID string, exportType models.ExportType, keep int) error {
	if keep <= 0 {
		return nil
	}
	records := a.recordsByType(profileID, exportType)
	if len(records) <= keep {
		return nil
	}
	toDelete := records[keep:]
	removeIDs := map[string]struct{}{}
	for _, rec := range toDelete {
		removeIDs[rec.ID] = struct{}{}
		if rec.FilePath != "" {
			_ = os.Remove(rec.FilePath)
		}
	}

	a.mu.Lock()
	var kept []models.ExportRecord
	for _, rec := range a.history {
		if rec.ProfileID == profileID && rec.EffectiveExportType() == exportType {
			if _, drop := removeIDs[rec.ID]; drop {
				continue
			}
		}
		kept = append(kept, rec)
	}
	history := append([]models.ExportRecord(nil), kept...)
	a.mu.Unlock()
	return a.store.SaveHistory(history)
}

func (a *App) applyRemoteRetention(profile models.Profile, exportType models.ExportType, keep int) error {
	if keep <= 0 {
		return nil
	}
	records := a.recordsByType(profile.ID, exportType)
	if len(records) <= keep {
		return nil
	}
	destByID, err := a.destinationsByID()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), remote.ListObjectsTimeout)
	defer cancel()

	for _, rec := range records[keep:] {
		for i := range rec.RemoteUploads {
			state := rec.RemoteUploads[i]
			if state.Status != models.RemoteUploadDone || state.RemoteKey == "" {
				continue
			}
			dest, ok := destByID[state.DestinationID]
			if !ok {
				continue
			}
			provider, perr := remote.NewProvider(dest)
			if perr != nil {
				continue
			}
			_ = provider.DeleteObject(ctx, state.RemoteKey)
			rec.RemoteUploads[i].Status = models.RemoteUploadFailed
			rec.RemoteUploads[i].Error = "removed by retention policy"
		}
		_ = a.UpdateHistoryRecord(rec)
	}
	return nil
}

func (a *App) recordsByType(profileID string, exportType models.ExportType) []models.ExportRecord {
	var out []models.ExportRecord
	for _, rec := range a.History() {
		if rec.ProfileID != profileID || rec.EffectiveExportType() != exportType {
			continue
		}
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ExportDate.After(out[j].ExportDate)
	})
	return out
}

func (a *App) destinationsByID() (map[string]models.RemoteDestination, error) {
	all, err := a.store.LoadRemoteDestinations()
	if err != nil {
		return nil, err
	}
	out := make(map[string]models.RemoteDestination, len(all))
	for _, d := range all {
		out[d.ID] = d
	}
	return out, nil
}
