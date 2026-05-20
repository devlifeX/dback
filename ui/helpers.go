package ui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"dback/models"
)

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func defaultProfile() models.Profile {
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	p := models.Profile{
		ID:             id,
		Name:           "New Host",
		Group:          "Default",
		ConnectionType: models.ConnectionTypeSSH,
		Port:           "22",
		AuthType:       models.AuthTypePassword,
		JumpPort:       "22",
		JumpAuthType:   models.AuthTypePassword,
		DBHost:         "127.0.0.1",
		DBPort:         "3306",
		DBType:         models.DBTypeMySQL,
		Destination:    ".",
	}
	settings := models.SettingsFromProfile(p)
	p.ExportSettings = &settings
	p.ImportSettings = &settings
	return p
}

func withLegacy(p models.Profile, settings models.TransferSettings) models.Profile {
	p.ConnectionType = settings.ConnectionType
	p.Host = settings.Host
	p.Port = settings.Port
	p.SSHUser = settings.SSHUser
	p.SSHPassword = settings.SSHPassword
	p.AuthType = settings.AuthType
	p.AuthKeyPath = settings.AuthKeyPath
	p.AuthKeyPEM = settings.AuthKeyPEM
	p.JumpHost = settings.JumpHost
	p.JumpPort = settings.JumpPort
	p.JumpUser = settings.JumpUser
	p.JumpPassword = settings.JumpPassword
	p.JumpAuthType = settings.JumpAuthType
	p.JumpAuthKeyPath = settings.JumpAuthKeyPath
	p.JumpAuthKeyPEM = settings.JumpAuthKeyPEM
	p.WPUrl = settings.WPUrl
	p.WPKey = settings.WPKey
	p.DBHost = settings.DBHost
	p.DBPort = settings.DBPort
	p.DBUser = settings.DBUser
	p.DBPassword = settings.DBPassword
	p.DBType = settings.DBType
	p.IsDocker = settings.IsDocker
	p.ContainerID = settings.ContainerID
	p.TargetDBName = settings.TargetDBName
	p.Destination = settings.Destination
	p.PreImportQuery = settings.PreImportQuery
	p.RunQueryBeforeImport = settings.RunQueryBeforeImport
	p.PostImportQuery = settings.PostImportQuery
	p.RunQueryAfterImport = settings.RunQueryAfterImport
	return p
}

func ptrSettings(s models.TransferSettings) *models.TransferSettings {
	return &s
}

func mergeImportQuerySettings(importSettings, query models.TransferSettings) models.TransferSettings {
	importSettings.PreImportQuery = query.PreImportQuery
	importSettings.RunQueryBeforeImport = query.RunQueryBeforeImport
	importSettings.PostImportQuery = query.PostImportQuery
	importSettings.RunQueryAfterImport = query.RunQueryAfterImport
	return importSettings
}

func loadKeyFromReader(reader io.ReadCloser) (pem, displayName string, err error) {
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", "", err
	}
	return string(data), "selected key", nil
}

func isDocumentURIPath(path string) bool {
	return strings.HasPrefix(path, "/document/") || strings.Contains(path, "mf%3A")
}
