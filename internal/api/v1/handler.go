package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"dback/internal/app"
	"dback/internal/audit"
	"dback/internal/config"
	"dback/internal/controlplane"
	"dback/internal/metrics"
	"dback/internal/notify"
	"dback/internal/trigger"
	"dback/models"
)

type Handler struct {
	App      *app.App
	CP       *controlplane.Service
	Triggers *trigger.Registry
	Notify   *notify.Router
	Audit    *audit.Writer
	Metrics  *metrics.Collector
	Cfg      config.Config
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ListMeta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type Paginated struct {
	Items any      `json:"items"`
	Meta  ListMeta `json:"meta"`
	ETag  string   `json:"etag,omitempty"`
}

func parseLimitOffset(r *http.Request, defaultLimit, maxLimit int) (limit, offset int) {
	limit = queryInt(r, "limit", defaultLimit)
	offset = queryInt(r, "offset", 0)
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func queryInt(r *http.Request, key string, fallback int) int {
	v := strings.TrimSpace(r.URL.Query().Get(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func etagFor(revision uint64) string {
	return fmt.Sprintf(`W/"%d"`, revision)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ErrorBody{Code: code, Message: message})
}

func setRevisionETag(w http.ResponseWriter, revision uint64) {
	w.Header().Set("ETag", etagFor(revision))
}

func checkMutationPrecondition(w http.ResponseWriter, r *http.Request, revision uint64) bool {
	ifMatch := strings.TrimSpace(r.Header.Get("If-Match"))
	if ifMatch == "" {
		return true
	}
	if ifMatch != etagFor(revision) {
		writeError(w, http.StatusPreconditionFailed, "precondition_failed", "If-Match does not match current revision")
		return false
	}
	return true
}

func requireVaultUnlocked(w http.ResponseWriter, app *app.App) bool {
	if !app.IsUnlocked() {
		writeError(w, http.StatusServiceUnavailable, "vault_locked", "vault is locked")
		return false
	}
	return true
}

func paginateSlice[T any](items []T, limit, offset int) ([]T, ListMeta) {
	total := len(items)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return items[offset:end], ListMeta{Total: total, Limit: limit, Offset: offset}
}

type OperationDTO struct {
	ID         string                 `json:"id"`
	Kind       string                 `json:"kind"`
	ProfileID  string                 `json:"profile_id"`
	TriggerRef string                 `json:"trigger_ref,omitempty"`
	Status     string                 `json:"status"`
	StartedAt  time.Time              `json:"started_at,omitempty"`
	FinishedAt time.Time              `json:"finished_at,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Progress   string                 `json:"progress,omitempty"`
	Artifacts  []operationArtifactDTO `json:"artifacts,omitempty"`
}

type operationArtifactDTO struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`
	Path string `json:"path,omitempty"`
}

func operationFromRecord(rec *controlplane.OperationRecord) OperationDTO {
	if rec == nil {
		return OperationDTO{}
	}
	dto := OperationDTO{
		ID:         rec.ID,
		Kind:       string(rec.Kind),
		ProfileID:  rec.ProfileID,
		TriggerRef: rec.TriggerRef,
		Status:     string(rec.Status),
		StartedAt:  rec.StartedAt,
		FinishedAt: rec.FinishedAt,
		Error:      rec.Error,
		Progress:   rec.Progress,
	}
	for _, a := range rec.Artifacts {
		dto.Artifacts = append(dto.Artifacts, operationArtifactDTO{
			Type: string(a.Type),
			ID:   a.ID,
			Path: a.Path,
		})
	}
	return dto
}

type HostDTO struct {
	models.Profile
}

func hostFromModel(p models.Profile) HostDTO {
	p.SSHPassword = ""
	p.AuthKeyPEM = ""
	p.JumpPassword = ""
	p.JumpAuthKeyPEM = ""
	p.DBPassword = ""
	p.WPKey = ""
	return HostDTO{Profile: p}
}

func redactDestination(d models.RemoteDestination) models.RemoteDestination {
	if d.S3 != nil {
		cp := d.S3.Clone()
		cp.SecretKey = ""
		d.S3 = cp
	}
	return d
}
