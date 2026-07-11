package v1

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func spaFileServer(root string) http.Handler {
	cleanRoot := filepath.Clean(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/health/") {
			http.NotFound(w, r)
			return
		}
		rel := strings.TrimPrefix(r.URL.Path, "/")
		if rel == "" {
			rel = "index.html"
		}
		full := filepath.Join(cleanRoot, filepath.Clean(rel))
		if !withinDir(cleanRoot, full) {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		if _, err := os.Stat(full); os.IsNotExist(err) {
			http.ServeFile(w, r, filepath.Join(cleanRoot, "index.html"))
			return
		}
		http.ServeFile(w, r, full)
	})
}

func withinDir(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
