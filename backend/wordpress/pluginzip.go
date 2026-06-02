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
	"unicode"

	"dback/wordpress/dback-db-tools"
)

var pluginVersionPattern = regexp.MustCompile(`define\('DBACK_DB_TOOLS_VERSION',\s*'([^']+)'\)`)
const releaseZipRootFolder = "dback-db-tools"

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
	filename := sanitizeDownloadFilename(fmt.Sprintf("dback-%s-%s.zip", hostLabel, version))

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	err = fs.WalkDir(dbackdbtools.Files, ".", func(relPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !dbackdbtools.IncludeInReleaseZip(relPath) {
			return nil
		}

		data, err := dbackdbtools.Files.ReadFile(relPath)
		if err != nil {
			return err
		}
		if relPath == "dback-db-tools.php" {
			data = []byte(strings.ReplaceAll(string(data), dbackdbtools.APIKeyPlaceholder, apiKey))
		}

		// Keep a stable plugin directory so extracting newer versions overwrites the
		// existing wp-content/plugins/dback-db-tools folder instead of creating a new one.
		zipPath := path.Join(releaseZipRootFolder, filepathToSlash(relPath))
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
		"*", "-",
		"?", "-",
		"\"", "-",
		"<", "-",
		">", "-",
		"|", "-",
	)
	s = replacer.Replace(s)
	s = regexp.MustCompile(`[^a-z0-9._-]+`).ReplaceAllString(s, "-")
	s = strings.Trim(s, "-.")
	if s == "" {
		return "site"
	}
	if len(s) > 120 {
		s = strings.Trim(s[:120], "-.")
	}
	if isWindowsReservedName(s) {
		s = "site-" + s
	}
	return s
}

var windowsReservedNames = map[string]struct{}{
	"con": {}, "prn": {}, "aux": {}, "nul": {},
	"com1": {}, "com2": {}, "com3": {}, "com4": {}, "com5": {}, "com6": {}, "com7": {}, "com8": {}, "com9": {},
	"lpt1": {}, "lpt2": {}, "lpt3": {}, "lpt4": {}, "lpt5": {}, "lpt6": {}, "lpt7": {}, "lpt8": {}, "lpt9": {},
}

func isWindowsReservedName(name string) bool {
	base := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(name)), ".")
	if i := strings.Index(base, "."); i >= 0 {
		base = base[:i]
	}
	_, ok := windowsReservedNames[base]
	return ok
}

// sanitizeDownloadFilename makes a suggested save-as name safe on Windows/macOS/Linux.
func sanitizeDownloadFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "dback-plugin.zip"
	}
	name = strings.Map(func(r rune) rune {
		if r < 32 || unicode.IsControl(r) {
			return -1
		}
		return r
	}, name)
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, ":", "-")
	name = strings.ReplaceAll(name, "*", "-")
	name = strings.ReplaceAll(name, "?", "-")
	name = strings.ReplaceAll(name, "\"", "-")
	name = strings.ReplaceAll(name, "<", "-")
	name = strings.ReplaceAll(name, ">", "-")
	name = strings.ReplaceAll(name, "|", "-")
	name = strings.TrimRight(name, ". ")
	if !strings.HasSuffix(strings.ToLower(name), ".zip") {
		name += ".zip"
	}
	if len(name) > 200 {
		ext := ".zip"
		name = strings.TrimRight(name[:200-len(ext)], ". ") + ext
	}
	base := strings.TrimSuffix(strings.ToLower(name), ".zip")
	if isWindowsReservedName(base) {
		name = "dback-" + name
	}
	return name
}

func sanitizeZipRootFolder(folder string) string {
	folder = strings.TrimSpace(folder)
	folder = strings.TrimRight(folder, ". ")
	if folder == "" {
		return "dback-plugin"
	}
	if isWindowsReservedName(folder) {
		return "dback-" + folder
	}
	return folder
}
