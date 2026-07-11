package v1

import (
	"encoding/json"
	"net/http"

	"dback/models"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) mountHosts(r chi.Router) {
	r.Get("/", h.listHosts)
	r.Post("/", h.createHost)
	r.Route("/{id}", func(r chi.Router) {
		r.Get("/", h.getHost)
		r.Put("/", h.updateHost)
		r.Delete("/", h.deleteHost)
		r.Post("/test-connection", h.testHostConnection)
	})
}

func (h *Handler) listHosts(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	profiles := h.App.Profiles()
	items := make([]HostDTO, 0, len(profiles))
	for _, p := range profiles {
		items = append(items, hostFromModel(p))
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusOK, Paginated{Items: items, Meta: ListMeta{Total: len(items)}})
}

func (h *Handler) getHost(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	for _, p := range h.App.Profiles() {
		if p.ID == chi.URLParam(r, "id") {
			setRevisionETag(w, h.App.DataRevision())
			writeJSON(w, http.StatusOK, hostFromModel(p))
			return
		}
	}
	writeError(w, http.StatusNotFound, "not_found", "host not found")
}

func (h *Handler) createHost(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if !checkMutationPrecondition(w, r, h.App.DataRevision()) {
		return
	}
	var profile models.Profile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	if err := h.App.SaveProfile(profile); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_host", err.Error())
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusCreated, hostFromModel(profile))
}

func (h *Handler) updateHost(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if !checkMutationPrecondition(w, r, h.App.DataRevision()) {
		return
	}
	var profile models.Profile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	profile.ID = chi.URLParam(r, "id")
	if err := h.App.SaveProfile(profile); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_host", err.Error())
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusOK, hostFromModel(profile))
}

func (h *Handler) deleteHost(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if !checkMutationPrecondition(w, r, h.App.DataRevision()) {
		return
	}
	if err := h.App.DeleteProfile(chi.URLParam(r, "id")); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "host not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) testHostConnection(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	var profile models.Profile
	for _, p := range h.App.Profiles() {
		if p.ID == chi.URLParam(r, "id") {
			profile = p
			break
		}
	}
	if profile.ID == "" {
		writeError(w, http.StatusNotFound, "not_found", "host not found")
		return
	}
	if err := h.App.TestConnection(profile); err != nil {
		writeError(w, http.StatusBadGateway, "connection_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
