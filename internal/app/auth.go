package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"dback/internal/sms"
	"dback/internal/store"
	"dback/models"

	"golang.org/x/crypto/bcrypt"
)

const (
	sessionTTL       = 24 * time.Hour
	maxOTPAttempts   = 5
	authRateLimitKey = "auth"
)

var (
	ErrOTPRequired      = errors.New("otp required")
	ErrOTPInvalid       = errors.New("invalid or expired otp")
	ErrSessionInvalid   = errors.New("invalid or expired session")
	ErrSMSNotConfigured = errors.New("sms provider not configured")
)

func (a *App) GetAuthSettings() (models.AuthSettings, error) {
	settings, err := a.store.LoadAuthSettings()
	if err != nil {
		return models.AuthSettings{}, err
	}
	return redactAuthSettings(settings), nil
}

func (a *App) GetAuthSettingsFull() (models.AuthSettings, error) {
	return a.store.LoadAuthSettings()
}

func (a *App) SaveAuthSettings(incoming models.AuthSettings) error {
	existing, err := a.store.LoadAuthSettings()
	if err != nil {
		return err
	}
	incoming.SMSConfig = mergeSMSConfig(existing.SMSConfig, incoming.SMSConfig, incoming.SMSProvider)
	if incoming.OTPTTLSeconds <= 0 {
		incoming.OTPTTLSeconds = existing.OTPTTLSeconds
	}
	if incoming.OTPLength <= 0 {
		incoming.OTPLength = existing.OTPLength
	}
	if incoming.TwoFactorEnabled && incoming.SMSProvider == "" {
		return fmt.Errorf("sms provider is required when two-factor is enabled")
	}
	if incoming.SMSProvider != "" {
		if err := sms.ValidateProvider(sms.ProviderID(incoming.SMSProvider), incoming.SMSConfig); err != nil {
			return err
		}
	}
	return a.store.SaveAuthSettings(incoming)
}

func mergeSMSConfig(existing, incoming json.RawMessage, provider models.SMSProvider) json.RawMessage {
	if len(incoming) == 0 {
		return existing
	}
	switch provider {
	case models.SMSProviderKavenegar:
		return mergeSecretField(existing, incoming, "api_key")
	case models.SMSProviderMeliPayamak:
		out := mergeSecretField(existing, incoming, "password")
		return out
	default:
		return incoming
	}
}

func redactAuthSettings(s models.AuthSettings) models.AuthSettings {
	if s.SMSProvider != "" {
		s.SMSConfig = sms.RedactConfig(sms.ProviderID(s.SMSProvider), s.SMSConfig)
	}
	return s
}

type LoginResult struct {
	Token        string `json:"token,omitempty"`
	OTPRequired  bool   `json:"otp_required,omitempty"`
	ChallengeID  string `json:"challenge_id,omitempty"`
	ExpiresInSec int    `json:"expires_in_sec,omitempty"`
}

func (a *App) Login(ctx context.Context, phone, password string) (LoginResult, models.User, error) {
	user, err := a.Authenticate(phone, password)
	if err != nil {
		return LoginResult{}, models.User{}, err
	}
	settings, err := a.store.LoadAuthSettings()
	if err != nil {
		return LoginResult{}, models.User{}, err
	}
	if settings.TwoFactorEnabled {
		challengeID, ttl, err := a.StartOTP(ctx, user.ID)
		if err != nil {
			return LoginResult{}, models.User{}, err
		}
		return LoginResult{OTPRequired: true, ChallengeID: challengeID, ExpiresInSec: int(ttl.Seconds())}, redactUser(user), nil
	}
	token, err := a.createSession(user.ID)
	if err != nil {
		return LoginResult{}, models.User{}, err
	}
	return LoginResult{Token: token}, redactUser(user), nil
}

func (a *App) StartOTP(ctx context.Context, userID string) (challengeID string, ttl time.Duration, err error) {
	settings, err := a.store.LoadAuthSettings()
	if err != nil {
		return "", 0, err
	}
	if settings.SMSProvider == "" {
		return "", 0, ErrSMSNotConfigured
	}
	user, err := a.store.GetUser(userID)
	if err != nil {
		return "", 0, err
	}
	_ = a.store.DeleteOTPChallengesForUser(userID)

	length := settings.OTPLength
	if length <= 0 {
		length = 6
	}
	code, err := generateOTPCode(length)
	if err != nil {
		return "", 0, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return "", 0, err
	}
	ttlSec := settings.OTPTTLSeconds
	if ttlSec <= 0 {
		ttlSec = 300
	}
	ttl = time.Duration(ttlSec) * time.Second
	ch := models.OTPChallenge{
		UserID:    userID,
		CodeHash:  string(hash),
		ExpiresAt: time.Now().UTC().Add(ttl),
	}
	if err := a.store.CreateOTPChallenge(ch); err != nil {
		return "", 0, err
	}

	provider, ok := sms.NewRegistry().Provider(sms.ProviderID(settings.SMSProvider))
	if !ok {
		return "", 0, ErrSMSNotConfigured
	}
	body := fmt.Sprintf("DBack login code: %s", code)
	if err := provider.Send(ctx, settings.SMSConfig, user.Phone, body); err != nil {
		return "", 0, fmt.Errorf("send otp: %w", err)
	}
	return ch.ID, ttl, nil
}

func (a *App) VerifyOTP(ctx context.Context, challengeID, code string) (string, models.User, error) {
	ch, err := a.store.GetOTPChallenge(challengeID)
	if err != nil {
		return "", models.User{}, ErrOTPInvalid
	}
	if time.Now().UTC().After(ch.ExpiresAt) {
		_ = a.store.DeleteOTPChallengesForUser(ch.UserID)
		return "", models.User{}, ErrOTPInvalid
	}
	if ch.Attempts >= maxOTPAttempts {
		return "", models.User{}, ErrOTPInvalid
	}
	if err := bcrypt.CompareHashAndPassword([]byte(ch.CodeHash), []byte(code)); err != nil {
		ch.Attempts++
		_ = a.store.UpdateOTPChallenge(ch)
		return "", models.User{}, ErrOTPInvalid
	}
	_ = a.store.DeleteOTPChallengesForUser(ch.UserID)
	user, err := a.store.GetUser(ch.UserID)
	if err != nil {
		return "", models.User{}, err
	}
	if !user.Enabled {
		return "", models.User{}, ErrUserDisabled
	}
	token, err := a.createSession(user.ID)
	if err != nil {
		return "", models.User{}, err
	}
	return token, redactUser(user), nil
}

func (a *App) createSession(userID string) (string, error) {
	token, hash, err := newSessionToken()
	if err != nil {
		return "", err
	}
	expires := time.Now().UTC().Add(sessionTTL)
	if err := a.store.CreateSession(userID, hash, expires); err != nil {
		return "", err
	}
	return token, nil
}

func (a *App) ValidateSession(token string) (string, error) {
	hash := hashSessionToken(token)
	sess, err := a.store.GetSessionByTokenHash(hash)
	if err != nil {
		if errors.Is(err, store.ErrSessionInvalid) {
			return "", ErrSessionInvalid
		}
		return "", err
	}
	user, err := a.store.GetUser(sess.UserID)
	if err != nil {
		return "", ErrSessionInvalid
	}
	if !user.Enabled {
		return "", ErrUserDisabled
	}
	return sess.UserID, nil
}

func (a *App) Logout(token string) error {
	if token == "" {
		return nil
	}
	return a.store.DeleteSession(hashSessionToken(token))
}

func newSessionToken() (plain string, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	plain = base64.RawURLEncoding.EncodeToString(b)
	return plain, hashSessionToken(plain), nil
}

func hashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func generateOTPCode(length int) (string, error) {
	if length <= 0 {
		length = 6
	}
	var out string
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		out += fmt.Sprintf("%d", n.Int64())
	}
	return out, nil
}

func IsOTPInvalid(err error) bool {
	return errors.Is(err, ErrOTPInvalid) || errors.Is(err, store.ErrOTPInvalid)
}

func IsSessionInvalid(err error) bool {
	return errors.Is(err, ErrSessionInvalid) || errors.Is(err, store.ErrSessionInvalid)
}
