package store

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"dback/models"
)

var (
	ErrRemoteDestinationNotFound = errors.New("remote destination not found")
	ErrRemoteDestinationInUse    = errors.New("remote destination is in use")
	ErrAppSettingsDestRequired   = errors.New("app settings destination is required")
)

// DestinationUsage describes where a remote destination is referenced.
type DestinationUsage struct {
	UsedForAppSettings bool
	ProfileIDs         []string
	ProfileNames       []string
}

func (s *Store) LoadRemoteDestinations() ([]models.RemoteDestination, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.unlocked {
		return nil, ErrVaultLocked
	}
	out := make([]models.RemoteDestination, len(s.remoteDestinations))
	for i, d := range s.remoteDestinations {
		out[i] = d.Clone()
	}
	return out, nil
}

func (s *Store) RemoteDestinationByID(id string) (models.RemoteDestination, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.unlocked {
		return models.RemoteDestination{}, ErrVaultLocked
	}
	for _, d := range s.remoteDestinations {
		if d.ID == id {
			return d.Clone(), nil
		}
	}
	return models.RemoteDestination{}, ErrRemoteDestinationNotFound
}

func (s *Store) AppSettingsDestinationID() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.unlocked {
		return "", ErrVaultLocked
	}
	return s.appSettingsDestinationID, nil
}

func (s *Store) SetAppSettingsDestinationID(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.unlocked {
		return ErrVaultLocked
	}
	if strings.TrimSpace(id) == "" {
		s.appSettingsDestinationID = ""
		s.syncLegacyFromAppSettingsLocked()
		s.bumpRevisionLocked()
		return s.persistVaultLocked()
	}
	if !s.hasDestinationLocked(id) {
		return ErrRemoteDestinationNotFound
	}
	s.appSettingsDestinationID = id
	s.syncLegacyFromAppSettingsLocked()
	s.bumpRevisionLocked()
	return s.persistVaultLocked()
}

func (s *Store) SaveRemoteDestination(dest models.RemoteDestination) error {
	if err := models.ValidateRemoteDestination(dest); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.unlocked {
		return ErrVaultLocked
	}
	dest.Name = strings.TrimSpace(dest.Name)
	if dest.S3 != nil {
		dest.S3.Endpoint = strings.TrimSpace(dest.S3.Endpoint)
		dest.S3.Region = strings.TrimSpace(dest.S3.Region)
		dest.S3.Bucket = strings.TrimSpace(dest.S3.Bucket)
		dest.S3.AccessKeyID = strings.TrimSpace(dest.S3.AccessKeyID)
	}
	if dest.ID == "" {
		dest.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	for _, existing := range s.remoteDestinations {
		if existing.ID != dest.ID && strings.EqualFold(existing.Name, dest.Name) {
			return fmt.Errorf("destination name %q already exists", dest.Name)
		}
	}
	for i, existing := range s.remoteDestinations {
		if existing.ID == dest.ID {
			if dest.S3 != nil && strings.TrimSpace(dest.S3.SecretKey) == "" && existing.S3 != nil {
				dest.S3.SecretKey = existing.S3.SecretKey
			}
			s.remoteDestinations[i] = dest.Clone()
			s.syncLegacyFromAppSettingsLocked()
			s.bumpRevisionLocked()
			return s.persistVaultLocked()
		}
	}
	s.remoteDestinations = append(s.remoteDestinations, dest.Clone())
	s.syncLegacyFromAppSettingsLocked()
	s.bumpRevisionLocked()
	return s.persistVaultLocked()
}

func (s *Store) DestinationUsage(id string) (DestinationUsage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.unlocked {
		return DestinationUsage{}, ErrVaultLocked
	}
	if !s.hasDestinationLocked(id) {
		return DestinationUsage{}, ErrRemoteDestinationNotFound
	}
	usage := DestinationUsage{}
	if s.appSettingsDestinationID == id {
		usage.UsedForAppSettings = true
	}
	for _, p := range s.profiles {
		for _, destID := range p.RemoteUploadDestinationIDs {
			if destID == id {
				usage.ProfileIDs = append(usage.ProfileIDs, p.ID)
				usage.ProfileNames = append(usage.ProfileNames, p.Name)
				break
			}
		}
	}
	return usage, nil
}

func (s *Store) DeleteRemoteDestination(id string, force bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.unlocked {
		return ErrVaultLocked
	}
	idx := -1
	for i, d := range s.remoteDestinations {
		if d.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return ErrRemoteDestinationNotFound
	}
	usage := s.destinationUsageLocked(id)
	if !force && (usage.UsedForAppSettings || len(usage.ProfileIDs) > 0) {
		return ErrRemoteDestinationInUse
	}
	if force {
		if usage.UsedForAppSettings {
			s.appSettingsDestinationID = ""
		}
		for i := range s.profiles {
			ids := s.profiles[i].RemoteUploadDestinationIDs
			if len(ids) == 0 {
				continue
			}
			filtered := ids[:0]
			for _, destID := range ids {
				if destID != id {
					filtered = append(filtered, destID)
				}
			}
			s.profiles[i].RemoteUploadDestinationIDs = filtered
		}
	}
	s.remoteDestinations = append(s.remoteDestinations[:idx], s.remoteDestinations[idx+1:]...)
	s.syncLegacyFromAppSettingsLocked()
	s.bumpRevisionLocked()
	return s.persistVaultLocked()
}

func (s *Store) hasDestinationLocked(id string) bool {
	for _, d := range s.remoteDestinations {
		if d.ID == id {
			return true
		}
	}
	return false
}

func (s *Store) destinationUsageLocked(id string) DestinationUsage {
	usage := DestinationUsage{}
	if s.appSettingsDestinationID == id {
		usage.UsedForAppSettings = true
	}
	for _, p := range s.profiles {
		for _, destID := range p.RemoteUploadDestinationIDs {
			if destID == id {
				usage.ProfileIDs = append(usage.ProfileIDs, p.ID)
				usage.ProfileNames = append(usage.ProfileNames, p.Name)
				break
			}
		}
	}
	return usage
}

func (s *Store) syncLegacyFromAppSettingsLocked() {
	if s.appSettingsDestinationID == "" {
		s.sync = nil
		return
	}
	for _, d := range s.remoteDestinations {
		if d.ID == s.appSettingsDestinationID {
			s.sync = d.ToSyncSettings()
			return
		}
	}
	s.sync = nil
}

func migrateRemoteDestinations(payload *models.AppVaultPayload) bool {
	if payload.RemoteDestinationsMigrated {
		return false
	}
	changed := false
	if len(payload.RemoteDestinations) == 0 && models.SyncSettingsConfigured(payload.Sync) {
		id := fmt.Sprintf("%d", time.Now().UnixNano())
		dest := models.RemoteDestinationFromSyncSettings(id, "Default", *payload.Sync)
		payload.RemoteDestinations = []models.RemoteDestination{dest}
		payload.AppSettingsDestinationID = id
		changed = true
	}
	payload.RemoteDestinationsMigrated = true
	return changed
}
