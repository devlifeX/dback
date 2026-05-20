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
	dest := defaultBackupDir(DesktopPlatform{})
	return models.Profile{
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
		Destination:    dest,
	}
}

func appendTemplateSQL(current, templateBody string, vars models.QueryVars) string {
	sql := models.SubstituteQuery(templateBody, vars)
	sql = strings.TrimSpace(sql)
	current = strings.TrimSpace(current)
	if current == "" {
		return sql
	}
	if sql == "" {
		return current
	}
	return current + "\n\n" + sql
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
