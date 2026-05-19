package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"dback/models"
)

const CurrentVersion = 2

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

func (s *Store) ProfilesPath() string {
	return filepath.Join(s.baseDir, "profiles.json")
}

func (s *Store) HistoryPath() string {
	return filepath.Join(s.baseDir, "export_history.json")
}

func (s *Store) LogsPath() string {
	return filepath.Join(s.baseDir, "logs.json")
}

func (s *Store) LoadProfiles() ([]models.Profile, error) {
	var bundle models.ProfileBundle
	if err := readJSON(s.ProfilesPath(), &bundle); err == nil && bundle.Profiles != nil {
		return migrateProfiles(bundle.Profiles), nil
	} else if !errors.Is(err, os.ErrNotExist) {
		var legacy models.AppConfig
		if legacyErr := readJSON(s.ProfilesPath(), &legacy); legacyErr == nil {
			return migrateProfiles(legacy.Profiles), nil
		}
		return nil, err
	}
	return []models.Profile{}, nil
}

func (s *Store) SaveProfiles(profiles []models.Profile) error {
	return s.writeJSON(s.ProfilesPath(), models.ProfileBundle{
		Version:  CurrentVersion,
		Profiles: migrateProfiles(profiles),
	})
}

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

func (s *Store) ImportProfiles(path string, includeSecrets bool) ([]models.Profile, error) {
	var bundle models.ProfileBundle
	if err := readJSON(path, &bundle); err != nil {
		return nil, err
	}
	profiles := migrateProfiles(bundle.Profiles)
	if !includeSecrets {
		profiles = stripSecrets(profiles)
	}
	return profiles, nil
}

func (s *Store) ExportProfiles(path string, profiles []models.Profile, includeSecrets bool) error {
	data := migrateProfiles(profiles)
	if !includeSecrets {
		data = stripSecrets(data)
	}
	return writeJSON(path, models.ProfileBundle{
		Version:  CurrentVersion,
		Profiles: data,
	})
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
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
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

func migrateProfiles(profiles []models.Profile) []models.Profile {
	for i := range profiles {
		p := &profiles[i]
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
		if p.ExportSettings == nil {
			settings := models.SettingsFromProfile(*p)
			p.ExportSettings = &settings
		}
		if p.ImportSettings == nil {
			settings := models.SettingsFromProfile(*p)
			p.ImportSettings = &settings
		}
	}
	return profiles
}

func stripSecrets(profiles []models.Profile) []models.Profile {
	for i := range profiles {
		profiles[i].SSHPassword = ""
		profiles[i].JumpPassword = ""
		profiles[i].DBPassword = ""
		profiles[i].WPKey = ""
		if profiles[i].ExportSettings != nil {
			profiles[i].ExportSettings.SSHPassword = ""
			profiles[i].ExportSettings.JumpPassword = ""
			profiles[i].ExportSettings.DBPassword = ""
			profiles[i].ExportSettings.WPKey = ""
		}
		if profiles[i].ImportSettings != nil {
			profiles[i].ImportSettings.SSHPassword = ""
			profiles[i].ImportSettings.JumpPassword = ""
			profiles[i].ImportSettings.DBPassword = ""
			profiles[i].ImportSettings.WPKey = ""
		}
	}
	return profiles
}
