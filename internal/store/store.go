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

	unlocked  bool
	dataKey   []byte
	masterKey []byte
	vaultSalt string
	revision  uint64

	profiles            []models.Profile
	templates           []models.SQLTemplate
	history             []models.ExportRecord
	logs                []models.LogEntry
	sync                *models.SyncSettings
	syncActivity        models.SyncActivity
	importDestByProfile map[string]string
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
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.unlocked {
		return nil, ErrVaultLocked
	}
	return append([]models.Profile(nil), s.profiles...), nil
}

func (s *Store) SaveProfiles(profiles []models.Profile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.unlocked {
		return ErrVaultLocked
	}
	s.profiles = flattenProfiles(profiles)
	for i := range s.profiles {
		s.profiles[i].ExportSettings = nil
		s.profiles[i].ImportSettings = nil
	}
	s.bumpRevisionLocked()
	return s.persistVaultLocked()
}

func (s *Store) LoadTemplates() ([]models.SQLTemplate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.unlocked {
		return nil, ErrVaultLocked
	}
	return append([]models.SQLTemplate(nil), s.templates...), nil
}

func (s *Store) SaveTemplates(templates []models.SQLTemplate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.unlocked {
		return ErrVaultLocked
	}
	s.templates = append([]models.SQLTemplate(nil), templates...)
	s.bumpRevisionLocked()
	return s.persistVaultLocked()
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
			Description: "Create admin user devlife",
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
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.unlocked {
		return nil, ErrVaultLocked
	}
	return append([]models.ExportRecord(nil), s.history...), nil
}

func (s *Store) SaveHistory(records []models.ExportRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.unlocked {
		return ErrVaultLocked
	}
	s.history = append([]models.ExportRecord(nil), records...)
	s.bumpRevisionLocked()
	return s.persistVaultLocked()
}

func (s *Store) LoadLogs() ([]models.LogEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.unlocked {
		return nil, ErrVaultLocked
	}
	return append([]models.LogEntry(nil), s.logs...), nil
}

func (s *Store) SaveLogs(entries []models.LogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.unlocked {
		return ErrVaultLocked
	}
	s.logs = append([]models.LogEntry(nil), entries...)
	s.bumpRevisionLocked()
	return s.persistVaultLocked()
}

func (s *Store) LoadSyncSettings() (*models.SyncSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.unlocked {
		return nil, ErrVaultLocked
	}
	return s.sync.Clone(), nil
}

func (s *Store) SaveSyncSettings(settings models.SyncSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.unlocked {
		return ErrVaultLocked
	}
	s.sync = settings.Clone()
	s.bumpRevisionLocked()
	return s.persistVaultLocked()
}

func (s *Store) LoadSyncActivity() (models.SyncActivity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.unlocked {
		return models.SyncActivity{}, ErrVaultLocked
	}
	return s.syncActivity, nil
}

func (s *Store) RecordSyncPush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.unlocked {
		return ErrVaultLocked
	}
	s.syncActivity.LastPushAt = time.Now()
	s.bumpRevisionLocked()
	return s.persistVaultLocked()
}

func (s *Store) RecordSyncPull() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.unlocked {
		return ErrVaultLocked
	}
	s.syncActivity.LastPullAt = time.Now()
	s.bumpRevisionLocked()
	return s.persistVaultLocked()
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

// ImportDestForProfile returns the last chosen import destination for a source host profile.
func (s *Store) ImportDestForProfile(sourceProfileID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.unlocked {
		return ""
	}
	return s.importDestByProfile[sourceProfileID]
}

// SetImportDestForProfile remembers the import destination for a source host profile.
func (s *Store) SetImportDestForProfile(sourceProfileID, destProfileID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.unlocked {
		return ErrVaultLocked
	}
	if sourceProfileID == "" || destProfileID == "" {
		return nil
	}
	if s.importDestByProfile == nil {
		s.importDestByProfile = map[string]string{}
	}
	s.importDestByProfile[sourceProfileID] = destProfileID
	s.bumpRevisionLocked()
	return s.persistVaultLocked()
}

// ValidateMasterPassphrase checks the passphrase against the vault unlock key.
func (s *Store) ValidateMasterPassphrase(passphrase string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.unlocked {
		return ErrVaultLocked
	}
	cached, err := s.masterPassphraseLocked()
	if err != nil {
		return err
	}
	if passphrase != cached {
		return ErrWrongMasterKey
	}
	return nil
}

// ImportProfilesBundle loads profiles from a bundle file (plain or encrypted).
func (s *Store) ImportProfilesBundle(path string, includeSecrets bool, passphrase string) ([]models.Profile, error) {
	var bundle models.ProfileBundle
	if err := readJSON(path, &bundle); err != nil {
		return nil, err
	}
	return s.importProfileBundle(bundle, includeSecrets, passphrase)
}

func (s *Store) ExportProfiles(path string, profiles []models.Profile, includeSecrets bool, passphrase string) error {
	if includeSecrets && passphrase == "" {
		return ErrIncludeSecretsNoPassphrase
	}
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

// AppImportData holds decoded app bundle contents for merge preview.
type AppImportData struct {
	Profiles  []models.Profile
	Templates []models.SQLTemplate
	History   []models.ExportRecord
	Logs      []models.LogEntry
	Sync      *models.SyncSettings
}

// TemplateConflict describes an imported template that replaces an existing one.
type TemplateConflict struct {
	Imported models.SQLTemplate `json:"imported"`
	Existing models.SQLTemplate `json:"existing"`
	Reason   string             `json:"reason"`
}

// ImportAppDataBundle loads app data from a bundle file (plain, encrypted, or legacy profile bundle).
func (s *Store) ImportAppDataBundle(path string, includeSecrets bool, passphrase string) (AppImportData, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return AppImportData{}, err
	}
	return s.ImportAppDataBytes(raw, includeSecrets, passphrase)
}

// ImportAppDataBytes loads app data from a JSON bundle in memory.
func (s *Store) ImportAppDataBytes(raw []byte, includeSecrets bool, passphrase string) (AppImportData, error) {
	var appBundle models.AppBundle
	if err := json.Unmarshal(raw, &appBundle); err == nil && appBundleHasPayload(appBundle) {
		return s.decodeAppBundle(appBundle, includeSecrets, passphrase)
	}

	var profileBundle models.ProfileBundle
	if err := json.Unmarshal(raw, &profileBundle); err == nil && (profileBundle.Profiles != nil || profileBundle.Encrypted) {
		profiles, err := s.importProfileBundle(profileBundle, includeSecrets, passphrase)
		if err != nil {
			return AppImportData{}, err
		}
		return AppImportData{Profiles: profiles}, nil
	}

	return AppImportData{}, fmt.Errorf("unrecognized app data bundle format")
}

func appBundleHasPayload(b models.AppBundle) bool {
	return b.Encrypted || len(b.Profiles) > 0 || len(b.Templates) > 0 || len(b.History) > 0 || len(b.Logs) > 0
}

func (s *Store) importProfileBundle(bundle models.ProfileBundle, includeSecrets bool, passphrase string) ([]models.Profile, error) {
	if bundle.Encrypted {
		if !includeSecrets {
			return nil, errors.New("encrypted bundle requires include secrets and passphrase")
		}
		profiles, err := secrets.DecryptBundle(bundle, passphrase)
		if err != nil {
			return nil, err
		}
		return flattenProfiles(profiles), nil
	}
	profiles := flattenProfiles(bundle.Profiles)
	if !includeSecrets {
		profiles = stripSecrets(profiles)
	}
	return profiles, nil
}

func (s *Store) decodeAppBundle(bundle models.AppBundle, includeSecrets bool, passphrase string) (AppImportData, error) {
	if bundle.Encrypted {
		if !includeSecrets {
			return AppImportData{}, errors.New("encrypted bundle requires include secrets and passphrase")
		}
		decoded, err := secrets.DecryptAppBundle(bundle, passphrase)
		if err != nil {
			return AppImportData{}, err
		}
		return AppImportData{
			Profiles:  flattenProfiles(decoded.Profiles),
			Templates: decoded.Templates,
			History:   decoded.History,
			Logs:      decoded.Logs,
			Sync:      decoded.Sync.Clone(),
		}, nil
	}
	profiles := flattenProfiles(bundle.Profiles)
	if !includeSecrets {
		profiles = stripSecrets(profiles)
	}
	return AppImportData{
		Profiles:  profiles,
		Templates: append([]models.SQLTemplate(nil), bundle.Templates...),
		History:   append([]models.ExportRecord(nil), bundle.History...),
		Logs:      append([]models.LogEntry(nil), bundle.Logs...),
		Sync:      bundle.Sync.Clone(),
	}, nil
}

func (s *Store) ExportAppData(path string, data AppImportData, includeSecrets bool, passphrase string) error {
	raw, err := s.MarshalAppDataBundle(data, includeSecrets, passphrase)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0600)
}

func (s *Store) MarshalAppDataBundle(data AppImportData, includeSecrets bool, passphrase string) ([]byte, error) {
	if includeSecrets && passphrase == "" {
		return nil, ErrIncludeSecretsNoPassphrase
	}
	payload := AppImportData{
		Profiles:  flattenProfiles(data.Profiles),
		Templates: append([]models.SQLTemplate(nil), data.Templates...),
		History:   append([]models.ExportRecord(nil), data.History...),
		Logs:      append([]models.LogEntry(nil), data.Logs...),
		Sync:      data.Sync.Clone(),
	}
	for i := range payload.Profiles {
		payload.Profiles[i].ExportSettings = nil
		payload.Profiles[i].ImportSettings = nil
	}
	if includeSecrets && passphrase != "" {
		bundle, err := secrets.EncryptAppBundle(payload.Profiles, payload.Templates, payload.History, payload.Logs, payload.Sync, passphrase)
		if err != nil {
			return nil, err
		}
		return json.MarshalIndent(bundle, "", "  ")
	}
	if !includeSecrets {
		payload.Profiles = stripSecrets(payload.Profiles)
		if payload.Sync != nil {
			payload.Sync = &models.SyncSettings{
				Endpoint:    payload.Sync.Endpoint,
				Region:      payload.Sync.Region,
				Bucket:      payload.Sync.Bucket,
				AccessKeyID: payload.Sync.AccessKeyID,
				UseSSL:      payload.Sync.UseSSL,
			}
		}
	}
	bundle := models.AppBundle{
		Version:    CurrentVersion,
		ExportedAt: time.Now(),
		Profiles:   payload.Profiles,
		Templates:  payload.Templates,
		History:    payload.History,
		Logs:       payload.Logs,
		Sync:       payload.Sync,
	}
	return json.MarshalIndent(bundle, "", "  ")
}

// MarshalAppDataBundleForSync encrypts the current app data with the cached master key.
func (s *Store) MarshalAppDataBundleForSync(data AppImportData) ([]byte, error) {
	s.mu.Lock()
	passphrase, err := s.masterPassphraseLocked()
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return s.MarshalAppDataBundle(data, true, passphrase)
}

// ImportAppDataBundleForSync decrypts a sync bundle using the cached master key.
func (s *Store) ImportAppDataBundleForSync(raw []byte) (AppImportData, error) {
	s.mu.Lock()
	passphrase, err := s.masterPassphraseLocked()
	s.mu.Unlock()
	if err != nil {
		return AppImportData{}, err
	}
	return s.ImportAppDataBytes(raw, true, passphrase)
}

func DetectTemplateConflicts(existing, imported []models.SQLTemplate) []TemplateConflict {
	byID := map[string]models.SQLTemplate{}
	byName := map[string]models.SQLTemplate{}
	for _, t := range existing {
		byID[t.ID] = t
		byName[t.Name] = t
	}
	var conflicts []TemplateConflict
	seen := map[string]bool{}
	for _, t := range imported {
		if ex, ok := byID[t.ID]; ok {
			key := "id:" + t.ID
			if !seen[key] {
				conflicts = append(conflicts, TemplateConflict{Imported: t, Existing: ex, Reason: "id"})
				seen[key] = true
			}
			continue
		}
		if ex, ok := byName[t.Name]; ok {
			key := "name:" + t.Name
			if !seen[key] {
				conflicts = append(conflicts, TemplateConflict{Imported: t, Existing: ex, Reason: "name"})
				seen[key] = true
			}
		}
	}
	return conflicts
}

func MergeTemplates(existing, imported []models.SQLTemplate) []models.SQLTemplate {
	byID := map[string]int{}
	byName := map[string]int{}
	out := append([]models.SQLTemplate(nil), existing...)
	for i, t := range out {
		byID[t.ID] = i
		byName[t.Name] = i
	}
	for _, t := range imported {
		if idx, ok := byID[t.ID]; ok {
			out[idx] = t
			continue
		}
		if idx, ok := byName[t.Name]; ok {
			out[idx] = t
			continue
		}
		out = append(out, t)
		byID[t.ID] = len(out) - 1
		byName[t.Name] = len(out) - 1
	}
	return out
}

func MergeHistory(existing, imported []models.ExportRecord) []models.ExportRecord {
	seen := map[string]bool{}
	out := append([]models.ExportRecord(nil), existing...)
	for _, r := range out {
		if r.ID != "" {
			seen[r.ID] = true
		}
	}
	for _, r := range imported {
		if r.ID != "" && seen[r.ID] {
			continue
		}
		out = append(out, r)
		if r.ID != "" {
			seen[r.ID] = true
		}
	}
	return out
}

func MergeLogs(existing, imported []models.LogEntry) []models.LogEntry {
	seen := map[string]bool{}
	out := append([]models.LogEntry(nil), existing...)
	for _, e := range out {
		if e.ID != "" {
			seen[e.ID] = true
		}
	}
	for _, e := range imported {
		if e.ID != "" && seen[e.ID] {
			continue
		}
		out = append(out, e)
		if e.ID != "" {
			seen[e.ID] = true
		}
	}
	return out
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
		}
		if p.ImportSettings != nil {
			p.ImportSettings.MigrateQueryFields()
			importP = p.ApplySettings(p.ImportSettings)
			importP.ID = p.ID
			importP.Name = p.Name
			importP.Group = p.Group
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
	if p.ConnectionType == models.ConnectionTypeWordPress {
		if p.WPUrl != "" && p.Host == "" {
			p.Host = p.WPUrl
		}
		if p.WPUrl == "" && p.Host != "" {
			p.WPUrl = p.Host
		}
		if p.DBType == "" {
			p.DBType = models.DBTypeMySQL
		}
		return p
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
	p.FileBackupCompression = models.NormalizeArchiveCompression(p.FileBackupCompression)
	for i := range p.FileBackupPaths {
		_ = p.FileBackupPaths[i].Normalize()
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
		profiles[i].AuthKeyPEM = ""
		profiles[i].JumpAuthKeyPEM = ""
		profiles[i].WPKey = ""
	}
	return profiles
}
