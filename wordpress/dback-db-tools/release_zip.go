package dbackdbtools

import (
	"path"
	"strings"
)

// ReleaseZipExcludeNames are exact paths (or basenames) never shipped in the
// downloadable WordPress plugin zip from DBack.
var ReleaseZipExcludeNames = map[string]struct{}{
	"embed.go":           {},
	"release_zip.go":     {},
	"wordpress_agent.md": {},
}

// ReleaseZipExcludeExtensions are file extensions omitted from the release zip
// (dev docs, Go sources, etc.).
var ReleaseZipExcludeExtensions = map[string]struct{}{
	".md": {},
	".go": {},
}

// IncludeInReleaseZip reports whether an embedded plugin file should be packed
// into the user-facing download zip.
func IncludeInReleaseZip(relPath string) bool {
	relPath = filepathToSlash(strings.TrimPrefix(relPath, "./"))
	if relPath == "" {
		return false
	}

	base := path.Base(relPath)
	if _, ok := ReleaseZipExcludeNames[relPath]; ok {
		return false
	}
	if _, ok := ReleaseZipExcludeNames[base]; ok {
		return false
	}

	ext := strings.ToLower(path.Ext(relPath))
	if _, ok := ReleaseZipExcludeExtensions[ext]; ok {
		return false
	}

	return true
}

func filepathToSlash(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}
