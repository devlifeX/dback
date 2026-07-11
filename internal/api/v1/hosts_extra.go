package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"dback/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handler) duplicateHost(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if !checkMutationPrecondition(w, r, h.App.DataRevision()) {
		return
	}
	id := chi.URLParam(r, "id")
	var source models.Profile
	for _, p := range h.App.Profiles() {
		if p.ID == id {
			source = p
			break
		}
	}
	if source.ID == "" {
		writeError(w, http.StatusNotFound, "not_found", "host not found")
		return
	}
	clone := source
	clone.ID = uuid.NewString()
	clone.Name = nextDuplicateHostName(source.Name, h.App.Profiles())
	if err := h.App.SaveProfile(clone); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_host", err.Error())
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusCreated, hostFromModel(clone))
}

func nextDuplicateHostName(name string, profiles []models.Profile) string {
	base := strings.TrimSpace(name)
	if base == "" {
		base = "Host"
	}
	maxN := 1
	prefix := base + " "
	for _, p := range profiles {
		if p.Name == base {
			maxN = max(maxN, 1)
			continue
		}
		if strings.HasPrefix(p.Name, prefix) {
			if n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(p.Name, prefix))); err == nil && n >= maxN {
				maxN = n + 1
			}
		}
	}
	if maxN == 1 {
		for _, p := range profiles {
			if p.Name == base {
				return base + " 2"
			}
		}
		return base
	}
	return fmt.Sprintf("%s %d", base, maxN)
}

func (h *Handler) listPendingUploads(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	count, details, err := h.App.PendingUploads(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "upload_pending_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"count":   count,
		"details": details,
	})
}

type uploadPlanRequest struct {
	RecordIDs []string `json:"record_ids"`
}

func (h *Handler) planProfileUpload(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	var req uploadPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	plan, err := h.App.PrepareProfileUpload(r.Context(), chi.URLParam(r, "id"), req.RecordIDs)
	if err != nil {
		writeError(w, http.StatusBadRequest, "upload_plan_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (h *Handler) getImportDestination(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	destID := h.App.ImportDestForProfile(chi.URLParam(r, "id"))
	writeJSON(w, http.StatusOK, map[string]string{"destination_profile_id": destID})
}

type importDestRequest struct {
	DestinationProfileID string `json:"destination_profile_id"`
}

func (h *Handler) setImportDestination(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	var req importDestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	if err := h.App.SetImportDestForProfile(chi.URLParam(r, "id"), req.DestinationProfileID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type hostQueryRequest struct {
	SQL       string `json:"sql"`
	ConnectDB bool   `json:"connect_db"`
}

func (h *Handler) runHostQuery(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	var req hostQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
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
	result, err := h.App.RunImportQuery(r.Context(), profile, req.SQL, req.ConnectDB)
	if err != nil {
		writeError(w, http.StatusBadGateway, "query_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) downloadWordPressPlugin(w http.ResponseWriter, r *http.Request) {
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
	if !profile.UsesWordPress() {
		writeError(w, http.StatusBadRequest, "not_wordpress", "host is not a WordPress connection")
		return
	}
	if strings.TrimSpace(profile.WPKey) == "" {
		writeError(w, http.StatusBadRequest, "missing_wp_key", "generate a WordPress API key first")
		return
	}
	data, filename, err := h.App.BuildWordPressPluginZip(profile.WPUrl, profile.WPKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "plugin_build_failed", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *Handler) generateWPKey(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if !checkMutationPrecondition(w, r, h.App.DataRevision()) {
		return
	}
	id := chi.URLParam(r, "id")
	var profile models.Profile
	for _, p := range h.App.Profiles() {
		if p.ID == id {
			profile = p
			break
		}
	}
	if profile.ID == "" {
		writeError(w, http.StatusNotFound, "not_found", "host not found")
		return
	}
	key, err := h.App.GenerateWPKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "key_gen_failed", err.Error())
		return
	}
	profile.WPKey = key
	if err := h.App.SaveProfile(profile); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_host", err.Error())
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusOK, map[string]string{"wp_key": key})
}
