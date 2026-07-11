package vaultimport

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"dback/internal/secrets"
	"dback/models"
)

const vaultFileName = "app_data.vault.json"
const currentVersion = 5

// VaultSalt returns the KDF salt from the on-disk vault file header.
func VaultSalt(baseDir string) (string, error) {
	var file models.AppVaultFile
	if err := readJSON(filepath.Join(baseDir, vaultFileName), &file); err != nil {
		return "", err
	}
	return file.Salt, nil
}

func ReadVault(baseDir, passphrase string) (models.AppVaultPayload, error) {
	path := filepath.Join(baseDir, vaultFileName)
	var file models.AppVaultFile
	if err := readJSON(path, &file); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return models.AppVaultPayload{}, os.ErrNotExist
		}
		return models.AppVaultPayload{}, err
	}
	salt, err := base64.StdEncoding.DecodeString(file.Salt)
	if err != nil {
		return models.AppVaultPayload{}, fmt.Errorf("invalid vault salt: %w", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(file.Nonce)
	if err != nil {
		return models.AppVaultPayload{}, fmt.Errorf("invalid vault nonce: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(file.EncryptedPayload)
	if err != nil {
		return models.AppVaultPayload{}, fmt.Errorf("invalid vault payload: %w", err)
	}
	key := secrets.DeriveKey(passphrase, salt)
	payload, err := secrets.DecryptUnmarshalVault(key, nonce, ciphertext)
	if err != nil {
		return models.AppVaultPayload{}, errors.New("wrong master key")
	}
	return payload, nil
}

// HasVault reports whether the encrypted vault file exists.
func HasVault(baseDir string) bool {
	_, err := os.Stat(filepath.Join(baseDir, vaultFileName))
	return err == nil
}

// HasLegacyPlaintext reports whether pre-vault JSON files exist.
func HasLegacyPlaintext(baseDir string) bool {
	paths := []string{
		filepath.Join(baseDir, "profiles.json"),
		filepath.Join(baseDir, "templates.json"),
		filepath.Join(baseDir, "export_history.json"),
		filepath.Join(baseDir, "logs.json"),
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}

// LoadLegacyPayload reads plaintext JSON files into a vault payload.
func LoadLegacyPayload(baseDir string) (models.AppVaultPayload, error) {
	profiles, err := loadLegacyProfiles(baseDir)
	if err != nil {
		return models.AppVaultPayload{}, err
	}
	templates, err := loadLegacyTemplates(baseDir)
	if err != nil {
		return models.AppVaultPayload{}, err
	}
	history, err := loadLegacyHistory(baseDir)
	if err != nil {
		return models.AppVaultPayload{}, err
	}
	logs, err := loadLegacyLogs(baseDir)
	if err != nil {
		return models.AppVaultPayload{}, err
	}
	return models.AppVaultPayload{
		Version:   currentVersion,
		Profiles:  profiles,
		Templates: templates,
		History:   history,
		Logs:      logs,
	}, nil
}

// ArchiveVault renames the vault file after successful SQL import.
func ArchiveVault(baseDir string) error {
	src := filepath.Join(baseDir, vaultFileName)
	if _, err := os.Stat(src); err != nil {
		return nil
	}
	dst := src + ".migrated.bak"
	if err := os.Rename(src, dst); err != nil {
		return err
	}
	log.Printf("vaultimport: archived vault to %q", dst)
	return RemoveLegacyPlaintext(baseDir)
}

// RemoveLegacyPlaintext deletes legacy JSON files.
func RemoveLegacyPlaintext(baseDir string) error {
	names := []string{"profiles.json", "templates.json", "export_history.json", "logs.json"}
	for _, name := range names {
		p := filepath.Join(baseDir, name)
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

func loadLegacyProfiles(baseDir string) ([]models.Profile, error) {
	path := filepath.Join(baseDir, "profiles.json")
	var bundle models.ProfileBundle
	if err := readJSON(path, &bundle); err == nil && bundle.Profiles != nil {
		return bundle.Profiles, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		var legacy models.AppConfig
		if legacyErr := readJSON(path, &legacy); legacyErr == nil {
			return legacy.Profiles, nil
		}
		if err != nil {
			return nil, err
		}
	}
	return []models.Profile{}, nil
}

func loadLegacyTemplates(baseDir string) ([]models.SQLTemplate, error) {
	path := filepath.Join(baseDir, "templates.json")
	var bundle models.TemplateBundle
	err := readJSON(path, &bundle)
	if err == nil && bundle.Templates != nil {
		return bundle.Templates, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return nil, err
}

func loadLegacyHistory(baseDir string) ([]models.ExportRecord, error) {
	path := filepath.Join(baseDir, "export_history.json")
	var history models.BackupHistory
	if err := readJSON(path, &history); err == nil && history.Records != nil {
		return history.Records, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		var legacy []models.ExportRecord
		if legacyErr := readJSON(path, &legacy); legacyErr == nil {
			return legacy, nil
		}
		if err != nil {
			return nil, err
		}
	}
	return []models.ExportRecord{}, nil
}

func loadLegacyLogs(baseDir string) ([]models.LogEntry, error) {
	path := filepath.Join(baseDir, "logs.json")
	var logs models.ActivityLog
	if err := readJSON(path, &logs); err == nil && logs.Entries != nil {
		return logs.Entries, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		var legacy []models.LogEntry
		if legacyErr := readJSON(path, &legacy); legacyErr == nil {
			return legacy, nil
		}
		if err != nil {
			return nil, err
		}
	}
	return []models.LogEntry{}, nil
}

func readJSON(path string, value any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(value)
}
