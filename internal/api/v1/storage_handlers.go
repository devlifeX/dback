package v1

import (
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) mountStorage(r chi.Router) {
	r.Get("/local", h.listLocalStorage)
	r.Get("/local/download", h.downloadLocalStorage)
	r.Get("/remote", h.listRemoteStorage)
	r.Get("/remote/download", h.downloadRemoteStorage)
}

func (h *Handler) listLocalStorage(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	listing, err := h.App.ListLocalStorage(path)
	if err != nil {
		writeStorageError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, listing)
}

func (h *Handler) downloadLocalStorage(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	if path == "" {
		writeError(w, http.StatusBadRequest, "invalid_path", "path is required")
		return
	}
	f, entry, err := h.App.OpenLocalStorageFile(path)
	if err != nil {
		writeStorageError(w, err)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(entry.Name))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, f)
}

func (h *Handler) listRemoteStorage(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	destID := strings.TrimSpace(r.URL.Query().Get("destination_id"))
	if destID == "" {
		writeError(w, http.StatusBadRequest, "invalid_destination", "destination_id is required")
		return
	}
	prefix := strings.TrimSpace(r.URL.Query().Get("prefix"))
	listing, err := h.App.ListRemoteStorage(r.Context(), destID, prefix)
	if err != nil {
		writeStorageError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, listing)
}

func (h *Handler) downloadRemoteStorage(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	destID := strings.TrimSpace(r.URL.Query().Get("destination_id"))
	key := strings.TrimSpace(r.URL.Query().Get("key"))
	if destID == "" || key == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "destination_id and key are required")
		return
	}
	reader, entry, err := h.App.OpenRemoteStorageObject(r.Context(), destID, key)
	if err != nil {
		writeStorageError(w, err)
		return
	}
	defer reader.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(entry.Name))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, reader)
}

func writeStorageError(w http.ResponseWriter, err error) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "not found"), strings.Contains(msg, "not allowed"), strings.Contains(msg, "invalid"):
		writeError(w, http.StatusBadRequest, "storage_error", msg)
	case strings.Contains(msg, "directory"):
		writeError(w, http.StatusBadRequest, "invalid_path", msg)
	default:
		writeError(w, http.StatusBadGateway, "storage_error", msg)
	}
}
