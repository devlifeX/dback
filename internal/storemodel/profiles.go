package storemodel

import (
	"fmt"
	"time"

	"dback/models"
)

func seedTemplates() []models.SQLTemplate {
	now := time.Now()
	return []models.SQLTemplate{
		{
			ID:          "seed-recreate-db",
			Name:        "Recreate database",
			Description: "Drop and recreate target database",
			Body:        "DROP DATABASE IF EXISTS {databasename};\nCREATE DATABASE {databasename};",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "seed-create-admin",
			Name:        "Create admin user",
			Description: "Create admin user devlife",
			Body:        sqlTemplateCreateAdminUser,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}
}

const sqlTemplateCreateAdminUser = `INSERT INTO wp_users
(user_login, user_pass, user_nicename, user_email, user_registered, user_status, display_name)
VALUES ('devlife', MD5('devlife'), 'devlife', 'devlife@example.com', NOW(), 0, 'devlife');

DELETE FROM wp_usermeta WHERE user_id IN (SELECT ID FROM (SELECT ID FROM wp_users WHERE user_login = 'devlife') t);

INSERT INTO wp_usermeta (user_id, meta_key, meta_value)
SELECT ID, 'wp_capabilities', 'a:1:{s:13:"administrator";b:1;}' FROM wp_users WHERE user_login = 'devlife';

INSERT INTO wp_usermeta (user_id, meta_key, meta_value)
SELECT ID, 'wp_user_level', '10' FROM wp_users WHERE user_login = 'devlife';`

func flattenProfiles(profiles []models.Profile) []models.Profile {
	var out []models.Profile
	for _, p := range profiles {
		out = append(out, flattenProfile(p)...)
	}
	return out
}

func flattenProfile(p models.Profile) []models.Profile {
	p = normalizeProfile(p)

	if p.ExportSettings != nil || p.ImportSettings != nil {
		export := p
		importP := p
		if p.ExportSettings != nil {
			p.ExportSettings.MigrateQueryFields()
			export = p.ApplySettings(p.ExportSettings)
			export.ID = p.ID
			export.Name = p.Name
			export.Group = p.Group
		}
		if p.ImportSettings != nil {
			p.ImportSettings.MigrateQueryFields()
			importP = p.ApplySettings(p.ImportSettings)
			importP.ID = p.ID
			importP.Name = p.Name
			importP.Group = p.Group
			importP.PreImportQuery = p.ImportSettings.PreImportQuery
			importP.RunQueryBeforeImport = p.ImportSettings.RunQueryBeforeImport
			importP.PostImportQuery = p.ImportSettings.PostImportQuery
			importP.RunQueryAfterImport = p.ImportSettings.RunQueryAfterImport
		}

		if models.SettingsEqual(p.ExportSettings, p.ImportSettings) {
			host := normalizeProfile(export)
			host.ExportSettings = nil
			host.ImportSettings = nil
			return []models.Profile{host}
		}

		exportHost := normalizeProfile(export)
		exportHost.ExportSettings = nil
		exportHost.ImportSettings = nil

		importHost := normalizeProfile(importP)
		importHost.ID = fmt.Sprintf("%s-import", p.ID)
		importHost.Name = p.Name + " (import)"
		importHost.ExportSettings = nil
		importHost.ImportSettings = nil
		return []models.Profile{exportHost, importHost}
	}

	p.ExportSettings = nil
	p.ImportSettings = nil
	return []models.Profile{normalizeProfile(p)}
}

func normalizeProfile(p models.Profile) models.Profile {
	if p.ID == "" {
		p.ID = p.Name
	}
	if p.Group == "" {
		p.Group = "Default"
	}
	if p.ConnectionType == models.ConnectionTypeWordPress {
		if p.WPUrl != "" && p.Host == "" {
			p.Host = p.WPUrl
		}
		if p.WPUrl == "" && p.Host != "" {
			p.WPUrl = p.Host
		}
		if p.DBType == "" {
			p.DBType = models.DBTypeMySQL
		}
		return p
	}
	if p.Port == "" {
		p.Port = "22"
	}
	if p.ConnectionType == "" {
		p.ConnectionType = models.ConnectionTypeSSH
	}
	if p.AuthType == "" {
		p.AuthType = models.AuthTypePassword
	}
	if p.JumpPort == "" {
		p.JumpPort = "22"
	}
	if p.JumpAuthType == "" {
		p.JumpAuthType = models.AuthTypePassword
	}
	if p.DBType == "" {
		p.DBType = models.DBTypeMySQL
	}
	p.DBType = normalizeDBType(p.DBType)
	if p.DBHost == "" {
		p.DBHost = "127.0.0.1"
	}
	if p.DBPort == "" {
		p.DBPort = "3306"
	}
	p.FileBackupCompression = models.NormalizeArchiveCompression(p.FileBackupCompression)
	for i := range p.FileBackupPaths {
		_ = p.FileBackupPaths[i].Normalize()
	}
	return p
}

func normalizeDBType(t models.DBType) models.DBType {
	switch t {
	case models.DBTypeMariaDB:
		return models.DBTypeMariaDB
	default:
		return models.DBTypeMySQL
	}
}

func stripSecrets(profiles []models.Profile) []models.Profile {
	for i := range profiles {
		profiles[i].SSHPassword = ""
		profiles[i].JumpPassword = ""
		profiles[i].DBPassword = ""
		profiles[i].AuthKeyPEM = ""
		profiles[i].JumpAuthKeyPEM = ""
		profiles[i].WPKey = ""
	}
	return profiles
}

func stripRemoteDestinationSecrets(destinations []models.RemoteDestination) []models.RemoteDestination {
	if len(destinations) == 0 {
		return nil
	}
	out := make([]models.RemoteDestination, len(destinations))
	for i, d := range destinations {
		out[i] = d.Clone()
		if out[i].S3 != nil {
			out[i].S3.SecretKey = ""
		}
	}
	return out
}

func cloneRemoteDestinations(src []models.RemoteDestination) []models.RemoteDestination {
	if len(src) == 0 {
		return nil
	}
	out := make([]models.RemoteDestination, len(src))
	for i, d := range src {
		out[i] = d.Clone()
	}
	return out
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func FlattenProfiles(profiles []models.Profile) []models.Profile {
	return flattenProfiles(profiles)
}

func FlattenProfile(p models.Profile) []models.Profile {
	return flattenProfile(p)
}

func MigrateRemoteDestinations(payload *models.AppVaultPayload) bool {
	return migrateRemoteDestinations(payload)
}

func migrateRemoteDestinations(payload *models.AppVaultPayload) bool {
	if payload.RemoteDestinationsMigrated {
		return false
	}
	changed := false
	if len(payload.RemoteDestinations) == 0 && models.SyncSettingsConfigured(payload.Sync) {
		id := fmt.Sprintf("%d", time.Now().UnixNano())
		dest := models.RemoteDestinationFromSyncSettings(id, "Default", *payload.Sync)
		payload.RemoteDestinations = []models.RemoteDestination{dest}
		payload.AppSettingsDestinationID = id
		changed = true
	}
	payload.RemoteDestinationsMigrated = true
	return changed
}
