package wordpress

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"path"
	"regexp"
	"strings"

	"dback/wordpress/dback-db-tools"
)

var pluginVersionPattern = regexp.MustCompile(`define\('DBACK_DB_TOOLS_VERSION',\s*'([^']+)'\)`)

// BuildPluginZip returns a WordPress plugin zip with the API key hardcoded.
func BuildPluginZip(siteURL, apiKey string) ([]byte, string, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, "", fmt.Errorf("API key is required")
	}

	version, err := embeddedPluginVersion()
	if err != nil {
		return nil, "", err
	}

	hostLabel := hostnameFromSiteURL(siteURL)
	filename := fmt.Sprintf("dback-%s-%s.zip", hostLabel, version)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	err = fs.WalkDir(dbackdbtools.Files, ".", func(relPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if relPath == "embed.go" {
			return nil
		}

		data, err := dbackdbtools.Files.ReadFile(relPath)
		if err != nil {
			return err
		}
		if relPath == "dback-db-tools.php" {
			data = []byte(strings.ReplaceAll(string(data), dbackdbtools.APIKeyPlaceholder, apiKey))
		}

		zipPath := path.Join(dbackdbtools.PluginSlug, filepathToSlash(relPath))
		w, err := zw.Create(zipPath)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, bytes.NewReader(data))
		return err
	})
	if err != nil {
		_ = zw.Close()
		return nil, "", err
	}
	if err := zw.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), filename, nil
}

func filepathToSlash(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}

func embeddedPluginVersion() (string, error) {
	data, err := dbackdbtools.Files.ReadFile("dback-db-tools.php")
	if err != nil {
		return "", fmt.Errorf("read embedded plugin version: %w", err)
	}
	matches := pluginVersionPattern.FindSubmatch(data)
	if len(matches) < 2 {
		return "", fmt.Errorf("plugin version not found in embedded template")
	}
	return string(matches[1]), nil
}

func hostnameFromSiteURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "site"
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return sanitizeFilename(raw)
	}
	host := u.Hostname()
	if host == "" {
		return "site"
	}
	return sanitizeFilename(host)
}

func sanitizeFilename(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	replacer := strings.NewReplacer(
		"https://", "",
		"http://", "",
		"/", "-",
		"\\", "-",
		":", "-",
		" ", "-",
	)
	s = replacer.Replace(s)
	s = regexp.MustCompile(`[^a-z0-9._-]+`).ReplaceAllString(s, "-")
	s = strings.Trim(s, "-.")
	if s == "" {
		return "site"
	}
	return s
}
