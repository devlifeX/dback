package v1

import (
	"encoding/json"
	"net/http"
	"strings"

	"dback/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handler) mountTemplates(r chi.Router) {
	r.Get("/", h.listTemplates)
	r.Post("/", h.createTemplate)
	r.Route("/{id}", func(r chi.Router) {
		r.Put("/", h.updateTemplate)
		r.Delete("/", h.deleteTemplate)
	})
}

func (h *Handler) listTemplates(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	templates := h.App.Templates()
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusOK, paginatedResponse(templates, ListMeta{Total: len(templates)}))
}

func (h *Handler) createTemplate(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if !checkMutationPrecondition(w, r, h.App.DataRevision()) {
		return
	}
	var t models.SQLTemplate
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	if err := h.App.SaveTemplate(t); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_template", err.Error())
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusCreated, t)
}

func (h *Handler) updateTemplate(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if !checkMutationPrecondition(w, r, h.App.DataRevision()) {
		return
	}
	var t models.SQLTemplate
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	t.ID = chi.URLParam(r, "id")
	if err := h.App.SaveTemplate(t); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_template", err.Error())
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) deleteTemplate(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if !checkMutationPrecondition(w, r, h.App.DataRevision()) {
		return
	}
	if err := h.App.DeleteTemplate(chi.URLParam(r, "id")); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "template not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) mountBackups(r chi.Router) {
	r.Get("/", h.listBackups)
	r.Route("/{id}", func(r chi.Router) {
		r.Get("/", h.getBackup)
		r.Get("/download", h.downloadBackup)
		r.Post("/verify/quick", h.quickVerifyBackup)
	})
}

func (h *Handler) listBackups(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	limit, offset := parseLimitOffset(r, 50, 500)
	history := h.App.History()
	page, meta := paginateSlice(history, limit, offset)
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusOK, paginatedResponse(page, meta))
}

func (h *Handler) getBackup(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	id := chi.URLParam(r, "id")
	for _, rec := range h.App.History() {
		if rec.ID == id {
			writeJSON(w, http.StatusOK, rec)
			return
		}
	}
	writeError(w, http.StatusNotFound, "not_found", "backup not found")
}

func (h *Handler) quickVerifyBackup(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	result, err := h.App.QuickVerify(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "verify_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) mountDestinations(r chi.Router) {
	r.Get("/", h.listDestinations)
	r.Post("/", h.createDestination)
	r.Route("/{id}", func(r chi.Router) {
		r.Get("/", h.getDestination)
		r.Get("/usage", h.destinationUsage)
		r.Put("/", h.updateDestination)
		r.Delete("/", h.deleteDestination)
		r.Post("/test", h.testDestination)
	})
}

func (h *Handler) getDestination(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	dest, err := h.App.RemoteDestinationByID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusOK, dest)
}

func (h *Handler) listDestinations(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	dests, err := h.App.ListRemoteDestinations()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	items := make([]models.RemoteDestination, 0, len(dests))
	for _, d := range dests {
		items = append(items, redactDestination(d))
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusOK, paginatedResponse(items, ListMeta{Total: len(items)}))
}

func (h *Handler) createDestination(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if !checkMutationPrecondition(w, r, h.App.DataRevision()) {
		return
	}
	var dest models.RemoteDestination
	if err := json.NewDecoder(r.Body).Decode(&dest); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	if err := h.App.SaveRemoteDestination(dest); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_destination", err.Error())
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusCreated, redactDestination(dest))
}

func (h *Handler) updateDestination(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if !checkMutationPrecondition(w, r, h.App.DataRevision()) {
		return
	}
	var dest models.RemoteDestination
	if err := json.NewDecoder(r.Body).Decode(&dest); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	dest.ID = chi.URLParam(r, "id")
	if err := h.App.SaveRemoteDestination(dest); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_destination", err.Error())
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusOK, redactDestination(dest))
}

func (h *Handler) deleteDestination(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if !checkMutationPrecondition(w, r, h.App.DataRevision()) {
		return
	}
	force := strings.EqualFold(r.URL.Query().Get("force"), "true")
	if err := h.App.DeleteRemoteDestination(chi.URLParam(r, "id"), force); err != nil {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) testDestination(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	dest, err := h.App.RemoteDestinationByID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "destination not found")
		return
	}
	if err := h.App.TestRemoteDestination(r.Context(), dest); err != nil {
		writeError(w, http.StatusBadGateway, "test_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) mountNotifications(r chi.Router) {
	r.Get("/", h.listNotifications)
	r.Post("/", h.createNotification)
	r.Route("/{id}", func(r chi.Router) {
		r.Get("/", h.getNotification)
		r.Put("/", h.updateNotification)
		r.Delete("/", h.deleteNotification)
		r.Post("/test", h.testNotification)
	})
}

func (h *Handler) listNotifications(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	channels, err := h.App.ListNotifyChannels()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusOK, paginatedResponse(channels, ListMeta{Total: len(channels)}))
}

func (h *Handler) getNotification(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	ch, err := h.App.GetNotifyChannel(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "notification channel not found")
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusOK, ch)
}

func (h *Handler) createNotification(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if !checkMutationPrecondition(w, r, h.App.DataRevision()) {
		return
	}
	var ch models.NotifyChannel
	if err := json.NewDecoder(r.Body).Decode(&ch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	if ch.ID == "" {
		ch.ID = uuid.NewString()
	}
	if err := h.App.SaveNotifyChannel(ch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_channel", err.Error())
		return
	}
	saved, err := h.App.GetNotifyChannel(ch.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusCreated, saved)
}

func (h *Handler) updateNotification(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if !checkMutationPrecondition(w, r, h.App.DataRevision()) {
		return
	}
	var ch models.NotifyChannel
	if err := json.NewDecoder(r.Body).Decode(&ch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	ch.ID = chi.URLParam(r, "id")
	if err := h.App.SaveNotifyChannel(ch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_channel", err.Error())
		return
	}
	saved, err := h.App.GetNotifyChannel(ch.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusOK, saved)
}

func (h *Handler) deleteNotification(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if !checkMutationPrecondition(w, r, h.App.DataRevision()) {
		return
	}
	if err := h.App.DeleteNotifyChannel(chi.URLParam(r, "id")); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "notification channel not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) testNotification(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if h.Notify == nil {
		writeError(w, http.StatusServiceUnavailable, "notify_unavailable", "notify router not running")
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.Notify.Test(r.Context(), id); err != nil {
		writeError(w, http.StatusBadGateway, "test_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (h *Handler) mountLogs(r chi.Router) {
	r.Get("/", h.listLogs)
}

func (h *Handler) listLogs(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	limit, offset := parseLimitOffset(r, 100, 1000)
	logs := h.App.Logs()
	page, meta := paginateSlice(logs, limit, offset)
	writeJSON(w, http.StatusOK, paginatedResponse(page, meta))
}

func (h *Handler) mountSync(r chi.Router) {
	r.Get("/settings", h.getSyncSettings)
	r.Put("/settings", h.putSyncSettings)
	r.Post("/push", h.syncPush)
	r.Post("/pull", h.syncPull)
	r.Post("/test", h.syncTestConnection)
	r.Get("/activity", h.syncActivity)
	r.Post("/preview", h.syncImportPreview)
	r.Post("/import", h.syncImportApply)
}

func (h *Handler) getSyncSettings(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	settings, err := h.App.SyncSettings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (h *Handler) putSyncSettings(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if !checkMutationPrecondition(w, r, h.App.DataRevision()) {
		return
	}
	var settings models.SyncSettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	if err := h.App.SaveSyncSettings(settings); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_sync", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (h *Handler) syncPush(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if err := h.App.SyncPush(r.Context()); err != nil {
		writeError(w, http.StatusBadGateway, "sync_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "pushed"})
}

func (h *Handler) syncPull(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	raw, err := h.App.SyncDownload(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "sync_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"bytes": len(raw)})
}

func (h *Handler) mountSystem(r chi.Router) {
	r.Get("/version", h.systemVersion)
	r.Get("/revision", h.systemRevision)
	r.Get("/audit", h.listAudit)
	r.Get("/storage", h.systemStorage)
	r.Get("/server-info", h.systemServerInfo)
}

func (h *Handler) systemVersion(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"version": "phase5",
		"api":     "v1",
	})
}

func (h *Handler) systemRevision(w http.ResponseWriter, _ *http.Request) {
	rev := h.App.DataRevision()
	setRevisionETag(w, rev)
	writeJSON(w, http.StatusOK, map[string]any{"revision": rev})
}

func (h *Handler) listAudit(w http.ResponseWriter, r *http.Request) {
	if h.Audit == nil {
		writeJSON(w, http.StatusOK, paginatedResponse([]any{}, ListMeta{}))
		return
	}
	limit, _ := parseLimitOffset(r, 100, 500)
	entries := h.Audit.List(limit)
	writeJSON(w, http.StatusOK, paginatedResponse(entries, ListMeta{Total: len(entries), Limit: limit}))
}
