package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"dback/internal/secrets"
	"dback/models"
)

const CurrentVersion = 3

type Store struct {
	baseDir string
	mu      sync.Mutex
}

func New(baseDir string) *Store {
	if baseDir == "" {
		baseDir = "."
	}
	return &Store{baseDir: baseDir}
}

func (s *Store) ProfilesPath() string  { return filepath.Join(s.baseDir, "profiles.json") }
func (s *Store) HistoryPath() string   { return filepath.Join(s.baseDir, "export_history.json") }
func (s *Store) LogsPath() string      { return filepath.Join(s.baseDir, "logs.json") }
func (s *Store) TemplatesPath() string { return filepath.Join(s.baseDir, "templates.json") }

func (s *Store) LoadProfiles() ([]models.Profile, error) {
	var bundle models.ProfileBundle
	if err := readJSON(s.ProfilesPath(), &bundle); err == nil && bundle.Profiles != nil {
		return flattenProfiles(bundle.Profiles), nil
	} else if !errors.Is(err, os.ErrNotExist) {
		var legacy models.AppConfig
		if legacyErr := readJSON(s.ProfilesPath(), &legacy); legacyErr == nil {
			return flattenProfiles(legacy.Profiles), nil
		}
		return nil, err
	}
	return []models.Profile{}, nil
}

func (s *Store) SaveProfiles(profiles []models.Profile) error {
	normalized := flattenProfiles(profiles)
	for i := range normalized {
		normalized[i].ExportSettings = nil
		normalized[i].ImportSettings = nil
	}
	return s.writeJSON(s.ProfilesPath(), models.ProfileBundle{
		Version:  CurrentVersion,
		Profiles: normalized,
	})
}

func (s *Store) LoadTemplates() ([]models.SQLTemplate, error) {
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

func (s *Store) SaveTemplates(templates []models.SQLTemplate) error {
	return s.writeJSON(s.TemplatesPath(), models.TemplateBundle{
		Version:   CurrentVersion,
		Templates: templates,
	})
}

func seedTemplates() []models.SQLTemplate {
	now := time.Now()
	return []models.SQLTemplate{
		{
			ID:          "seed-recreate-db",
			Name:        "Recreate database",
			Description: "Drop and recreate target database",
			Body:        "DROP DATABASE IF EXISTS {databasename};\nCREATE DATABASE {databasename};",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "seed-create-admin",
			Name:        "Create admin user",
			Description: "WordPress admin user devlife",
			Body:        sqlTemplateCreateAdminUser,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}
}

const sqlTemplateCreateAdminUser = `INSERT INTO wp_users
(user_login, user_pass, user_nicename, user_email, user_registered, user_status, display_name)
VALUES ('devlife', MD5('devlife'), 'devlife', 'devlife@example.com', NOW(), 0, 'devlife');

DELETE FROM wp_usermeta WHERE user_id IN (SELECT ID FROM (SELECT ID FROM wp_users WHERE user_login = 'devlife') t);

INSERT INTO wp_usermeta (user_id, meta_key, meta_value)
SELECT ID, 'wp_capabilities', 'a:1:{s:13:"administrator";b:1;}' FROM wp_users WHERE user_login = 'devlife';

INSERT INTO wp_usermeta (user_id, meta_key, meta_value)
SELECT ID, 'wp_user_level', '10' FROM wp_users WHERE user_login = 'devlife';`

func (s *Store) LoadHistory() ([]models.ExportRecord, error) {
	var history models.BackupHistory
	if err := readJSON(s.HistoryPath(), &history); err == nil && history.Records != nil {
		return history.Records, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		var legacy []models.ExportRecord
		if legacyErr := readJSON(s.HistoryPath(), &legacy); legacyErr == nil {
			return legacy, nil
		}
		return nil, err
	}
	return []models.ExportRecord{}, nil
}

func (s *Store) SaveHistory(records []models.ExportRecord) error {
	return s.writeJSON(s.HistoryPath(), models.BackupHistory{
		Version: CurrentVersion,
		Records: records,
	})
}

func (s *Store) LoadLogs() ([]models.LogEntry, error) {
	var logs models.ActivityLog
	if err := readJSON(s.LogsPath(), &logs); err == nil && logs.Entries != nil {
		return logs.Entries, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		var legacy []models.LogEntry
		if legacyErr := readJSON(s.LogsPath(), &legacy); legacyErr == nil {
			return legacy, nil
		}
		return nil, err
	}
	return []models.LogEntry{}, nil
}

func (s *Store) SaveLogs(entries []models.LogEntry) error {
	return s.writeJSON(s.LogsPath(), models.ActivityLog{
		Version: CurrentVersion,
		Entries: entries,
	})
}

// ImportProfilesBundle loads profiles from a bundle file (plain or encrypted).
func (s *Store) ImportProfilesBundle(path string, includeSecrets bool, passphrase string) ([]models.Profile, error) {
	var bundle models.ProfileBundle
	if err := readJSON(path, &bundle); err != nil {
		return nil, err
	}
	var profiles []models.Profile
	var err error
	if bundle.Encrypted {
		if !includeSecrets {
			return nil, errors.New("encrypted bundle requires include secrets and passphrase")
		}
		profiles, err = secrets.DecryptBundle(bundle, passphrase)
		if err != nil {
			return nil, err
		}
	} else {
		profiles = flattenProfiles(bundle.Profiles)
		if !includeSecrets {
			profiles = stripSecrets(profiles)
		}
	}
	return profiles, nil
}

func (s *Store) ExportProfiles(path string, profiles []models.Profile, includeSecrets bool, passphrase string) error {
	data := flattenProfiles(profiles)
	if includeSecrets && passphrase != "" {
		bundle, err := secrets.EncryptBundle(data, passphrase)
		if err != nil {
			return err
		}
		return writeJSON(path, bundle)
	}
	if !includeSecrets {
		data = stripSecrets(data)
	}
	for i := range data {
		data[i].ExportSettings = nil
		data[i].ImportSettings = nil
	}
	return writeJSON(path, models.ProfileBundle{
		Version:  CurrentVersion,
		Profiles: data,
	})
}

// ProfileConflict describes an imported host that replaces an existing one.
type ProfileConflict struct {
	Imported models.Profile `json:"imported"`
	Existing models.Profile `json:"existing"`
	Reason   string         `json:"reason"`
}

// DetectProfileConflicts finds imported profiles that would overwrite existing hosts.
func DetectProfileConflicts(existing, imported []models.Profile) []ProfileConflict {
	byID := map[string]models.Profile{}
	byName := map[string]models.Profile{}
	for _, p := range existing {
		byID[p.ID] = p
		byName[p.Name] = p
	}
	var conflicts []ProfileConflict
	seen := map[string]bool{}
	for _, p := range imported {
		if ex, ok := byID[p.ID]; ok {
			key := "id:" + p.ID
			if !seen[key] {
				conflicts = append(conflicts, ProfileConflict{Imported: p, Existing: ex, Reason: "id"})
				seen[key] = true
			}
			continue
		}
		if ex, ok := byName[p.Name]; ok {
			key := "name:" + p.Name
			if !seen[key] {
				conflicts = append(conflicts, ProfileConflict{Imported: p, Existing: ex, Reason: "name"})
				seen[key] = true
			}
		}
	}
	return conflicts
}

// MergeProfiles merges imported profiles into existing by ID, then by name.
func MergeProfiles(existing, imported []models.Profile) []models.Profile {
	byID := map[string]int{}
	byName := map[string]int{}
	out := append([]models.Profile(nil), existing...)
	for i, p := range out {
		byID[p.ID] = i
		byName[p.Name] = i
	}
	for _, p := range imported {
		if idx, ok := byID[p.ID]; ok {
			out[idx] = p
			continue
		}
		if idx, ok := byName[p.Name]; ok {
			out[idx] = p
			continue
		}
		out = append(out, p)
		byID[p.ID] = len(out) - 1
		byName[p.Name] = len(out) - 1
	}
	return out
}

func (s *Store) writeJSON(path string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return writeJSON(path, value)
}

func readJSON(path string, value any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(value)
}

func writeJSON(path string, value any) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	tmp := path + ".tmp"
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func flattenProfiles(profiles []models.Profile) []models.Profile {
	var out []models.Profile
	for _, p := range profiles {
		out = append(out, flattenProfile(p)...)
	}
	return out
}

func flattenProfile(p models.Profile) []models.Profile {
	p = normalizeProfile(p)

	if p.ExportSettings != nil || p.ImportSettings != nil {
		export := p
		importP := p
		if p.ExportSettings != nil {
			p.ExportSettings.MigrateQueryFields()
			export = p.ApplySettings(p.ExportSettings)
			export.ID = p.ID
			export.Name = p.Name
			export.Group = p.Group
			export.PluginPath = p.PluginPath
		}
		if p.ImportSettings != nil {
			p.ImportSettings.MigrateQueryFields()
			importP = p.ApplySettings(p.ImportSettings)
			importP.ID = p.ID
			importP.Name = p.Name
			importP.Group = p.Group
			importP.PluginPath = p.PluginPath
			importP.PreImportQuery = p.ImportSettings.PreImportQuery
			importP.RunQueryBeforeImport = p.ImportSettings.RunQueryBeforeImport
			importP.PostImportQuery = p.ImportSettings.PostImportQuery
			importP.RunQueryAfterImport = p.ImportSettings.RunQueryAfterImport
		}

		if models.SettingsEqual(p.ExportSettings, p.ImportSettings) {
			host := normalizeProfile(export)
			host.ExportSettings = nil
			host.ImportSettings = nil
			return []models.Profile{host}
		}

		exportHost := normalizeProfile(export)
		exportHost.ExportSettings = nil
		exportHost.ImportSettings = nil

		importHost := normalizeProfile(importP)
		importHost.ID = fmt.Sprintf("%s-import", p.ID)
		importHost.Name = p.Name + " (import)"
		importHost.ExportSettings = nil
		importHost.ImportSettings = nil
		return []models.Profile{exportHost, importHost}
	}

	p.ExportSettings = nil
	p.ImportSettings = nil
	return []models.Profile{normalizeProfile(p)}
}

func normalizeProfile(p models.Profile) models.Profile {
	if p.ID == "" {
		p.ID = p.Name
	}
	if p.Group == "" {
		p.Group = "Default"
	}
	if p.Port == "" {
		p.Port = "22"
	}
	if p.ConnectionType == "" {
		p.ConnectionType = models.ConnectionTypeSSH
	}
	if p.AuthType == "" {
		p.AuthType = models.AuthTypePassword
	}
	if p.JumpPort == "" {
		p.JumpPort = "22"
	}
	if p.JumpAuthType == "" {
		p.JumpAuthType = models.AuthTypePassword
	}
	if p.DBType == "" {
		p.DBType = models.DBTypeMySQL
	}
	p.DBType = normalizeDBType(p.DBType)
	if p.DBHost == "" {
		p.DBHost = "127.0.0.1"
	}
	if p.DBPort == "" {
		p.DBPort = "3306"
	}
	return p
}

func normalizeDBType(t models.DBType) models.DBType {
	switch t {
	case models.DBTypeMariaDB:
		return models.DBTypeMariaDB
	default:
		return models.DBTypeMySQL
	}
}

func stripSecrets(profiles []models.Profile) []models.Profile {
	for i := range profiles {
		profiles[i].SSHPassword = ""
		profiles[i].JumpPassword = ""
		profiles[i].DBPassword = ""
		profiles[i].WPKey = ""
		profiles[i].AuthKeyPEM = ""
		profiles[i].JumpAuthKeyPEM = ""
	}
	return profiles
}
