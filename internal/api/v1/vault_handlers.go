package v1

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"dback/models"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) mountVaultExport(r chi.Router) {
	r.Get("/app-data", h.exportAppData)
}

func (h *Handler) mountVaultImport(r chi.Router) {
	r.Post("/preview", h.importPreview)
	r.Post("/apply", h.importApply)
}

type importBundleRequest struct {
	Passphrase      string `json:"passphrase"`
	IncludeSecrets  bool   `json:"include_secrets"`
	ContentBase64   string `json:"content_base64,omitempty"`
	EncryptedBundle string `json:"encrypted_bundle,omitempty"`
}

func (h *Handler) exportAppData(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	passphrase := strings.TrimSpace(r.URL.Query().Get("passphrase"))
	includeSecrets := !strings.EqualFold(r.URL.Query().Get("include_secrets"), "false")
	raw, err := h.App.ExportAppDataBytes(includeSecrets, passphrase)
	if err != nil {
		writeError(w, http.StatusBadRequest, "export_failed", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="dback-app-data.json"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func (h *Handler) importPreview(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	raw, passphrase, includeSecrets, err := readImportBundle(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_import", err.Error())
		return
	}
	imported, profileConflicts, templateConflicts, err := h.App.PreviewImportAppDataBytes(raw, includeSecrets, passphrase)
	if err != nil {
		writeError(w, http.StatusBadRequest, "import_preview_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"profiles_count":     len(imported.Profiles),
		"templates_count":    len(imported.Templates),
		"history_count":      len(imported.History),
		"profile_conflicts":  profileConflicts,
		"template_conflicts": templateConflicts,
	})
}

func (h *Handler) importApply(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if !checkMutationPrecondition(w, r, h.App.DataRevision()) {
		return
	}
	raw, passphrase, includeSecrets, err := readImportBundle(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_import", err.Error())
		return
	}
	if err := h.App.ImportAppDataBytes(raw, includeSecrets, passphrase); err != nil {
		writeError(w, http.StatusBadRequest, "import_failed", err.Error())
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusOK, map[string]string{"status": "imported"})
}

func readImportBundle(r *http.Request) (raw []byte, passphrase string, includeSecrets bool, err error) {
	ct := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseMultipartForm(64 << 20); err != nil {
			return nil, "", false, err
		}
		passphrase = strings.TrimSpace(r.FormValue("passphrase"))
		includeSecrets = strings.EqualFold(r.FormValue("include_secrets"), "true")
		file, _, ferr := r.FormFile("file")
		if ferr != nil {
			return nil, "", false, ferr
		}
		defer file.Close()
		raw, err = io.ReadAll(file)
		return raw, passphrase, includeSecrets, err
	}
	var req importBundleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, "", false, err
	}
	passphrase = req.Passphrase
	includeSecrets = req.IncludeSecrets
	if req.ContentBase64 != "" {
		decoded, err := base64.StdEncoding.DecodeString(req.ContentBase64)
		if err != nil {
			return nil, "", false, err
		}
		return decoded, passphrase, includeSecrets, nil
	}
	return []byte(req.EncryptedBundle), passphrase, includeSecrets, nil
}

func (h *Handler) syncTestConnection(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	settings, err := h.App.SyncSettings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if settings == nil {
		writeError(w, http.StatusBadRequest, "sync_not_configured", "sync settings not configured")
		return
	}
	if err := h.App.TestSyncConnection(r.Context(), *settings); err != nil {
		writeError(w, http.StatusBadGateway, "sync_test_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) syncActivity(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	activity, err := h.App.SyncActivity()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, activity)
}

func (h *Handler) syncImportPreview(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	raw, err := h.App.SyncDownload(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "sync_pull_failed", err.Error())
		return
	}
	preview, profileConflicts, templateConflicts, err := h.App.PreviewSyncImport(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "sync_preview_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"bytes":              len(raw),
		"profiles_count":     len(preview.Profiles),
		"templates_count":    len(preview.Templates),
		"profile_conflicts":  profileConflicts,
		"template_conflicts": templateConflicts,
	})
}

func (h *Handler) syncImportApply(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if !checkMutationPrecondition(w, r, h.App.DataRevision()) {
		return
	}
	raw, err := h.App.SyncDownload(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "sync_pull_failed", err.Error())
		return
	}
	preview, _, _, err := h.App.PreviewSyncImport(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "sync_preview_failed", err.Error())
		return
	}
	if err := h.App.ImportAppDataFromBundle(preview); err != nil {
		writeError(w, http.StatusBadRequest, "sync_import_failed", err.Error())
		return
	}
	if err := h.App.RecordSyncPull(); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusOK, map[string]string{"status": "imported"})
}

func (h *Handler) destinationUsage(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	usage, err := h.App.DestinationUsage(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, usage)
}

func (h *Handler) downloadBackup(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	id := chi.URLParam(r, "id")
	var record models.ExportRecord
	for _, rec := range h.App.History() {
		if rec.ID == id {
			record = rec
			break
		}
	}
	if record.ID == "" {
		writeError(w, http.StatusNotFound, "not_found", "backup not found")
		return
	}
	if strings.TrimSpace(record.FilePath) == "" {
		writeError(w, http.StatusNotFound, "file_missing", "backup file path not available")
		return
	}
	f, err := os.Open(record.FilePath)
	if err != nil {
		writeError(w, http.StatusNotFound, "file_missing", "backup file not found on server disk")
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(record.FilePath))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, f)
}

func (h *Handler) systemStorage(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	local := h.App.LocalStorageSummary()
	remote := h.App.RemoteStorageSummary()
	writeJSON(w, http.StatusOK, map[string]any{
		"local": map[string]any{
			"bytes": local.Bytes,
			"files": local.Files,
			"roots": local.Roots,
		},
		"remote": map[string]any{
			"bytes":        remote.Bytes,
			"objects":      remote.Objects,
			"destinations": remote.Destinations,
		},
		"backup_records": len(h.App.History()),
		"backup_bytes":   local.Bytes,
		"hosts":          len(h.App.Profiles()),
	})
}
