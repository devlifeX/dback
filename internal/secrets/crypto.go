package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"dback/models"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    = 3
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeyLen  = 32
)

type secretPayload struct {
	SSHPassword    string `json:"ssh_password,omitempty"`
	JumpPassword   string `json:"jump_password,omitempty"`
	DBPassword     string `json:"db_password,omitempty"`
	AuthKeyPEM     string `json:"auth_key_pem,omitempty"`
	JumpAuthKeyPEM string `json:"jump_auth_key_pem,omitempty"`
}

type appSecretPayload struct {
	Profiles []secretPayload `json:"profiles"`
}

// EncryptBundle encrypts profile secrets into an encrypted ProfileBundle.
func EncryptBundle(profiles []models.Profile, passphrase string) (models.ProfileBundle, error) {
	if passphrase == "" {
		return models.ProfileBundle{}, errors.New("passphrase required for encrypted export")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return models.ProfileBundle{}, err
	}
	key := deriveKey(passphrase, salt)

	payloads := make([]secretPayload, len(profiles))
	stripped := make([]models.Profile, len(profiles))
	for i, p := range profiles {
		payloads[i] = secretPayload{
			SSHPassword:    p.SSHPassword,
			JumpPassword:   p.JumpPassword,
			DBPassword:     p.DBPassword,
			AuthKeyPEM:     p.AuthKeyPEM,
			JumpAuthKeyPEM: p.JumpAuthKeyPEM,
		}
		stripped[i] = stripProfileSecrets(p)
	}

	inner, err := json.Marshal(struct {
		Profiles []models.Profile `json:"profiles"`
		Secrets  []secretPayload  `json:"secrets"`
	}{Profiles: stripped, Secrets: payloads})
	if err != nil {
		return models.ProfileBundle{}, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return models.ProfileBundle{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return models.ProfileBundle{}, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return models.ProfileBundle{}, err
	}
	ciphertext := gcm.Seal(nil, nonce, inner, nil)

	return models.ProfileBundle{
		Version:          3,
		Encrypted:        true,
		Salt:             base64.StdEncoding.EncodeToString(salt),
		Nonce:            base64.StdEncoding.EncodeToString(nonce),
		EncryptedPayload: base64.StdEncoding.EncodeToString(ciphertext),
	}, nil
}

// DecryptBundle decrypts an encrypted ProfileBundle and restores secrets.
func DecryptBundle(bundle models.ProfileBundle, passphrase string) ([]models.Profile, error) {
	if !bundle.Encrypted {
		return bundle.Profiles, nil
	}
	if passphrase == "" {
		return nil, errors.New("passphrase required to decrypt profile bundle")
	}
	salt, err := base64.StdEncoding.DecodeString(bundle.Salt)
	if err != nil {
		return nil, fmt.Errorf("invalid salt: %w", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(bundle.Nonce)
	if err != nil {
		return nil, fmt.Errorf("invalid nonce: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(bundle.EncryptedPayload)
	if err != nil {
		return nil, fmt.Errorf("invalid ciphertext: %w", err)
	}

	key := deriveKey(passphrase, salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decryption failed: wrong passphrase or corrupted bundle")
	}

	var payload struct {
		Profiles []models.Profile `json:"profiles"`
		Secrets  []secretPayload  `json:"secrets"`
	}
	if err := json.Unmarshal(plain, &payload); err != nil {
		return nil, err
	}
	for i := range payload.Profiles {
		if i < len(payload.Secrets) {
			s := payload.Secrets[i]
			payload.Profiles[i].SSHPassword = s.SSHPassword
			payload.Profiles[i].JumpPassword = s.JumpPassword
			payload.Profiles[i].DBPassword = s.DBPassword
			payload.Profiles[i].AuthKeyPEM = s.AuthKeyPEM
			payload.Profiles[i].JumpAuthKeyPEM = s.JumpAuthKeyPEM
		}
	}
	return payload.Profiles, nil
}

func deriveKey(passphrase string, salt []byte) []byte {
	return argon2.IDKey([]byte(passphrase), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
}

func stripProfileSecrets(p models.Profile) models.Profile {
	p.SSHPassword = ""
	p.JumpPassword = ""
	p.DBPassword = ""
	p.AuthKeyPEM = ""
	p.JumpAuthKeyPEM = ""
	p.ExportSettings = nil
	p.ImportSettings = nil
	return p
}

type appPlainPayload struct {
	Profiles  []models.Profile      `json:"profiles"`
	Secrets   appSecretPayload      `json:"secrets"`
	Templates []models.SQLTemplate  `json:"templates"`
	History   []models.ExportRecord `json:"history"`
	Logs      []models.LogEntry     `json:"logs"`
}

// EncryptAppBundle encrypts profile secrets and app metadata into an encrypted AppBundle.
func EncryptAppBundle(profiles []models.Profile, templates []models.SQLTemplate, history []models.ExportRecord, logs []models.LogEntry, passphrase string) (models.AppBundle, error) {
	if passphrase == "" {
		return models.AppBundle{}, errors.New("passphrase required for encrypted export")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return models.AppBundle{}, err
	}
	key := deriveKey(passphrase, salt)

	secretsList := make([]secretPayload, len(profiles))
	stripped := make([]models.Profile, len(profiles))
	for i, p := range profiles {
		secretsList[i] = secretPayload{
			SSHPassword:    p.SSHPassword,
			JumpPassword:   p.JumpPassword,
			DBPassword:     p.DBPassword,
			AuthKeyPEM:     p.AuthKeyPEM,
			JumpAuthKeyPEM: p.JumpAuthKeyPEM,
		}
		stripped[i] = stripProfileSecrets(p)
	}

	inner, err := json.Marshal(appPlainPayload{
		Profiles:  stripped,
		Secrets:   appSecretPayload{Profiles: secretsList},
		Templates: templates,
		History:   history,
		Logs:      logs,
	})
	if err != nil {
		return models.AppBundle{}, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return models.AppBundle{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return models.AppBundle{}, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return models.AppBundle{}, err
	}
	ciphertext := gcm.Seal(nil, nonce, inner, nil)

	return models.AppBundle{
		Version:          3,
		ExportedAt:       time.Now(),
		Encrypted:        true,
		Salt:             base64.StdEncoding.EncodeToString(salt),
		Nonce:            base64.StdEncoding.EncodeToString(nonce),
		EncryptedPayload: base64.StdEncoding.EncodeToString(ciphertext),
	}, nil
}

// DecryptAppBundle decrypts an encrypted AppBundle and restores profile secrets.
func DecryptAppBundle(bundle models.AppBundle, passphrase string) (models.AppBundle, error) {
	if !bundle.Encrypted {
		return bundle, nil
	}
	if passphrase == "" {
		return models.AppBundle{}, errors.New("passphrase required to decrypt app bundle")
	}
	salt, err := base64.StdEncoding.DecodeString(bundle.Salt)
	if err != nil {
		return models.AppBundle{}, fmt.Errorf("invalid salt: %w", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(bundle.Nonce)
	if err != nil {
		return models.AppBundle{}, fmt.Errorf("invalid nonce: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(bundle.EncryptedPayload)
	if err != nil {
		return models.AppBundle{}, fmt.Errorf("invalid ciphertext: %w", err)
	}

	key := deriveKey(passphrase, salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return models.AppBundle{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return models.AppBundle{}, err
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return models.AppBundle{}, errors.New("decryption failed: wrong passphrase or corrupted bundle")
	}

	var payload appPlainPayload
	if err := json.Unmarshal(plain, &payload); err != nil {
		return models.AppBundle{}, err
	}
	for i := range payload.Profiles {
		if i < len(payload.Secrets.Profiles) {
			s := payload.Secrets.Profiles[i]
			payload.Profiles[i].SSHPassword = s.SSHPassword
			payload.Profiles[i].JumpPassword = s.JumpPassword
			payload.Profiles[i].DBPassword = s.DBPassword
			payload.Profiles[i].AuthKeyPEM = s.AuthKeyPEM
			payload.Profiles[i].JumpAuthKeyPEM = s.JumpAuthKeyPEM
		}
	}
	return models.AppBundle{
		Version:    bundle.Version,
		ExportedAt: bundle.ExportedAt,
		Profiles:   payload.Profiles,
		Templates:  payload.Templates,
		History:    payload.History,
		Logs:       payload.Logs,
	}, nil
}
