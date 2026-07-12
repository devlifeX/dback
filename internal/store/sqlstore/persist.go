package sqlstore

import (
	"database/sql"
	"encoding/json"
	"time"

	"dback/internal/secrets"
	"dback/internal/storemodel"
	"dback/models"
)

func (s *Store) loadAllLocked() error {
	enc := s.fieldEnc()
	var rev int64
	var hostSort, appDestID string
	var migrated int
	var syncActivityJSON string
	if err := s.db.QueryRow(`SELECT revision, host_sort, app_settings_destination_id, remote_destinations_migrated, sync_activity_json FROM app_settings WHERE id = 1`).Scan(&rev, &hostSort, &appDestID, &migrated, &syncActivityJSON); err != nil && err != sql.ErrNoRows {
		return err
	}
	s.revision = uint64(rev)
	s.hostSort = hostSort
	s.appSettingsDestinationID = appDestID
	s.remoteDestinationsMigrated = migrated != 0
	_ = json.Unmarshal([]byte(syncActivityJSON), &s.syncActivity)

	s.profiles, _ = s.loadProfilesLocked(enc)
	s.templates, _ = s.loadTemplatesLocked()
	s.history, _ = s.loadHistoryLocked()
	s.logs, _ = s.loadLogsLocked()
	s.remoteDestinations, _ = s.loadDestinationsLocked(enc)
	s.tasks, _ = s.loadTasksLocked()
	s.taskRuns, _ = s.loadTaskRunsLocked()
	s.notifyChannels, _ = s.loadNotifyChannelsLocked(enc)
	s.users, _ = s.loadUsersLocked(enc)
	s.authSettings, _ = s.loadAuthSettingsLocked(enc)
	s.importDestByProfile, _ = s.loadImportPrefsLocked()
	s.sync, _ = s.loadSyncSettingsLocked(enc)
	s.syncLegacyFromAppSettingsLocked()
	if len(s.templates) == 0 {
		s.templates = seedTemplates()
	}
	if s.importDestByProfile == nil {
		s.importDestByProfile = map[string]string{}
	}
	return nil
}

func (s *Store) persistAllLocked() error {
	enc := s.fieldEnc()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := s.replaceProfilesTx(tx, enc); err != nil {
		return err
	}
	if err := s.replaceTemplatesTx(tx); err != nil {
		return err
	}
	if err := s.replaceHistoryTx(tx); err != nil {
		return err
	}
	if err := s.replaceLogsTx(tx); err != nil {
		return err
	}
	if err := s.replaceDestinationsTx(tx, enc); err != nil {
		return err
	}
	if err := s.replaceTasksTx(tx); err != nil {
		return err
	}
	if err := s.replaceTaskRunsTx(tx); err != nil {
		return err
	}
	if err := s.replaceNotifyChannelsTx(tx, enc); err != nil {
		return err
	}
	if err := s.replaceUsersTx(tx, enc); err != nil {
		return err
	}
	if err := s.replaceAuthSettingsTx(tx, enc); err != nil {
		return err
	}
	if err := s.replaceImportPrefsTx(tx); err != nil {
		return err
	}
	if err := s.replaceSyncSettingsTx(tx, enc); err != nil {
		return err
	}
	actJSON, _ := json.Marshal(s.syncActivity)
	_, err = tx.Exec(`UPDATE app_settings SET host_sort=?, app_settings_destination_id=?, remote_destinations_migrated=?, sync_activity_json=?, revision=? WHERE id=1`,
		s.hostSort, s.appSettingsDestinationID, boolToInt(s.remoteDestinationsMigrated), string(actJSON), s.revision)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (s *Store) loadProfilesLocked(enc *secrets.FieldEncryptor) ([]models.Profile, error) {
	rows, err := s.db.Query(`SELECT data_json FROM profiles`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Profile
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var p models.Profile
		if err := json.Unmarshal([]byte(raw), &p); err != nil {
			return nil, err
		}
		p, err = decryptProfile(p, enc)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) replaceProfilesTx(tx *sql.Tx, enc *secrets.FieldEncryptor) error {
	if _, err := tx.Exec(`DELETE FROM profiles`); err != nil {
		return err
	}
	profiles := storemodel.FlattenProfiles(s.profiles)
	for _, p := range profiles {
		p.ExportSettings = nil
		p.ImportSettings = nil
		ep, err := encryptProfile(p, enc)
		if err != nil {
			return err
		}
		raw, err := json.Marshal(ep)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO profiles (id, data_json) VALUES (?, ?)`, ep.ID, string(raw)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) loadTemplatesLocked() ([]models.SQLTemplate, error) {
	rows, err := s.db.Query(`SELECT data_json FROM sql_templates`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.SQLTemplate
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var t models.SQLTemplate
		if err := json.Unmarshal([]byte(raw), &t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) replaceTemplatesTx(tx *sql.Tx) error {
	if _, err := tx.Exec(`DELETE FROM sql_templates`); err != nil {
		return err
	}
	for _, t := range s.templates {
		raw, err := json.Marshal(t)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO sql_templates (id, data_json) VALUES (?, ?)`, t.ID, string(raw)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) loadHistoryLocked() ([]models.ExportRecord, error) {
	rows, err := s.db.Query(`SELECT data_json FROM export_records ORDER BY export_date`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ExportRecord
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var r models.ExportRecord
		if err := json.Unmarshal([]byte(raw), &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) replaceHistoryTx(tx *sql.Tx) error {
	if _, err := tx.Exec(`DELETE FROM export_records`); err != nil {
		return err
	}
	for _, r := range s.history {
		raw, err := json.Marshal(r)
		if err != nil {
			return err
		}
		date := r.ExportDate.UTC().Format(time.RFC3339Nano)
		if r.ID == "" {
			r.ID = newID()
		}
		if _, err := tx.Exec(`INSERT INTO export_records (id, profile_id, export_date, data_json) VALUES (?, ?, ?, ?)`, r.ID, r.ProfileID, date, string(raw)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) loadLogsLocked() ([]models.LogEntry, error) {
	rows, err := s.db.Query(`SELECT data_json FROM log_entries ORDER BY timestamp`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.LogEntry
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var e models.LogEntry
		if err := json.Unmarshal([]byte(raw), &e); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) replaceLogsTx(tx *sql.Tx) error {
	if _, err := tx.Exec(`DELETE FROM log_entries`); err != nil {
		return err
	}
	for _, e := range s.logs {
		raw, err := json.Marshal(e)
		if err != nil {
			return err
		}
		ts := e.Timestamp.UTC().Format(time.RFC3339Nano)
		if e.ID == "" {
			e.ID = newID()
		}
		if _, err := tx.Exec(`INSERT INTO log_entries (id, profile_id, timestamp, data_json) VALUES (?, ?, ?, ?)`, e.ID, e.ProfileID, ts, string(raw)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) loadDestinationsLocked(enc *secrets.FieldEncryptor) ([]models.RemoteDestination, error) {
	rows, err := s.db.Query(`SELECT data_json FROM remote_destinations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.RemoteDestination
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var d models.RemoteDestination
		if err := json.Unmarshal([]byte(raw), &d); err != nil {
			return nil, err
		}
		d, err = decryptDestination(d, enc)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) replaceDestinationsTx(tx *sql.Tx, enc *secrets.FieldEncryptor) error {
	if _, err := tx.Exec(`DELETE FROM remote_destinations`); err != nil {
		return err
	}
	for _, d := range s.remoteDestinations {
		ed, err := encryptDestination(d, enc)
		if err != nil {
			return err
		}
		raw, err := json.Marshal(ed)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO remote_destinations (id, data_json) VALUES (?, ?)`, ed.ID, string(raw)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) loadSyncSettingsLocked(enc *secrets.FieldEncryptor) (*models.SyncSettings, error) {
	var raw string
	err := s.db.QueryRow(`SELECT data_json FROM sync_settings WHERE id = 1`).Scan(&raw)
	if err == sql.ErrNoRows || raw == "" || raw == "{}" {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var sync models.SyncSettings
	if err := json.Unmarshal([]byte(raw), &sync); err != nil {
		return nil, err
	}
	return decryptSync(&sync, enc)
}

func (s *Store) replaceSyncSettingsTx(tx *sql.Tx, enc *secrets.FieldEncryptor) error {
	if _, err := tx.Exec(`DELETE FROM sync_settings`); err != nil {
		return err
	}
	if s.sync == nil {
		return nil
	}
	es, err := encryptSync(s.sync, enc)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(es)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO sync_settings (id, data_json) VALUES (1, ?)`, string(raw))
	return err
}

func (s *Store) loadImportPrefsLocked() (map[string]string, error) {
	rows, err := s.db.Query(`SELECT source_profile_id, dest_profile_id FROM import_dest_prefs`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var src, dst string
		if err := rows.Scan(&src, &dst); err != nil {
			return nil, err
		}
		out[src] = dst
	}
	return out, rows.Err()
}

func (s *Store) replaceImportPrefsTx(tx *sql.Tx) error {
	if _, err := tx.Exec(`DELETE FROM import_dest_prefs`); err != nil {
		return err
	}
	for src, dst := range s.importDestByProfile {
		if _, err := tx.Exec(`INSERT INTO import_dest_prefs (source_profile_id, dest_profile_id) VALUES (?, ?)`, src, dst); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) loadTasksLocked() ([]models.Task, error) {
	rows, err := s.db.Query(`SELECT data_json FROM tasks`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Task
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var t models.Task
		if err := json.Unmarshal([]byte(raw), &t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) replaceTasksTx(tx *sql.Tx) error {
	if _, err := tx.Exec(`DELETE FROM tasks`); err != nil {
		return err
	}
	for _, t := range s.tasks {
		raw, err := json.Marshal(t)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO tasks (id, data_json) VALUES (?, ?)`, t.ID, string(raw)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) loadTaskRunsLocked() ([]models.TaskRunRecord, error) {
	rows, err := s.db.Query(`SELECT data_json FROM task_runs ORDER BY started_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.TaskRunRecord
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var r models.TaskRunRecord
		if err := json.Unmarshal([]byte(raw), &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) replaceTaskRunsTx(tx *sql.Tx) error {
	if _, err := tx.Exec(`DELETE FROM task_runs`); err != nil {
		return err
	}
	for _, r := range s.taskRuns {
		raw, err := json.Marshal(r)
		if err != nil {
			return err
		}
		ts := r.StartedAt.UTC().Format(time.RFC3339Nano)
		if _, err := tx.Exec(`INSERT INTO task_runs (id, task_id, started_at, data_json) VALUES (?, ?, ?, ?)`, r.ID, r.TaskID, ts, string(raw)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) loadNotifyChannelsLocked(enc *secrets.FieldEncryptor) ([]models.NotifyChannel, error) {
	rows, err := s.db.Query(`SELECT data_json FROM notify_channels`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.NotifyChannel
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var ch models.NotifyChannel
		if err := json.Unmarshal([]byte(raw), &ch); err != nil {
			return nil, err
		}
		cfg, err := decryptNotifyConfig(ch.Config, enc)
		if err != nil {
			return nil, err
		}
		ch.Config = cfg
		out = append(out, ch)
	}
	return out, rows.Err()
}

func (s *Store) replaceNotifyChannelsTx(tx *sql.Tx, enc *secrets.FieldEncryptor) error {
	if _, err := tx.Exec(`DELETE FROM notify_channels`); err != nil {
		return err
	}
	for _, ch := range s.notifyChannels {
		cp := ch
		cfg, err := encryptNotifyConfig(ch.Config, enc)
		if err != nil {
			return err
		}
		cp.Config = cfg
		raw, err := json.Marshal(cp)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO notify_channels (id, data_json) VALUES (?, ?)`, cp.ID, string(raw)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) persistLocked() error {
	return s.persistAllLocked()
}
