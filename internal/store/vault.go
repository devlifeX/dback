package store

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"dback/internal/secrets"
	"dback/models"
)

const vaultFileName = "app_data.vault.json"

var (
	ErrVaultLocked                = errors.New("vault is locked")
	ErrVaultExists                = errors.New("vault already exists")
	ErrVaultNotFound              = errors.New("vault not found")
	ErrWrongMasterKey             = errors.New("wrong master key")
	ErrMasterKeyRequired          = errors.New("master key is required")
	ErrIncludeSecretsNoPassphrase = errors.New("passphrase required when including secrets")
	ErrLegacyPlaintextWithVault   = errors.New("legacy plaintext files found alongside encrypted vault")
	ErrSyncNotConfigured          = errors.New("sync settings are not configured")
)

const minMasterKeyLen = 4

func (s *Store) VaultPath() string {
	return filepath.Join(s.baseDir, vaultFileName)
}

func (s *Store) HasVault() bool {
	_, err := os.Stat(s.VaultPath())
	return err == nil
}

func (s *Store) HasLegacyPlaintext() bool {
	paths := []string{s.ProfilesPath(), s.TemplatesPath(), s.HistoryPath(), s.LogsPath()}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}

func (s *Store) IsUnlocked() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.unlocked
}

func (s *Store) Revision() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.revision
}

func (s *Store) bumpRevisionLocked() {
	s.revision++
}

func (s *Store) requireUnlocked() error {
	if !s.unlocked {
		return ErrVaultLocked
	}
	return nil
}

// CreateVault initializes a new encrypted vault with seed data.
func (s *Store) CreateVault(passphrase string) error {
	if err := validateMasterKey(passphrase); err != nil {
		log.Printf("store.CreateVault: invalid master key: %v", err)
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.unlocked {
		return nil
	}
	if _, err := os.Stat(s.VaultPath()); err == nil {
		log.Printf("store.CreateVault: vault already exists at %q", s.VaultPath())
		return ErrVaultExists
	}

	log.Printf("store.CreateVault: creating new vault at %q", s.VaultPath())
	payload := models.AppVaultPayload{
		Version:   CurrentVersion,
		Profiles:  []models.Profile{},
		Templates: seedTemplates(),
		History:   []models.ExportRecord{},
		Logs:      []models.LogEntry{},
	}
	if err := s.writeVaultLocked(passphrase, payload); err != nil {
		log.Printf("store.CreateVault: writeVaultLocked failed: %v", err)
		return err
	}
	s.applyPayloadLocked(payload)
	s.setMasterKeyLocked(passphrase)
	s.unlocked = true
	s.bumpRevisionLocked()
	log.Printf("store.CreateVault: vault created successfully")
	return nil
}

// Unlock opens an existing vault or migrates legacy plaintext files into a new vault.
func (s *Store) Unlock(passphrase string) error {
	if passphrase == "" {
		return ErrMasterKeyRequired
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.unlocked {
		return nil
	}

	if _, err := os.Stat(s.VaultPath()); err == nil {
		log.Printf("store.Unlock: vault found at %q", s.VaultPath())
		if s.hasLegacyPlaintextLocked() {
			log.Printf("store.Unlock: legacy plaintext files present alongside vault, removing them")
			if err := s.removeLegacyPlaintextLocked(); err != nil {
				return fmt.Errorf("%w: %v", ErrLegacyPlaintextWithVault, err)
			}
		}
		payload, key, err := s.readVaultFileLocked(passphrase)
		if err != nil {
			log.Printf("store.Unlock: readVaultFileLocked failed: %v", err)
			return err
		}
		s.dataKey = key
		s.applyPayloadLocked(payload)
		s.setMasterKeyLocked(passphrase)
		s.unlocked = true
		s.bumpRevisionLocked()
		_ = s.removeLegacyPlaintextLocked()
		log.Printf("store.Unlock: vault unlocked (profiles=%d templates=%d)", len(payload.Profiles), len(payload.Templates))
		return nil
	}

	if s.hasLegacyPlaintextLocked() {
		log.Printf("store.Unlock: no vault found, migrating from legacy plaintext at %q", s.baseDir)
		payload, err := s.loadLegacyPayloadLocked()
		if err != nil {
			log.Printf("store.Unlock: loadLegacyPayloadLocked failed: %v", err)
			return err
		}
		if err := s.writeVaultLocked(passphrase, payload); err != nil {
			log.Printf("store.Unlock: writeVaultLocked (migrate) failed: %v", err)
			return err
		}
		if err := s.removeLegacyPlaintextLocked(); err != nil {
			log.Printf("store.Unlock: removeLegacyPlaintextLocked failed: %v", err)
			return err
		}
		s.applyPayloadLocked(payload)
		s.setMasterKeyLocked(passphrase)
		s.unlocked = true
		s.bumpRevisionLocked()
		log.Printf("store.Unlock: legacy migration complete (profiles=%d templates=%d)", len(payload.Profiles), len(payload.Templates))
		return nil
	}

	log.Printf("store.Unlock: no vault and no legacy plaintext found at %q", s.baseDir)
	return ErrVaultNotFound
}

func (s *Store) applyPayloadLocked(payload models.AppVaultPayload) {
	s.profiles = flattenProfiles(payload.Profiles)
	s.templates = append([]models.SQLTemplate(nil), payload.Templates...)
	if len(s.templates) == 0 {
		s.templates = seedTemplates()
	}
	s.history = append([]models.ExportRecord(nil), payload.History...)
	s.logs = append([]models.LogEntry(nil), payload.Logs...)
	s.sync = payload.Sync.Clone()
	s.syncActivity = payload.SyncActivity
	if len(payload.ImportDestByProfile) > 0 {
		s.importDestByProfile = cloneStringMap(payload.ImportDestByProfile)
	} else {
		s.importDestByProfile = map[string]string{}
	}
}

func (s *Store) persistVaultLocked() error {
	if !s.unlocked || len(s.dataKey) == 0 {
		return ErrVaultLocked
	}
	payload := s.currentPayloadLocked()
	nonce, ciphertext, err := secrets.MarshalEncryptVault(s.dataKey, payload)
	if err != nil {
		return err
	}
	file := models.AppVaultFile{
		Version:          CurrentVersion,
		Salt:             s.vaultSalt,
		Nonce:            base64.StdEncoding.EncodeToString(nonce),
		UpdatedAt:        time.Now(),
		EncryptedPayload: base64.StdEncoding.EncodeToString(ciphertext),
	}
	return writeJSON(s.VaultPath(), file)
}

func (s *Store) currentPayloadLocked() models.AppVaultPayload {
	profiles := append([]models.Profile(nil), s.profiles...)
	for i := range profiles {
		profiles[i].ExportSettings = nil
		profiles[i].ImportSettings = nil
	}
	return models.AppVaultPayload{
		Version:             CurrentVersion,
		Profiles:            profiles,
		Templates:           append([]models.SQLTemplate(nil), s.templates...),
		History:             append([]models.ExportRecord(nil), s.history...),
		Logs:                append([]models.LogEntry(nil), s.logs...),
		Sync:                s.sync.Clone(),
		SyncActivity:        s.syncActivity,
		ImportDestByProfile: cloneStringMap(s.importDestByProfile),
	}
}

func (s *Store) writeVaultLocked(passphrase string, payload models.AppVaultPayload) error {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		log.Printf("store.writeVaultLocked: rand.Read failed: %v", err)
		return err
	}
	key := secrets.DeriveKey(passphrase, salt)
	nonce, ciphertext, err := secrets.MarshalEncryptVault(key, payload)
	if err != nil {
		log.Printf("store.writeVaultLocked: encrypt failed: %v", err)
		return err
	}
	file := models.AppVaultFile{
		Version:          CurrentVersion,
		Salt:             base64.StdEncoding.EncodeToString(salt),
		Nonce:            base64.StdEncoding.EncodeToString(nonce),
		UpdatedAt:        time.Now(),
		EncryptedPayload: base64.StdEncoding.EncodeToString(ciphertext),
	}
	if err := writeJSON(s.VaultPath(), file); err != nil {
		log.Printf("store.writeVaultLocked: writeJSON to %q failed: %v", s.VaultPath(), err)
		return err
	}
	s.dataKey = key
	s.vaultSalt = file.Salt
	return nil
}

func (s *Store) readVaultFileLocked(passphrase string) (models.AppVaultPayload, []byte, error) {
	var file models.AppVaultFile
	if err := readJSON(s.VaultPath(), &file); err != nil {
		log.Printf("store.readVaultFileLocked: readJSON from %q failed: %v", s.VaultPath(), err)
		return models.AppVaultPayload{}, nil, err
	}
	salt, err := base64.StdEncoding.DecodeString(file.Salt)
	if err != nil {
		return models.AppVaultPayload{}, nil, fmt.Errorf("invalid vault salt: %w", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(file.Nonce)
	if err != nil {
		return models.AppVaultPayload{}, nil, fmt.Errorf("invalid vault nonce: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(file.EncryptedPayload)
	if err != nil {
		return models.AppVaultPayload{}, nil, fmt.Errorf("invalid vault payload: %w", err)
	}
	key := secrets.DeriveKey(passphrase, salt)
	payload, err := secrets.DecryptUnmarshalVault(key, nonce, ciphertext)
	if err != nil {
		log.Printf("store.readVaultFileLocked: decrypt failed (wrong key or corrupt vault): %v", err)
		return models.AppVaultPayload{}, nil, ErrWrongMasterKey
	}
	s.vaultSalt = file.Salt
	return payload, key, nil
}

func (s *Store) hasLegacyPlaintextLocked() bool {
	paths := []string{s.ProfilesPath(), s.TemplatesPath(), s.HistoryPath(), s.LogsPath()}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}

func (s *Store) loadLegacyPayloadLocked() (models.AppVaultPayload, error) {
	profiles, err := s.loadLegacyProfiles()
	if err != nil {
		return models.AppVaultPayload{}, err
	}
	templates, err := s.loadLegacyTemplates()
	if err != nil {
		return models.AppVaultPayload{}, err
	}
	history, err := s.loadLegacyHistory()
	if err != nil {
		return models.AppVaultPayload{}, err
	}
	logs, err := s.loadLegacyLogs()
	if err != nil {
		return models.AppVaultPayload{}, err
	}
	return models.AppVaultPayload{
		Version:   CurrentVersion,
		Profiles:  profiles,
		Templates: templates,
		History:   history,
		Logs:      logs,
	}, nil
}

func (s *Store) loadLegacyProfiles() ([]models.Profile, error) {
	var bundle models.ProfileBundle
	if err := readJSON(s.ProfilesPath(), &bundle); err == nil && bundle.Profiles != nil {
		return flattenProfiles(bundle.Profiles), nil
	} else if !errors.Is(err, os.ErrNotExist) {
		var legacy models.AppConfig
		if legacyErr := readJSON(s.ProfilesPath(), &legacy); legacyErr == nil {
			return flattenProfiles(legacy.Profiles), nil
		}
		if err != nil {
			return nil, err
		}
	}
	return []models.Profile{}, nil
}

func (s *Store) loadLegacyTemplates() ([]models.SQLTemplate, error) {
	var bundle models.TemplateBundle
	err := readJSON(s.TemplatesPath(), &bundle)
	if err == nil && bundle.Templates != nil {
		return bundle.Templates, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return seedTemplates(), nil
	}
	return nil, err
}

func (s *Store) loadLegacyHistory() ([]models.ExportRecord, error) {
	var history models.BackupHistory
	if err := readJSON(s.HistoryPath(), &history); err == nil && history.Records != nil {
		return history.Records, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		var legacy []models.ExportRecord
		if legacyErr := readJSON(s.HistoryPath(), &legacy); legacyErr == nil {
			return legacy, nil
		}
		if err != nil {
			return nil, err
		}
	}
	return []models.ExportRecord{}, nil
}

func (s *Store) loadLegacyLogs() ([]models.LogEntry, error) {
	var logs models.ActivityLog
	if err := readJSON(s.LogsPath(), &logs); err == nil && logs.Entries != nil {
		return logs.Entries, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		var legacy []models.LogEntry
		if legacyErr := readJSON(s.LogsPath(), &legacy); legacyErr == nil {
			return legacy, nil
		}
		if err != nil {
			return nil, err
		}
	}
	return []models.LogEntry{}, nil
}

func (s *Store) removeLegacyPlaintextLocked() error {
	paths := []string{s.ProfilesPath(), s.TemplatesPath(), s.HistoryPath(), s.LogsPath()}
	for _, p := range paths {
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		legacy := p + ".legacy"
		if err := os.Remove(legacy); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func (s *Store) Lock() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.dataKey {
		s.dataKey[i] = 0
	}
	s.dataKey = nil
	s.clearMasterKeyLocked()
	s.vaultSalt = ""
	s.unlocked = false
	s.profiles = nil
	s.templates = nil
	s.history = nil
	s.logs = nil
	s.sync = nil
	s.syncActivity = models.SyncActivity{}
}

func (s *Store) setMasterKeyLocked(passphrase string) {
	s.clearMasterKeyLocked()
	s.masterKey = []byte(passphrase)
}

func (s *Store) clearMasterKeyLocked() {
	for i := range s.masterKey {
		s.masterKey[i] = 0
	}
	s.masterKey = nil
}

func (s *Store) masterPassphraseLocked() (string, error) {
	if !s.unlocked || len(s.masterKey) == 0 {
		return "", ErrVaultLocked
	}
	return string(s.masterKey), nil
}

func validateMasterKey(passphrase string) error {
	if passphrase == "" {
		return ErrMasterKeyRequired
	}
	if len(passphrase) < minMasterKeyLen {
		return fmt.Errorf("master key must be at least %d characters", minMasterKeyLen)
	}
	return nil
}
