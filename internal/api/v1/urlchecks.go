package v1

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) listHostURLChecks(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	profileID := chi.URLParam(r, "id")
	found := false
	for _, p := range h.App.Profiles() {
		if p.ID == profileID {
			found = true
			break
		}
	}
	if !found {
		writeError(w, http.StatusNotFound, "not_found", "host not found")
		return
	}

	urlFilter := r.URL.Query().Get("url")
	var from, to time.Time
	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			from = t
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			to = t
		}
	}
	if from.IsZero() {
		from = time.Now().UTC().Add(-7 * 24 * time.Hour)
	}

	buckets, err := h.App.ListURLCheckHourly(profileID, urlFilter, from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "url_checks_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, paginatedResponse(buckets, ListMeta{Total: len(buckets)}))
}
