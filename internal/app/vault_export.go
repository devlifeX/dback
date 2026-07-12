package app

import (
	"os"

	"dback/internal/store"
)

func (a *App) ExportAppDataBytes(includeSecrets bool, passphrase string) ([]byte, error) {
	return a.store.MarshalAppDataBundle(a.collectAppImportData(), includeSecrets, passphrase)
}

func (a *App) PreviewImportAppDataBytes(raw []byte, includeSecrets bool, passphrase string) (store.AppImportData, []store.ProfileConflict, []store.TemplateConflict, error) {
	imported, err := a.importBundleBytes(raw, includeSecrets, passphrase)
	if err != nil {
		return store.AppImportData{}, nil, nil, err
	}
	profileConflicts := store.DetectProfileConflicts(a.Profiles(), imported.Profiles)
	templateConflicts := store.DetectTemplateConflicts(a.Templates(), imported.Templates)
	return imported, profileConflicts, templateConflicts, nil
}

func (a *App) ImportAppDataBytes(raw []byte, includeSecrets bool, passphrase string) error {
	imported, err := a.importBundleBytes(raw, includeSecrets, passphrase)
	if err != nil {
		return err
	}
	return a.applyImportedAppData(imported)
}

func (a *App) importBundleBytes(raw []byte, includeSecrets bool, passphrase string) (store.AppImportData, error) {
	f, err := os.CreateTemp("", "dback-import-*.json")
	if err != nil {
		return store.AppImportData{}, err
	}
	path := f.Name()
	defer os.Remove(path)
	if _, err := f.Write(raw); err != nil {
		f.Close()
		return store.AppImportData{}, err
	}
	if err := f.Close(); err != nil {
		return store.AppImportData{}, err
	}
	return a.store.ImportAppDataBundle(path, includeSecrets, passphrase)
}
