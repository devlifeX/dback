package sqlstore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"dback/internal/secrets"
	"dback/internal/storemodel"
	"dback/models"

	"github.com/google/uuid"
)

func (s *Store) ListUsers() ([]models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return nil, err
	}
	return append([]models.User{}, s.users...), nil
}

func (s *Store) GetUser(id string) (models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return models.User{}, err
	}
	for _, u := range s.users {
		if u.ID == id {
			return u, nil
		}
	}
	return models.User{}, storemodel.ErrUserNotFound
}

func (s *Store) GetUserByPhone(phone string) (models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return models.User{}, err
	}
	norm := normalizePhone(phone)
	for _, u := range s.users {
		if normalizePhone(u.Phone) == norm {
			return u, nil
		}
	}
	return models.User{}, storemodel.ErrUserNotFound
}

func (s *Store) SaveUser(user models.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	user.Phone = normalizePhone(user.Phone)
	if user.Phone == "" {
		return fmt.Errorf("phone is required")
	}
	if user.ID == "" {
		user.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	user.UpdatedAt = now
	for _, existing := range s.users {
		if existing.ID != user.ID && normalizePhone(existing.Phone) == user.Phone {
			return storemodel.ErrUserExists
		}
	}
	for i, existing := range s.users {
		if existing.ID == user.ID {
			if user.PasswordHash == "" {
				user.PasswordHash = existing.PasswordHash
			}
			if user.CreatedAt.IsZero() {
				user.CreatedAt = existing.CreatedAt
			}
			s.users[i] = user
			s.bumpRevisionLocked()
			return s.persistAllLocked()
		}
	}
	s.users = append(s.users, user)
	s.bumpRevisionLocked()
	return s.persistAllLocked()
}

func (s *Store) DeleteUser(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	for i, u := range s.users {
		if u.ID == id {
			s.users = append(s.users[:i], s.users[i+1:]...)
			s.bumpRevisionLocked()
			if err := s.persistAllLocked(); err != nil {
				return err
			}
			_ = s.deleteSessionsForUserLocked(id)
			_ = s.deleteOTPForUserLocked(id)
			return nil
		}
	}
	return storemodel.ErrUserNotFound
}

func (s *Store) LoadAuthSettings() (models.AuthSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return models.AuthSettings{}, err
	}
	return s.authSettings, nil
}

func (s *Store) SaveAuthSettings(settings models.AuthSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	s.authSettings = settings
	s.bumpRevisionLocked()
	return s.persistAllLocked()
}

func (s *Store) CreateSession(userID, tokenHash string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	if err := s.ensureDB(); err != nil {
		return err
	}
	_, err := s.db.Exec(`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES (?, ?, ?)`,
		tokenHash, userID, expiresAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) GetSessionByTokenHash(tokenHash string) (models.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return models.Session{}, err
	}
	if err := s.ensureDB(); err != nil {
		return models.Session{}, err
	}
	var userID, expiresRaw string
	err := s.db.QueryRow(`SELECT user_id, expires_at FROM sessions WHERE token_hash = ?`, tokenHash).Scan(&userID, &expiresRaw)
	if err == sql.ErrNoRows {
		return models.Session{}, storemodel.ErrSessionInvalid
	}
	if err != nil {
		return models.Session{}, err
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, expiresRaw)
	if err != nil {
		return models.Session{}, storemodel.ErrSessionInvalid
	}
	if time.Now().UTC().After(expiresAt) {
		_, _ = s.db.Exec(`DELETE FROM sessions WHERE token_hash = ?`, tokenHash)
		return models.Session{}, storemodel.ErrSessionInvalid
	}
	return models.Session{TokenHash: tokenHash, UserID: userID, ExpiresAt: expiresAt}, nil
}

func (s *Store) DeleteSession(tokenHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	if err := s.ensureDB(); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM sessions WHERE token_hash = ?`, tokenHash)
	return err
}

func (s *Store) DeleteSessionsForUser(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deleteSessionsForUserLocked(userID)
}

func (s *Store) deleteSessionsForUserLocked(userID string) error {
	if err := s.ensureDB(); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)
	return err
}

func (s *Store) CreateOTPChallenge(ch models.OTPChallenge) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	if err := s.ensureDB(); err != nil {
		return err
	}
	if ch.ID == "" {
		ch.ID = uuid.NewString()
	}
	_, err := s.db.Exec(`INSERT INTO otp_challenges (id, user_id, code_hash, expires_at, attempts) VALUES (?, ?, ?, ?, ?)`,
		ch.ID, ch.UserID, ch.CodeHash, ch.ExpiresAt.UTC().Format(time.RFC3339Nano), ch.Attempts)
	return err
}

func (s *Store) GetOTPChallenge(id string) (models.OTPChallenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return models.OTPChallenge{}, err
	}
	if err := s.ensureDB(); err != nil {
		return models.OTPChallenge{}, err
	}
	var userID, codeHash, expiresRaw string
	var attempts int
	err := s.db.QueryRow(`SELECT user_id, code_hash, expires_at, attempts FROM otp_challenges WHERE id = ?`, id).
		Scan(&userID, &codeHash, &expiresRaw, &attempts)
	if err == sql.ErrNoRows {
		return models.OTPChallenge{}, storemodel.ErrOTPInvalid
	}
	if err != nil {
		return models.OTPChallenge{}, err
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, expiresRaw)
	if err != nil {
		return models.OTPChallenge{}, storemodel.ErrOTPInvalid
	}
	return models.OTPChallenge{ID: id, UserID: userID, CodeHash: codeHash, ExpiresAt: expiresAt, Attempts: attempts}, nil
}

func (s *Store) UpdateOTPChallenge(ch models.OTPChallenge) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	if err := s.ensureDB(); err != nil {
		return err
	}
	_, err := s.db.Exec(`UPDATE otp_challenges SET code_hash=?, expires_at=?, attempts=? WHERE id=?`,
		ch.CodeHash, ch.ExpiresAt.UTC().Format(time.RFC3339Nano), ch.Attempts, ch.ID)
	return err
}

func (s *Store) DeleteOTPChallengesForUser(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deleteOTPForUserLocked(userID)
}

func (s *Store) deleteOTPForUserLocked(userID string) error {
	if err := s.ensureDB(); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM otp_challenges WHERE user_id = ?`, userID)
	return err
}

func normalizePhone(phone string) string {
	var b []byte
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			b = append(b, byte(r))
		}
	}
	return string(b)
}

func (s *Store) loadUsersLocked(enc *secrets.FieldEncryptor) ([]models.User, error) {
	rows, err := s.db.Query(`SELECT id, phone, name, password_hash, enabled, created_at, updated_at FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.User
	for rows.Next() {
		var u models.User
		var enabled int
		var createdRaw, updatedRaw string
		if err := rows.Scan(&u.ID, &u.Phone, &u.Name, &u.PasswordHash, &enabled, &createdRaw, &updatedRaw); err != nil {
			return nil, err
		}
		u.Enabled = enabled != 0
		hash, err := enc.Decrypt(u.PasswordHash)
		if err != nil {
			return nil, err
		}
		u.PasswordHash = hash
		u.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdRaw)
		u.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedRaw)
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) replaceUsersTx(tx *sql.Tx, enc *secrets.FieldEncryptor) error {
	if _, err := tx.Exec(`DELETE FROM users`); err != nil {
		return err
	}
	for _, u := range s.users {
		hash, err := enc.Encrypt(u.PasswordHash)
		if err != nil {
			return err
		}
		created := u.CreatedAt.UTC().Format(time.RFC3339Nano)
		updated := u.UpdatedAt.UTC().Format(time.RFC3339Nano)
		if _, err := tx.Exec(`INSERT INTO users (id, phone, name, password_hash, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			u.ID, u.Phone, u.Name, hash, boolToInt(u.Enabled), created, updated); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) loadAuthSettingsLocked(enc *secrets.FieldEncryptor) (models.AuthSettings, error) {
	var raw string
	err := s.db.QueryRow(`SELECT data_json FROM auth_settings WHERE id = 1`).Scan(&raw)
	if err == sql.ErrNoRows || raw == "" || raw == "{}" {
		return models.DefaultAuthSettings(), nil
	}
	if err != nil {
		return models.AuthSettings{}, err
	}
	var settings models.AuthSettings
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return models.AuthSettings{}, err
	}
	cfg, err := decryptSMSConfig(settings.SMSConfig, enc)
	if err != nil {
		return models.AuthSettings{}, err
	}
	settings.SMSConfig = cfg
	if settings.OTPTTLSeconds <= 0 {
		settings.OTPTTLSeconds = 300
	}
	if settings.OTPLength <= 0 {
		settings.OTPLength = 6
	}
	return settings, nil
}

func (s *Store) replaceAuthSettingsTx(tx *sql.Tx, enc *secrets.FieldEncryptor) error {
	if _, err := tx.Exec(`DELETE FROM auth_settings`); err != nil {
		return err
	}
	cp := s.authSettings
	cfg, err := encryptSMSConfig(cp.SMSConfig, enc)
	if err != nil {
		return err
	}
	cp.SMSConfig = cfg
	raw, err := json.Marshal(cp)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO auth_settings (id, data_json) VALUES (1, ?)`, string(raw))
	return err
}

func encryptSMSConfig(raw json.RawMessage, enc *secrets.FieldEncryptor) (json.RawMessage, error) {
	return encryptNotifyConfig(raw, enc)
}

func decryptSMSConfig(raw json.RawMessage, enc *secrets.FieldEncryptor) (json.RawMessage, error) {
	return decryptNotifyConfig(raw, enc)
}
