package storemodel

import "dback/models"

// DestinationUsage describes where a remote destination is referenced.
type DestinationUsage struct {
	UsedForAppSettings bool
	ProfileIDs         []string
	ProfileNames       []string
}

// AppImportData holds decoded app bundle contents for merge preview.
type AppImportData struct {
	Profiles                 []models.Profile
	Templates                []models.SQLTemplate
	History                  []models.ExportRecord
	Logs                     []models.LogEntry
	Sync                     *models.SyncSettings
	RemoteDestinations       []models.RemoteDestination
	AppSettingsDestinationID string
}

// TemplateConflict describes an imported template that replaces an existing one.
type TemplateConflict struct {
	Imported models.SQLTemplate `json:"imported"`
	Existing models.SQLTemplate `json:"existing"`
	Reason   string             `json:"reason"`
}

// ProfileConflict describes an imported host that replaces an existing one.
type ProfileConflict struct {
	Imported models.Profile `json:"imported"`
	Existing models.Profile `json:"existing"`
	Reason   string         `json:"reason"`
}
