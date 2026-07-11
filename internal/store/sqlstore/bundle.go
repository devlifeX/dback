package sqlstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"dback/internal/secrets"
	"dback/internal/storemodel"
	"dback/models"
)

func (s *Store) ImportProfilesBundle(path string, includeSecrets bool, passphrase string) ([]models.Profile, error) {
	var bundle models.ProfileBundle
	if err := readJSONFile(path, &bundle); err != nil {
		return nil, err
	}
	return s.importProfileBundle(bundle, includeSecrets, passphrase)
}

func (s *Store) ExportProfiles(path string, profiles []models.Profile, includeSecrets bool, passphrase string) error {
	if includeSecrets && passphrase == "" {
		return storemodel.ErrIncludeSecretsNoPassphrase
	}
	data := storemodel.FlattenProfiles(profiles)
	if includeSecrets && passphrase != "" {
		bundle, err := secrets.EncryptBundle(data, passphrase)
		if err != nil {
			return err
		}
		return writeJSON(path, bundle)
	}
	if !includeSecrets {
		data = stripProfileSecrets(data)
	}
	for i := range data {
		data[i].ExportSettings = nil
		data[i].ImportSettings = nil
	}
	return writeJSON(path, models.ProfileBundle{Version: storemodel.CurrentVersion, Profiles: data})
}

func (s *Store) ImportAppDataBundle(path string, includeSecrets bool, passphrase string) (storemodel.AppImportData, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return storemodel.AppImportData{}, err
	}
	return s.ImportAppDataBytes(raw, includeSecrets, passphrase)
}

func (s *Store) ImportAppDataBytes(raw []byte, includeSecrets bool, passphrase string) (storemodel.AppImportData, error) {
	var appBundle models.AppBundle
	if err := json.Unmarshal(raw, &appBundle); err == nil && appBundleHasPayload(appBundle) {
		return s.decodeAppBundle(appBundle, includeSecrets, passphrase)
	}
	var profileBundle models.ProfileBundle
	if err := json.Unmarshal(raw, &profileBundle); err == nil && (profileBundle.Profiles != nil || profileBundle.Encrypted) {
		profiles, err := s.importProfileBundle(profileBundle, includeSecrets, passphrase)
		if err != nil {
			return storemodel.AppImportData{}, err
		}
		return storemodel.AppImportData{Profiles: profiles}, nil
	}
	return storemodel.AppImportData{}, fmt.Errorf("unrecognized app data bundle format")
}

func (s *Store) ExportAppData(path string, data storemodel.AppImportData, includeSecrets bool, passphrase string) error {
	raw, err := s.MarshalAppDataBundle(data, includeSecrets, passphrase)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0600)
}

func (s *Store) MarshalAppDataBundle(data storemodel.AppImportData, includeSecrets bool, passphrase string) ([]byte, error) {
	if includeSecrets && passphrase == "" {
		return nil, storemodel.ErrIncludeSecretsNoPassphrase
	}
	payload := storemodel.AppImportData{
		Profiles:                 storemodel.FlattenProfiles(data.Profiles),
		Templates:                append([]models.SQLTemplate(nil), data.Templates...),
		History:                  append([]models.ExportRecord(nil), data.History...),
		Logs:                     append([]models.LogEntry(nil), data.Logs...),
		Sync:                     data.Sync.Clone(),
		RemoteDestinations:       cloneRemoteDestinations(data.RemoteDestinations),
		AppSettingsDestinationID: data.AppSettingsDestinationID,
	}
	for i := range payload.Profiles {
		payload.Profiles[i].ExportSettings = nil
		payload.Profiles[i].ImportSettings = nil
	}
	if includeSecrets && passphrase != "" {
		bundle, err := secrets.EncryptAppBundle(payload.Profiles, payload.Templates, payload.History, payload.Logs, payload.Sync, payload.RemoteDestinations, payload.AppSettingsDestinationID, passphrase)
		if err != nil {
			return nil, err
		}
		return json.MarshalIndent(bundle, "", "  ")
	}
	if !includeSecrets {
		payload.Profiles = stripProfileSecrets(payload.Profiles)
		payload.RemoteDestinations = stripRemoteDestinationSecrets(payload.RemoteDestinations)
		if payload.Sync != nil {
			payload.Sync = &models.SyncSettings{
				Endpoint: payload.Sync.Endpoint, Region: payload.Sync.Region, Bucket: payload.Sync.Bucket,
				AccessKeyID: payload.Sync.AccessKeyID, UseSSL: payload.Sync.UseSSL,
			}
		}
	}
	bundle := models.AppBundle{
		Version: storemodel.CurrentVersion, ExportedAt: time.Now(),
		Profiles: payload.Profiles, Templates: payload.Templates, History: payload.History, Logs: payload.Logs,
		Sync: payload.Sync, RemoteDestinations: payload.RemoteDestinations, AppSettingsDestinationID: payload.AppSettingsDestinationID,
	}
	return json.MarshalIndent(bundle, "", "  ")
}

func (s *Store) MarshalAppDataBundleForSync(data storemodel.AppImportData) ([]byte, error) {
	s.mu.Lock()
	passphrase, err := s.masterPassphraseLocked()
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return s.MarshalAppDataBundle(data, true, passphrase)
}

func (s *Store) ImportAppDataBundleForSync(raw []byte) (storemodel.AppImportData, error) {
	s.mu.Lock()
	passphrase, err := s.masterPassphraseLocked()
	s.mu.Unlock()
	if err != nil {
		return storemodel.AppImportData{}, err
	}
	return s.ImportAppDataBytes(raw, true, passphrase)
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
		return storemodel.FlattenProfiles(profiles), nil
	}
	profiles := storemodel.FlattenProfiles(bundle.Profiles)
	if !includeSecrets {
		profiles = stripProfileSecrets(profiles)
	}
	return profiles, nil
}

func (s *Store) decodeAppBundle(bundle models.AppBundle, includeSecrets bool, passphrase string) (storemodel.AppImportData, error) {
	if bundle.Encrypted {
		if !includeSecrets {
			return storemodel.AppImportData{}, errors.New("encrypted bundle requires include secrets and passphrase")
		}
		decoded, err := secrets.DecryptAppBundle(bundle, passphrase)
		if err != nil {
			return storemodel.AppImportData{}, err
		}
		return storemodel.AppImportData{
			Profiles: storemodel.FlattenProfiles(decoded.Profiles), Templates: decoded.Templates,
			History: decoded.History, Logs: decoded.Logs, Sync: decoded.Sync.Clone(),
			RemoteDestinations:       cloneRemoteDestinations(decoded.RemoteDestinations),
			AppSettingsDestinationID: decoded.AppSettingsDestinationID,
		}, nil
	}
	profiles := storemodel.FlattenProfiles(bundle.Profiles)
	if !includeSecrets {
		profiles = stripProfileSecrets(profiles)
	}
	return storemodel.AppImportData{
		Profiles: profiles, Templates: append([]models.SQLTemplate(nil), bundle.Templates...),
		History: append([]models.ExportRecord(nil), bundle.History...),
		Logs:    append([]models.LogEntry(nil), bundle.Logs...), Sync: bundle.Sync.Clone(),
		RemoteDestinations:       cloneRemoteDestinations(bundle.RemoteDestinations),
		AppSettingsDestinationID: bundle.AppSettingsDestinationID,
	}, nil
}

func appBundleHasPayload(b models.AppBundle) bool {
	return b.Encrypted || len(b.Profiles) > 0 || len(b.Templates) > 0 || len(b.History) > 0 || len(b.Logs) > 0 ||
		len(b.RemoteDestinations) > 0 || b.AppSettingsDestinationID != "" || b.Sync != nil
}

func stripProfileSecrets(profiles []models.Profile) []models.Profile {
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

func stripRemoteDestinationSecrets(destinations []models.RemoteDestination) []models.RemoteDestination {
	if len(destinations) == 0 {
		return nil
	}
	out := make([]models.RemoteDestination, len(destinations))
	for i, d := range destinations {
		out[i] = d.Clone()
		if out[i].S3 != nil {
			out[i].S3.SecretKey = ""
		}
	}
	return out
}

func writeJSON(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}
