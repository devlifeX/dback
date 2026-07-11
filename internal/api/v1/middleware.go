package v1

import (
	"crypto/subtle"
	"net/http"
	"strings"
	"time"
)

const maxBodyBytes = 1 << 20
const maxImportBodyBytes = 64 << 20

func (h *Handler) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimSpace(h.Cfg.APIToken) == "" {
			writeError(w, http.StatusServiceUnavailable, "api_token_unconfigured", "set DBACK_API_TOKEN or DBACK_API_TOKEN_FILE")
			return
		}
		token := bearerToken(r.Header.Get("Authorization"))
		if token == "" {
			token = strings.TrimSpace(r.URL.Query().Get("access_token"))
		}
		if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(h.Cfg.APIToken)) != 1 {
			writeError(w, http.StatusUnauthorized, "unauthorized", "invalid or missing bearer token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

func jsonOnlyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			ct := strings.ToLower(r.Header.Get("Content-Type"))
			isJSON := strings.HasPrefix(ct, "application/json")
			isMultipart := strings.HasPrefix(ct, "multipart/form-data")
			if !isJSON && !isMultipart {
				writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json")
				return
			}
			limit := int64(maxBodyBytes)
			if isMultipart || strings.HasPrefix(r.URL.Path, "/api/v1/import") {
				limit = maxImportBodyBytes
			}
			r.Body = http.MaxBytesReader(w, r.Body, limit)
		}
		next.ServeHTTP(w, r)
	})
}

func timeoutMiddleware(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, d, `{"code":"timeout","message":"request timed out"}`)
	}
}
