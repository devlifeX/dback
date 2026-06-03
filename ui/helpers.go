package ui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"dback/internal/paths"
	"dback/models"

	"gioui.org/widget"
)

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func setEditorText(e *widget.Editor, text string) {
	e.SetText(text)
}

func editorText(e *widget.Editor) string {
	return e.Text()
}

func defaultProfile() models.Profile {
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	dest := paths.DefaultBackupDestination()
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

func importableProfiles(profiles []models.Profile) []models.Profile {
	var out []models.Profile
	for _, p := range profiles {
		if p.AllowsImport() {
			out = append(out, p)
		}
	}
	return out
}

func hostConnectionSubtitle(p models.Profile) string {
	var conn string
	switch p.ConnectionType {
	case models.ConnectionTypeLocalhost:
		conn = "Localhost"
	case models.ConnectionTypeWordPress:
		site := strings.TrimSpace(p.WPUrl)
		if site == "" {
			site = p.Host
		}
		conn = site
	case models.ConnectionTypeJumpHost:
		conn = fmt.Sprintf("%s@%s:%s (via %s)", p.SSHUser, p.Host, p.Port, p.JumpHost)
	default:
		conn = fmt.Sprintf("%s@%s:%s", p.SSHUser, p.Host, p.Port)
	}
	subtitle := fmt.Sprintf("%s — %s", conn, p.TargetDBName)
	if p.IsDocker {
		subtitle += " (Docker)"
	}
	return subtitle
}
