package app

import (
	"dback/internal/store"
	"dback/models"
)

func (a *App) collectAppImportData() store.AppImportData {
	destinations, _ := a.store.LoadRemoteDestinations()
	appDestID, _ := a.store.AppSettingsDestinationID()
	syncSettings, _ := a.SyncSettings()
	var syncPtr *models.SyncSettings
	if syncSettings != nil {
		syncPtr = syncSettings
	}
	tasks, _ := a.ListTasks()
	users, _ := a.ListUsers()
	notifyChannels, _ := a.store.ListNotifyChannels()
	authSettings, _ := a.GetAuthSettingsFull()
	squidProxies, _ := a.ListSquidProxies()
	squidSettings, _ := a.GetSquidSettings()
	taskRuns, _ := a.ListTaskRuns("", 10_000)

	importDest := map[string]string{}
	for _, p := range a.Profiles() {
		if dest := a.store.ImportDestForProfile(p.ID); dest != "" {
			importDest[p.ID] = dest
		}
	}

	return store.AppImportData{
		Profiles:                 a.Profiles(),
		Templates:                a.Templates(),
		History:                  a.History(),
		Logs:                     a.Logs(),
		Sync:                     syncPtr,
		RemoteDestinations:       destinations,
		AppSettingsDestinationID: appDestID,
		Tasks:                    tasks,
		TaskRuns:                 taskRuns,
		NotifyChannels:           notifyChannels,
		Users:                    users,
		AuthSettings:             &authSettings,
		SquidProxies:             squidProxies,
		SquidSettings:            &squidSettings,
		ImportDestByProfile:      importDest,
	}
}
