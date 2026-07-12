package v1

import (
	"encoding/json"
	"net/http"

	"dback/internal/app"
	"dback/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handler) mountSquid(r chi.Router) {
	r.Get("/settings", h.getSquidSettings)
	r.Put("/settings", h.saveSquidSettings)
	r.Get("/proxies", h.listSquidProxies)
	r.Post("/proxies", h.createSquidProxy)
	r.Route("/proxies/{id}", func(r chi.Router) {
		r.Get("/", h.getSquidProxy)
		r.Put("/", h.updateSquidProxy)
		r.Delete("/", h.deleteSquidProxy)
	})
}

func (h *Handler) listSquidProxies(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	proxies, err := h.App.ListSquidProxies()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusOK, paginatedResponse(proxies, ListMeta{Total: len(proxies)}))
}

func (h *Handler) getSquidProxy(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	p, err := h.App.GetSquidProxy(chi.URLParam(r, "id"))
	if err != nil {
		if app.IsSquidProxyNotFound(err) {
			writeError(w, http.StatusNotFound, "not_found", "squid proxy not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) createSquidProxy(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if !checkMutationPrecondition(w, r, h.App.DataRevision()) {
		return
	}
	var p models.SquidProxy
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	if err := h.App.SaveSquidProxy(p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_proxy", err.Error())
		return
	}
	saved, err := h.App.GetSquidProxy(p.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusCreated, saved)
}

func (h *Handler) updateSquidProxy(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if !checkMutationPrecondition(w, r, h.App.DataRevision()) {
		return
	}
	id := chi.URLParam(r, "id")
	if _, err := h.App.GetSquidProxy(id); err != nil {
		if app.IsSquidProxyNotFound(err) {
			writeError(w, http.StatusNotFound, "not_found", "squid proxy not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	var p models.SquidProxy
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	p.ID = id
	if err := h.App.SaveSquidProxy(p); err != nil {
		if app.IsSquidProxyNotFound(err) {
			writeError(w, http.StatusNotFound, "not_found", "squid proxy not found")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_proxy", err.Error())
		return
	}
	saved, err := h.App.GetSquidProxy(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusOK, saved)
}

func (h *Handler) deleteSquidProxy(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if !checkMutationPrecondition(w, r, h.App.DataRevision()) {
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.App.DeleteSquidProxy(id); err != nil {
		if app.IsSquidProxyNotFound(err) {
			writeError(w, http.StatusNotFound, "not_found", "squid proxy not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getSquidSettings(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	settings, err := h.App.GetSquidSettings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusOK, settings)
}

func (h *Handler) saveSquidSettings(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if !checkMutationPrecondition(w, r, h.App.DataRevision()) {
		return
	}
	var settings models.SquidSettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	if err := h.App.SaveSquidSettings(settings); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_settings", err.Error())
		return
	}
	saved, err := h.App.GetSquidSettings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusOK, saved)
}
