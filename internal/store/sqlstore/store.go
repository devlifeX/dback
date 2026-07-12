package sqlstore

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"dback/internal/config"
	"dback/internal/secrets"
	"dback/internal/store/vaultimport"
	"dback/internal/storemodel"
	"dback/models"
)

const minMasterKeyLen = 4

type Store struct {
	baseDir string
	driver  string
	dsn     string
	db      *sql.DB
	mu      sync.Mutex

	unlocked  bool
	dataKey   []byte
	masterKey []byte
	vaultSalt string
	revision  uint64

	profiles                   []models.Profile
	templates                  []models.SQLTemplate
	history                    []models.ExportRecord
	logs                       []models.LogEntry
	sync                       *models.SyncSettings
	syncActivity               models.SyncActivity
	importDestByProfile        map[string]string
	hostSort                   string
	remoteDestinations         []models.RemoteDestination
	appSettingsDestinationID   string
	remoteDestinationsMigrated bool
	tasks                      []models.Task
	taskRuns                   []models.TaskRunRecord
	notifyChannels             []models.NotifyChannel
	users                      []models.User
	authSettings               models.AuthSettings
	squidProxies               []models.SquidProxy
	squidSettings              models.SquidSettings
}

func Open(baseDir string, dbCfg config.DBConfig) (*Store, error) {
	driver := dbCfg.DriverOrDefault()
	sqlDriver := "sqlite3"
	if driver == config.DBDriverMySQL {
		sqlDriver = "mysql"
	}
	dsn, err := dbCfg.ResolveDSN()
	if err != nil {
		return nil, err
	}
	if baseDir == "" {
		baseDir = "."
	}
	return &Store{baseDir: baseDir, driver: sqlDriver, dsn: dsn}, nil
}

func (s *Store) ensureDB() error {
	if s.db != nil {
		return nil
	}
	db, err := sql.Open(s.driver, s.dsn)
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		return err
	}
	if err := runMigrations(db, s.driver); err != nil {
		return err
	}
	s.db = db
	return nil
}

func (s *Store) HasVault() bool {
	if vaultimport.HasVault(s.baseDir) || vaultimport.HasLegacyPlaintext(s.baseDir) {
		return true
	}
	if err := s.ensureDB(); err != nil {
		return false
	}
	return tableExists(s.db, s.driver, "vault_meta") && s.hasVaultMeta()
}

func (s *Store) hasVaultMeta() bool {
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM vault_meta WHERE id = 1`).Scan(&count)
	return count > 0
}

func (s *Store) HasLegacyPlaintext() bool {
	return vaultimport.HasLegacyPlaintext(s.baseDir)
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

func (s *Store) CreateVault(passphrase string) error {
	if err := validateMasterKey(passphrase); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.unlocked {
		return nil
	}
	if err := s.ensureDB(); err != nil {
		return err
	}
	if s.hasVaultMeta() {
		return storemodel.ErrVaultExists
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	s.vaultSalt = base64.StdEncoding.EncodeToString(salt)
	s.dataKey = secrets.DeriveKey(passphrase, salt)
	if _, err := s.db.Exec(`INSERT INTO vault_meta (id, salt, schema_version) VALUES (1, ?, ?)`, s.vaultSalt, storemodel.CurrentVersion); err != nil {
		return err
	}
	if _, err := s.db.Exec(`INSERT INTO app_settings (id, revision, sync_activity_json) VALUES (1, 0, '{}')`); err != nil {
		return err
	}
	s.templates = seedTemplates()
	s.importDestByProfile = map[string]string{}
	s.unlocked = true
	s.setMasterKeyLocked(passphrase)
	if err := s.persistAllLocked(); err != nil {
		return err
	}
	s.bumpRevisionLocked()
	if err := s.writeKeyCheckLocked(); err != nil {
		return err
	}
	log.Printf("sqlstore.CreateVault: initialized database at %q", s.dsn)
	return nil
}

func (s *Store) Unlock(passphrase string) error {
	if passphrase == "" {
		return storemodel.ErrMasterKeyRequired
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.unlocked {
		return nil
	}
	if err := s.ensureDB(); err != nil {
		return err
	}

	if !s.hasVaultMeta() {
		if vaultimport.HasVault(s.baseDir) {
			salt, err := vaultimport.VaultSalt(s.baseDir)
			if err != nil {
				return err
			}
			payload, err := vaultimport.ReadVault(s.baseDir, passphrase)
			if err != nil {
				if err.Error() == "wrong master key" {
					return storemodel.ErrWrongMasterKey
				}
				return err
			}
			if err := s.bootstrapFromVaultImportLocked(passphrase, salt, payload); err != nil {
				return err
			}
			if err := vaultimport.ArchiveVault(s.baseDir); err != nil {
				return err
			}
			log.Printf("sqlstore.Unlock: migrated vault file into database")
			return nil
		}
		if vaultimport.HasLegacyPlaintext(s.baseDir) {
			payload, err := vaultimport.LoadLegacyPayload(s.baseDir)
			if err != nil {
				return err
			}
			if err := s.bootstrapNewLocked(passphrase); err != nil {
				return err
			}
			s.applyPayloadLocked(payload)
			if err := s.persistAllLocked(); err != nil {
				return err
			}
			if err := s.writeKeyCheckLocked(); err != nil {
				return err
			}
			_ = vaultimport.RemoveLegacyPlaintext(s.baseDir)
			s.bumpRevisionLocked()
			log.Printf("sqlstore.Unlock: migrated legacy plaintext into database")
			return nil
		}
		return storemodel.ErrVaultNotFound
	}

	if err := s.loadMetaAndKeyLocked(passphrase); err != nil {
		return err
	}
	if err := s.verifyKeyLocked(); err != nil {
		s.dataKey = nil
		return err
	}
	if err := s.loadAllLocked(); err != nil {
		return err
	}
	if vaultimport.HasLegacyPlaintext(s.baseDir) {
		_ = vaultimport.RemoveLegacyPlaintext(s.baseDir)
	}
	s.setMasterKeyLocked(passphrase)
	s.unlocked = true
	log.Printf("sqlstore.Unlock: database unlocked (profiles=%d)", len(s.profiles))
	return nil
}

func (s *Store) bootstrapNewLocked(passphrase string) error {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	s.vaultSalt = base64.StdEncoding.EncodeToString(salt)
	s.dataKey = secrets.DeriveKey(passphrase, salt)
	if _, err := s.db.Exec(`INSERT INTO vault_meta (id, salt, schema_version) VALUES (1, ?, ?)`, s.vaultSalt, storemodel.CurrentVersion); err != nil {
		return err
	}
	if _, err := s.db.Exec(`INSERT INTO app_settings (id, revision, sync_activity_json) VALUES (1, 0, '{}')`); err != nil {
		return err
	}
	s.importDestByProfile = map[string]string{}
	s.setMasterKeyLocked(passphrase)
	s.unlocked = true
	return nil
}

func (s *Store) bootstrapFromVaultImportLocked(passphrase, vaultSalt string, payload models.AppVaultPayload) error {
	salt, err := base64.StdEncoding.DecodeString(vaultSalt)
	if err != nil {
		return err
	}
	s.vaultSalt = vaultSalt
	s.dataKey = secrets.DeriveKey(passphrase, salt)
	if _, err := s.db.Exec(`INSERT INTO vault_meta (id, salt, schema_version) VALUES (1, ?, ?)`, s.vaultSalt, storemodel.CurrentVersion); err != nil {
		return err
	}
	if _, err := s.db.Exec(`INSERT INTO app_settings (id, revision, sync_activity_json) VALUES (1, 0, '{}')`); err != nil {
		return err
	}
	s.applyPayloadLocked(payload)
	s.setMasterKeyLocked(passphrase)
	s.unlocked = true
	if err := s.persistAllLocked(); err != nil {
		return err
	}
	if err := s.writeKeyCheckLocked(); err != nil {
		return err
	}
	s.bumpRevisionLocked()
	return nil
}

const vaultKeyCheckPlain = "dback-vault-check"

func (s *Store) writeKeyCheckLocked() error {
	check, err := s.fieldEnc().Encrypt(vaultKeyCheckPlain)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE vault_meta SET key_check = ? WHERE id = 1`, check)
	return err
}

func (s *Store) verifyKeyLocked() error {
	var check string
	if err := s.db.QueryRow(`SELECT key_check FROM vault_meta WHERE id = 1`).Scan(&check); err != nil {
		return err
	}
	if check == "" {
		return nil
	}
	plain, err := s.fieldEnc().Decrypt(check)
	if err != nil || plain != vaultKeyCheckPlain {
		return storemodel.ErrWrongMasterKey
	}
	return nil
}

func (s *Store) loadMetaAndKeyLocked(passphrase string) error {
	if err := s.db.QueryRow(`SELECT salt FROM vault_meta WHERE id = 1`).Scan(&s.vaultSalt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storemodel.ErrVaultNotFound
		}
		return err
	}
	salt, err := base64.StdEncoding.DecodeString(s.vaultSalt)
	if err != nil {
		return err
	}
	s.dataKey = secrets.DeriveKey(passphrase, salt)
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
	s.remoteDestinations = nil
	s.appSettingsDestinationID = ""
	s.remoteDestinationsMigrated = false
	s.tasks = nil
	s.taskRuns = nil
	s.notifyChannels = nil
	s.users = nil
	s.authSettings = models.DefaultAuthSettings()
	s.importDestByProfile = nil
}

func (s *Store) ValidateMasterPassphrase(passphrase string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.unlocked {
		return storemodel.ErrVaultLocked
	}
	cached, err := s.masterPassphraseLocked()
	if err != nil {
		return err
	}
	if passphrase != cached {
		return storemodel.ErrWrongMasterKey
	}
	return nil
}

func (s *Store) bumpRevisionLocked() {
	s.revision++
	if s.db != nil {
		_, _ = s.db.Exec(`UPDATE app_settings SET revision = ? WHERE id = 1`, s.revision)
	}
}

func (s *Store) requireUnlocked() error {
	if !s.unlocked {
		return storemodel.ErrVaultLocked
	}
	return nil
}

func (s *Store) fieldEnc() *secrets.FieldEncryptor {
	return secrets.NewFieldEncryptor(s.dataKey)
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
		return "", storemodel.ErrVaultLocked
	}
	return string(s.masterKey), nil
}

func validateMasterKey(passphrase string) error {
	if passphrase == "" {
		return storemodel.ErrMasterKeyRequired
	}
	if len(passphrase) < minMasterKeyLen {
		return fmt.Errorf("master key must be at least %d characters", minMasterKeyLen)
	}
	return nil
}

func seedTemplates() []models.SQLTemplate {
	now := time.Now()
	return []models.SQLTemplate{
		{ID: "seed-recreate-db", Name: "Recreate database", Description: "Drop and recreate target database", Body: "DROP DATABASE IF EXISTS {databasename};\nCREATE DATABASE {databasename};", CreatedAt: now, UpdatedAt: now},
		{ID: "seed-create-admin", Name: "Create admin user", Description: "Create admin user devlife", Body: storeSQLTemplateCreateAdmin, CreatedAt: now, UpdatedAt: now},
	}
}

const storeSQLTemplateCreateAdmin = `INSERT INTO wp_users
(user_login, user_pass, user_nicename, user_email, user_registered, user_status, display_name)
VALUES ('devlife', MD5('devlife'), 'devlife', 'devlife@example.com', NOW(), 0, 'devlife');

DELETE FROM wp_usermeta WHERE user_id IN (SELECT ID FROM (SELECT ID FROM wp_users WHERE user_login = 'devlife') t);

INSERT INTO wp_usermeta (user_id, meta_key, meta_value)
SELECT ID, 'wp_capabilities', 'a:1:{s:13:"administrator";b:1;}' FROM wp_users WHERE user_login = 'devlife';

INSERT INTO wp_usermeta (user_id, meta_key, meta_value)
SELECT ID, 'wp_user_level', '10' FROM wp_users WHERE user_login = 'devlife';`

func readJSONFile(path string, value any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(value)
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

func (s *Store) applyPayloadLocked(payload models.AppVaultPayload) {
	storemodel.MigrateRemoteDestinations(&payload)
	s.profiles = storemodel.FlattenProfiles(payload.Profiles)
	s.templates = append([]models.SQLTemplate(nil), payload.Templates...)
	if len(s.templates) == 0 {
		s.templates = seedTemplates()
	}
	s.history = append([]models.ExportRecord(nil), payload.History...)
	s.logs = append([]models.LogEntry(nil), payload.Logs...)
	s.sync = payload.Sync.Clone()
	s.syncActivity = payload.SyncActivity
	s.remoteDestinations = cloneRemoteDestinations(payload.RemoteDestinations)
	s.appSettingsDestinationID = payload.AppSettingsDestinationID
	s.remoteDestinationsMigrated = payload.RemoteDestinationsMigrated
	s.syncLegacyFromAppSettingsLocked()
	if len(payload.ImportDestByProfile) > 0 {
		s.importDestByProfile = cloneStringMap(payload.ImportDestByProfile)
	} else {
		s.importDestByProfile = map[string]string{}
	}
	s.hostSort = payload.HostSort
	s.tasks = append([]models.Task(nil), payload.Tasks...)
	s.taskRuns = append([]models.TaskRunRecord(nil), payload.TaskRuns...)
	s.notifyChannels = append([]models.NotifyChannel(nil), payload.NotifyChannels...)
	s.users = append([]models.User(nil), payload.Users...)
	if payload.AuthSettings != nil {
		s.authSettings = *payload.AuthSettings
	} else {
		s.authSettings = models.DefaultAuthSettings()
	}
}
