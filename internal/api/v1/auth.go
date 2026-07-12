package v1

import (
	"context"
	"encoding/json"
	"net/http"

	"dback/internal/app"
	"dback/models"

	"github.com/go-chi/chi/v5"
)

type ctxKey int

const ctxUserIDKey ctxKey = 1

func userIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxUserIDKey).(string)
	return v
}

func (h *Handler) mountUsers(r chi.Router) {
	r.Get("/", h.listUsers)
	r.Post("/", h.createUser)
	r.Get("/settings", h.getUserSettings)
	r.Put("/settings", h.updateUserSettings)
	r.Route("/{id}", func(r chi.Router) {
		r.Get("/", h.getUser)
		r.Put("/", h.updateUser)
		r.Delete("/", h.deleteUser)
	})
}

type loginRequest struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type verifyOTPRequest struct {
	ChallengeID string `json:"challenge_id"`
	Code        string `json:"code"`
}

type loginResponse struct {
	app.LoginResult
	User models.User `json:"user,omitempty"`
}

func (h *Handler) authLogin(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	result, user, err := h.App.Login(r.Context(), req.Phone, req.Password)
	if err != nil {
		switch {
		case err == app.ErrInvalidCredentials:
			writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid phone or password")
		case err == app.ErrUserDisabled:
			writeError(w, http.StatusForbidden, "user_disabled", "user account is disabled")
		case err == app.ErrSMSNotConfigured:
			writeError(w, http.StatusServiceUnavailable, "sms_not_configured", "two-factor is enabled but SMS is not configured")
		default:
			writeError(w, http.StatusBadRequest, "login_failed", err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, loginResponse{LoginResult: result, User: user})
}

func (h *Handler) authVerifyOTP(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	var req verifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	token, user, err := h.App.VerifyOTP(r.Context(), req.ChallengeID, req.Code)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_otp", "invalid or expired verification code")
		return
	}
	writeJSON(w, http.StatusOK, loginResponse{
		LoginResult: app.LoginResult{Token: token},
		User:        user,
	})
}

func (h *Handler) authLogout(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r.Header.Get("Authorization"))
	_ = h.App.Logout(token)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) authMe(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	userID := userIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "session required")
		return
	}
	user, err := h.App.GetUser(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "user not found")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	users, err := h.App.ListUsers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusOK, paginatedResponse(users, ListMeta{Total: len(users)}))
}

func (h *Handler) getUser(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	user, err := h.App.GetUser(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "user not found")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

type createUserRequest struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if !checkMutationPrecondition(w, r, h.App.DataRevision()) {
		return
	}
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	user, err := h.App.CreateUser(req.Phone, req.Password, req.Name)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_user", userPublicMessage(err))
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusCreated, user)
}

type updateUserRequest struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Enabled  *bool  `json:"enabled"`
}

func (h *Handler) updateUser(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if !checkMutationPrecondition(w, r, h.App.DataRevision()) {
		return
	}
	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	user, err := h.App.UpdateUser(chi.URLParam(r, "id"), req.Phone, req.Password, req.Name, req.Enabled)
	if err != nil {
		if app.IsUserNotFound(err) {
			writeError(w, http.StatusNotFound, "not_found", "user not found")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_user", userPublicMessage(err))
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if !checkMutationPrecondition(w, r, h.App.DataRevision()) {
		return
	}
	if err := h.App.DeleteUser(chi.URLParam(r, "id")); err != nil {
		if app.IsUserNotFound(err) {
			writeError(w, http.StatusNotFound, "not_found", "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getUserSettings(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	settings, err := h.App.GetAuthSettings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusOK, settings)
}

func (h *Handler) updateUserSettings(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if !checkMutationPrecondition(w, r, h.App.DataRevision()) {
		return
	}
	var settings models.AuthSettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	if err := h.App.SaveAuthSettings(settings); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_settings", err.Error())
		return
	}
	saved, err := h.App.GetAuthSettings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusOK, saved)
}

func userPublicMessage(err error) string {
	return appUserPublicMessage(err)
}

func appUserPublicMessage(err error) string {
	switch {
	case err == app.ErrInvalidPhone:
		return "invalid phone number (expected 09XXXXXXXXX)"
	case err == app.ErrInvalidPassword:
		return "password must be at least 6 characters"
	case err == app.ErrUserExists:
		return "user with this phone already exists"
	default:
		return err.Error()
	}
}

func authRateLimitMiddleware(rps float64, burst int) func(http.Handler) http.Handler {
	lim := newIPLimiter(rps, burst)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !lim.allow(clientIP(r)) {
				writeError(w, http.StatusTooManyRequests, "rate_limited", "too many authentication attempts")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}