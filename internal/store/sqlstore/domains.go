package sqlstore

import (
	"fmt"
	"strings"
	"time"

	"dback/internal/storemodel"
	"dback/models"

	"github.com/google/uuid"
)

func (s *Store) LoadProfiles() ([]models.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return nil, err
	}
	return append([]models.Profile(nil), s.profiles...), nil
}

func (s *Store) SaveProfiles(profiles []models.Profile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	s.profiles = storemodel.FlattenProfiles(profiles)
	s.bumpRevisionLocked()
	return s.persistAllLocked()
}

func (s *Store) LoadTemplates() ([]models.SQLTemplate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return nil, err
	}
	return append([]models.SQLTemplate(nil), s.templates...), nil
}

func (s *Store) SaveTemplates(templates []models.SQLTemplate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	s.templates = append([]models.SQLTemplate(nil), templates...)
	s.bumpRevisionLocked()
	return s.persistAllLocked()
}

func (s *Store) LoadHistory() ([]models.ExportRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return nil, err
	}
	return append([]models.ExportRecord(nil), s.history...), nil
}

func (s *Store) SaveHistory(records []models.ExportRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	s.history = append([]models.ExportRecord(nil), records...)
	s.bumpRevisionLocked()
	return s.persistAllLocked()
}

func (s *Store) LoadLogs() ([]models.LogEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return nil, err
	}
	return append([]models.LogEntry(nil), s.logs...), nil
}

func (s *Store) SaveLogs(entries []models.LogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	s.logs = append([]models.LogEntry(nil), entries...)
	s.bumpRevisionLocked()
	return s.persistAllLocked()
}

func (s *Store) LoadSyncSettings() (*models.SyncSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return nil, err
	}
	if s.sync == nil {
		return nil, nil
	}
	return s.sync.Clone(), nil
}

func (s *Store) SaveSyncSettings(settings models.SyncSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	if strings.TrimSpace(settings.SecretKey) == "" && s.sync != nil {
		settings.SecretKey = s.sync.SecretKey
	}
	s.sync = settings.Clone()
	if s.appSettingsDestinationID != "" {
		for i, d := range s.remoteDestinations {
			if d.ID == s.appSettingsDestinationID {
				s.remoteDestinations[i] = models.RemoteDestinationFromSyncSettings(d.ID, d.Name, settings)
				break
			}
		}
	} else if len(s.remoteDestinations) == 0 && models.SyncSettingsConfigured(&settings) {
		id := fmt.Sprintf("%d", time.Now().UnixNano())
		dest := models.RemoteDestinationFromSyncSettings(id, "Default", settings)
		s.remoteDestinations = []models.RemoteDestination{dest}
		s.appSettingsDestinationID = id
		s.remoteDestinationsMigrated = true
	}
	s.syncLegacyFromAppSettingsLocked()
	s.bumpRevisionLocked()
	return s.persistAllLocked()
}

func (s *Store) LoadSyncActivity() (models.SyncActivity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return models.SyncActivity{}, err
	}
	return s.syncActivity, nil
}

func (s *Store) RecordSyncPush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	s.syncActivity.LastPushAt = time.Now()
	s.bumpRevisionLocked()
	return s.persistAllLocked()
}

func (s *Store) RecordSyncPull() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	s.syncActivity.LastPullAt = time.Now()
	s.bumpRevisionLocked()
	return s.persistAllLocked()
}

func (s *Store) ImportDestForProfile(sourceProfileID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.unlocked {
		return ""
	}
	return s.importDestByProfile[sourceProfileID]
}

func (s *Store) SetImportDestForProfile(sourceProfileID, destProfileID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	if sourceProfileID == "" || destProfileID == "" {
		return nil
	}
	if s.importDestByProfile == nil {
		s.importDestByProfile = map[string]string{}
	}
	s.importDestByProfile[sourceProfileID] = destProfileID
	s.bumpRevisionLocked()
	return s.persistAllLocked()
}

func (s *Store) HostSort() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.unlocked {
		return ""
	}
	return s.hostSort
}

func (s *Store) SetHostSort(sort string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	s.hostSort = sort
	s.bumpRevisionLocked()
	return s.persistAllLocked()
}

func (s *Store) LoadRemoteDestinations() ([]models.RemoteDestination, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return nil, err
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
	if err := s.requireUnlocked(); err != nil {
		return models.RemoteDestination{}, err
	}
	for _, d := range s.remoteDestinations {
		if d.ID == id {
			return d.Clone(), nil
		}
	}
	return models.RemoteDestination{}, storemodel.ErrRemoteDestinationNotFound
}

func (s *Store) AppSettingsDestinationID() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return "", err
	}
	return s.appSettingsDestinationID, nil
}

func (s *Store) SetAppSettingsDestinationID(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	if strings.TrimSpace(id) == "" {
		s.appSettingsDestinationID = ""
		s.syncLegacyFromAppSettingsLocked()
		s.bumpRevisionLocked()
		return s.persistAllLocked()
	}
	if !s.hasDestinationLocked(id) {
		return storemodel.ErrRemoteDestinationNotFound
	}
	s.appSettingsDestinationID = id
	s.syncLegacyFromAppSettingsLocked()
	s.bumpRevisionLocked()
	return s.persistAllLocked()
}

func (s *Store) SaveRemoteDestination(dest models.RemoteDestination) error {
	if err := models.ValidateRemoteDestination(dest); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
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
			return s.persistAllLocked()
		}
	}
	s.remoteDestinations = append(s.remoteDestinations, dest.Clone())
	s.syncLegacyFromAppSettingsLocked()
	s.bumpRevisionLocked()
	return s.persistAllLocked()
}

func (s *Store) DestinationUsage(id string) (storemodel.DestinationUsage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return storemodel.DestinationUsage{}, err
	}
	if !s.hasDestinationLocked(id) {
		return storemodel.DestinationUsage{}, storemodel.ErrRemoteDestinationNotFound
	}
	return s.destinationUsageLocked(id), nil
}

func (s *Store) DeleteRemoteDestination(id string, force bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	idx := -1
	for i, d := range s.remoteDestinations {
		if d.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return storemodel.ErrRemoteDestinationNotFound
	}
	usage := s.destinationUsageLocked(id)
	if !force && (usage.UsedForAppSettings || len(usage.ProfileIDs) > 0) {
		return storemodel.ErrRemoteDestinationInUse
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
	return s.persistAllLocked()
}

func (s *Store) hasDestinationLocked(id string) bool {
	for _, d := range s.remoteDestinations {
		if d.ID == id {
			return true
		}
	}
	return false
}

func (s *Store) destinationUsageLocked(id string) storemodel.DestinationUsage {
	usage := storemodel.DestinationUsage{}
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

func (s *Store) ListTasks() ([]models.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return nil, err
	}
	return append([]models.Task{}, s.tasks...), nil
}

func (s *Store) GetTask(id string) (models.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return models.Task{}, err
	}
	for _, t := range s.tasks {
		if t.ID == id {
			return t, nil
		}
	}
	return models.Task{}, storemodel.ErrTaskNotFound
}

func (s *Store) SaveTask(task models.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	if task.ID == "" {
		task.ID = uuid.NewString()
	}
	for i, existing := range s.tasks {
		if existing.ID == task.ID {
			s.tasks[i] = task
			s.bumpRevisionLocked()
			return s.persistAllLocked()
		}
	}
	s.tasks = append(s.tasks, task)
	s.bumpRevisionLocked()
	return s.persistAllLocked()
}

func (s *Store) SetTaskEnabled(id string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	for i, t := range s.tasks {
		if t.ID == id {
			s.tasks[i].Enabled = enabled
			s.bumpRevisionLocked()
			return s.persistAllLocked()
		}
	}
	return storemodel.ErrTaskNotFound
}

func (s *Store) DeleteTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	for i, t := range s.tasks {
		if t.ID == id {
			s.tasks = append(s.tasks[:i], s.tasks[i+1:]...)
			s.bumpRevisionLocked()
			return s.persistAllLocked()
		}
	}
	return storemodel.ErrTaskNotFound
}

func (s *Store) ListTaskRuns(taskID string, limit int) ([]models.TaskRunRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}
	var out []models.TaskRunRecord
	for i := len(s.taskRuns) - 1; i >= 0 && len(out) < limit; i-- {
		if taskID == "" || s.taskRuns[i].TaskID == taskID {
			out = append(out, s.taskRuns[i])
		}
	}
	return out, nil
}

func (s *Store) AppendTaskRun(run models.TaskRunRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	if run.ID == "" {
		run.ID = uuid.NewString()
	}
	s.taskRuns = append(s.taskRuns, run)
	if len(s.taskRuns) > maxTaskRuns {
		s.taskRuns = s.taskRuns[len(s.taskRuns)-maxTaskRuns:]
	}
	s.bumpRevisionLocked()
	return s.persistAllLocked()
}

func (s *Store) UpdateTaskState(id string, fn func(*models.Task) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	for i := range s.tasks {
		if s.tasks[i].ID != id {
			continue
		}
		if err := fn(&s.tasks[i]); err != nil {
			return err
		}
		s.bumpRevisionLocked()
		return s.persistAllLocked()
	}
	return fmt.Errorf("%w: %s", storemodel.ErrTaskNotFound, id)
}

func (s *Store) ListNotifyChannels() ([]models.NotifyChannel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return nil, err
	}
	return append([]models.NotifyChannel{}, s.notifyChannels...), nil
}

func (s *Store) GetNotifyChannel(id string) (models.NotifyChannel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return models.NotifyChannel{}, err
	}
	for _, ch := range s.notifyChannels {
		if ch.ID == id {
			return ch, nil
		}
	}
	return models.NotifyChannel{}, storemodel.ErrNotifyChannelNotFound
}

func (s *Store) SaveNotifyChannel(ch models.NotifyChannel) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	if ch.ID == "" {
		ch.ID = uuid.NewString()
	}
	for i, existing := range s.notifyChannels {
		if existing.ID == ch.ID {
			s.notifyChannels[i] = ch
			s.bumpRevisionLocked()
			return s.persistAllLocked()
		}
	}
	s.notifyChannels = append(s.notifyChannels, ch)
	s.bumpRevisionLocked()
	return s.persistAllLocked()
}

func (s *Store) DeleteNotifyChannel(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	for i, ch := range s.notifyChannels {
		if ch.ID == id {
			s.notifyChannels = append(s.notifyChannels[:i], s.notifyChannels[i+1:]...)
			s.bumpRevisionLocked()
			return s.persistAllLocked()
		}
	}
	return storemodel.ErrNotifyChannelNotFound
}
