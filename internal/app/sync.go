package app

import (
	"context"

	"dback/internal/store"
	"dback/internal/sync"
	"dback/models"
)

func (a *App) SyncSettings() (*models.SyncSettings, error) {
	return a.store.LoadSyncSettings()
}

func (a *App) SyncActivity() (models.SyncActivity, error) {
	return a.store.LoadSyncActivity()
}

func (a *App) SaveSyncSettings(settings models.SyncSettings) error {
	settings.Endpoint = sync.NormalizeEndpoint(settings.Endpoint)
	return a.store.SaveSyncSettings(settings)
}

func (a *App) TestSyncConnection(ctx context.Context, cfg models.SyncSettings) error {
	cfg.Endpoint = sync.NormalizeEndpoint(cfg.Endpoint)
	return sync.TestConnection(ctx, cfg)
}

// SyncPush encrypts with the vault master key from the current unlock session.
func (a *App) SyncPush(ctx context.Context) error {
	cfg, err := a.store.LoadSyncSettings()
	if err != nil {
		return err
	}
	if cfg == nil {
		return store.ErrSyncNotConfigured
	}
	data, err := a.store.MarshalAppDataBundleForSync(a.currentAppImportData(cfg))
	if err != nil {
		return err
	}
	if err := sync.Push(ctx, *cfg, data); err != nil {
		return err
	}
	return a.store.RecordSyncPush()
}

func (a *App) SyncDownload(ctx context.Context) ([]byte, error) {
	cfg, err := a.store.LoadSyncSettings()
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, store.ErrSyncNotConfigured
	}
	return sync.Pull(ctx, *cfg)
}

// PreviewSyncImport decrypts with the vault master key from the current unlock session.
func (a *App) PreviewSyncImport(raw []byte) (store.AppImportData, []store.ProfileConflict, []store.TemplateConflict, error) {
	imported, err := a.store.ImportAppDataBundleForSync(raw)
	if err != nil {
		return store.AppImportData{}, nil, nil, err
	}
	profileConflicts := store.DetectProfileConflicts(a.Profiles(), imported.Profiles)
	templateConflicts := store.DetectTemplateConflicts(a.Templates(), imported.Templates)
	return imported, profileConflicts, templateConflicts, nil
}

func (a *App) RecordSyncPull() error {
	return a.store.RecordSyncPull()
}

func (a *App) currentAppImportData(syncSettings *models.SyncSettings) store.AppImportData {
	return store.AppImportData{
		Profiles:  a.Profiles(),
		Templates: a.Templates(),
		History:   a.History(),
		Logs:      a.Logs(),
		Sync:      syncSettings.Clone(),
	}
}

func (a *App) applyImportedAppData(imported store.AppImportData) error {
	a.mu.Lock()
	a.profiles = store.MergeProfiles(a.profiles, imported.Profiles)
	a.templates = store.MergeTemplates(a.templates, imported.Templates)
	a.history = store.MergeHistory(a.history, imported.History)
	a.logs = store.MergeLogs(a.logs, imported.Logs)
	profiles := append([]models.Profile(nil), a.profiles...)
	templates := append([]models.SQLTemplate(nil), a.templates...)
	history := append([]models.ExportRecord(nil), a.history...)
	logs := append([]models.LogEntry(nil), a.logs...)
	a.mu.Unlock()

	if err := a.store.SaveProfiles(profiles); err != nil {
		return err
	}
	if err := a.store.SaveTemplates(templates); err != nil {
		return err
	}
	if err := a.store.SaveHistory(history); err != nil {
		return err
	}
	if err := a.store.SaveLogs(logs); err != nil {
		return err
	}
	if imported.Sync != nil {
		if err := a.store.SaveSyncSettings(*imported.Sync); err != nil {
			return err
		}
	}
	return a.Reload()
}

func (a *App) ImportAppDataFromBundle(imported store.AppImportData) error {
	return a.applyImportedAppData(imported)
}
