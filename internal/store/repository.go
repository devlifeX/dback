package store

import (
	"dback/internal/config"
	"dback/internal/store/sqlstore"
	"dback/internal/storemodel"
	"dback/models"
)

// Re-export shared symbols for backward compatibility.
const CurrentVersion = storemodel.CurrentVersion

var (
	ErrVaultLocked                = storemodel.ErrVaultLocked
	ErrVaultExists                = storemodel.ErrVaultExists
	ErrVaultNotFound              = storemodel.ErrVaultNotFound
	ErrWrongMasterKey             = storemodel.ErrWrongMasterKey
	ErrMasterKeyRequired          = storemodel.ErrMasterKeyRequired
	ErrIncludeSecretsNoPassphrase = storemodel.ErrIncludeSecretsNoPassphrase
	ErrLegacyPlaintextWithVault   = storemodel.ErrLegacyPlaintextWithVault
	ErrSyncNotConfigured          = storemodel.ErrSyncNotConfigured
	ErrRemoteDestinationNotFound  = storemodel.ErrRemoteDestinationNotFound
	ErrRemoteDestinationInUse     = storemodel.ErrRemoteDestinationInUse
	ErrAppSettingsDestRequired    = storemodel.ErrAppSettingsDestRequired
	ErrTaskNotFound               = storemodel.ErrTaskNotFound
	ErrNotifyChannelNotFound      = storemodel.ErrNotifyChannelNotFound
)

type DestinationUsage = storemodel.DestinationUsage
type AppImportData = storemodel.AppImportData
type TemplateConflict = storemodel.TemplateConflict
type ProfileConflict = storemodel.ProfileConflict

var (
	FlattenProfiles           = storemodel.FlattenProfiles
	FlattenProfile            = storemodel.FlattenProfile
	MigrateRemoteDestinations = storemodel.MigrateRemoteDestinations
	DetectTemplateConflicts   = storemodel.DetectTemplateConflicts
	MergeTemplates            = storemodel.MergeTemplates
	MergeHistory              = storemodel.MergeHistory
	MergeLogs                 = storemodel.MergeLogs
	DetectProfileConflicts    = storemodel.DetectProfileConflicts
	MergeProfiles             = storemodel.MergeProfiles
)

// Repository is the application persistence layer (SQLite or MySQL).
type Repository interface {
	HasVault() bool
	HasLegacyPlaintext() bool
	IsUnlocked() bool
	Revision() uint64

	CreateVault(passphrase string) error
	Unlock(passphrase string) error
	Lock()
	ValidateMasterPassphrase(passphrase string) error

	LoadProfiles() ([]models.Profile, error)
	SaveProfiles(profiles []models.Profile) error
	LoadTemplates() ([]models.SQLTemplate, error)
	SaveTemplates(templates []models.SQLTemplate) error
	LoadHistory() ([]models.ExportRecord, error)
	SaveHistory(records []models.ExportRecord) error
	LoadLogs() ([]models.LogEntry, error)
	SaveLogs(entries []models.LogEntry) error

	LoadSyncSettings() (*models.SyncSettings, error)
	SaveSyncSettings(settings models.SyncSettings) error
	LoadSyncActivity() (models.SyncActivity, error)
	RecordSyncPush() error
	RecordSyncPull() error

	ImportDestForProfile(sourceProfileID string) string
	SetImportDestForProfile(sourceProfileID, destProfileID string) error
	HostSort() string
	SetHostSort(sort string) error

	LoadRemoteDestinations() ([]models.RemoteDestination, error)
	RemoteDestinationByID(id string) (models.RemoteDestination, error)
	AppSettingsDestinationID() (string, error)
	SetAppSettingsDestinationID(id string) error
	SaveRemoteDestination(dest models.RemoteDestination) error
	DestinationUsage(id string) (DestinationUsage, error)
	DeleteRemoteDestination(id string, force bool) error

	ListTasks() ([]models.Task, error)
	GetTask(id string) (models.Task, error)
	SaveTask(task models.Task) error
	SetTaskEnabled(id string, enabled bool) error
	DeleteTask(id string) error
	ListTaskRuns(taskID string, limit int) ([]models.TaskRunRecord, error)
	AppendTaskRun(run models.TaskRunRecord) error
	UpdateTaskState(id string, fn func(*models.Task) error) error

	ListNotifyChannels() ([]models.NotifyChannel, error)
	GetNotifyChannel(id string) (models.NotifyChannel, error)
	SaveNotifyChannel(ch models.NotifyChannel) error
	DeleteNotifyChannel(id string) error

	ImportProfilesBundle(path string, includeSecrets bool, passphrase string) ([]models.Profile, error)
	ExportProfiles(path string, profiles []models.Profile, includeSecrets bool, passphrase string) error
	ImportAppDataBundle(path string, includeSecrets bool, passphrase string) (AppImportData, error)
	ImportAppDataBytes(raw []byte, includeSecrets bool, passphrase string) (AppImportData, error)
	ExportAppData(path string, data AppImportData, includeSecrets bool, passphrase string) error
	MarshalAppDataBundle(data AppImportData, includeSecrets bool, passphrase string) ([]byte, error)
	MarshalAppDataBundleForSync(data AppImportData) ([]byte, error)
	ImportAppDataBundleForSync(raw []byte) (AppImportData, error)
}

// Options configures the store backend.
type Options struct {
	BaseDir string
	DB      config.DBConfig
}

// Open returns the default SQL-backed repository.
func Open(opts Options) (Repository, error) {
	return sqlstore.OpenStore(opts.BaseDir, opts.DB)
}

// New opens the default SQL store for baseDir (backward compatible).
func New(baseDir string) Repository {
	repo, err := Open(Options{
		BaseDir: baseDir,
		DB:      config.LoadDBConfig(baseDir),
	})
	if err != nil {
		panic("store.New: " + err.Error())
	}
	return repo
}
