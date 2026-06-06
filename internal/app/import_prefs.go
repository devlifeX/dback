package app

// ImportDestForProfile returns the last chosen import destination for a source host.
func (a *App) ImportDestForProfile(sourceProfileID string) string {
	return a.store.ImportDestForProfile(sourceProfileID)
}

// SetImportDestForProfile remembers the import destination for a source host.
func (a *App) SetImportDestForProfile(sourceProfileID, destProfileID string) error {
	return a.store.SetImportDestForProfile(sourceProfileID, destProfileID)
}
