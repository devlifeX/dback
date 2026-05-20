package models

import (
	"strings"
	"time"
)

// AuthType defines the SSH authentication method
type AuthType string

const (
	AuthTypePassword AuthType = "Password"
	AuthTypeKeyFile  AuthType = "Key File"
)

// ConnectionType defines how we access the server
type ConnectionType string

const (
	ConnectionTypeSSH      ConnectionType = "SSH"
	ConnectionTypeJumpHost ConnectionType = "JumpHost"
)

// legacyConnectionWordPress is filtered out on load; not supported in the UI.
const legacyConnectionWordPress ConnectionType = "WordPress"

// DBType defines the database type (MySQL/MariaDB only)
type DBType string

const (
	DBTypeMySQL   DBType = "MySQL"
	DBTypeMariaDB DBType = "MariaDB"
)

// Profile represents an independent Host with unified connection settings.
type Profile struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Group          string         `json:"group,omitempty"`
	Host           string         `json:"host"`
	Port           string         `json:"port"`
	ConnectionType ConnectionType `json:"connection_type"`

	SSHUser     string   `json:"ssh_user"`
	SSHPassword string   `json:"ssh_password"`
	AuthType    AuthType `json:"auth_type"`
	AuthKeyPath string   `json:"auth_key_path"`
	AuthKeyPEM  string   `json:"auth_key_pem,omitempty"`

	JumpHost        string   `json:"jump_host,omitempty"`
	JumpPort        string   `json:"jump_port,omitempty"`
	JumpUser        string   `json:"jump_user,omitempty"`
	JumpPassword    string   `json:"jump_password,omitempty"`
	JumpAuthType    AuthType `json:"jump_auth_type,omitempty"`
	JumpAuthKeyPath string   `json:"jump_auth_key_path,omitempty"`
	JumpAuthKeyPEM  string   `json:"jump_auth_key_pem,omitempty"`

	DBHost      string `json:"db_host"`
	DBPort      string `json:"db_port"`
	DBUser      string `json:"db_user"`
	DBPassword  string `json:"db_password"`
	DBType      DBType `json:"db_type"`
	IsDocker    bool   `json:"is_docker"`
	ContainerID string `json:"container_id"`

	TargetDBName         string `json:"target_db_name"`
	Destination          string `json:"destination"`
	PreImportQuery       string `json:"pre_import_query,omitempty"`
	RunQueryBeforeImport bool   `json:"run_query_before_import,omitempty"`
	PostImportQuery      string `json:"post_import_query,omitempty"`
	RunQueryAfterImport  bool   `json:"run_query_after_import,omitempty"`

	// Legacy fields — read-only for migration; not written on save.
	ExportSettings *TransferSettings `json:"export_settings,omitempty"`
	ImportSettings *TransferSettings `json:"import_settings,omitempty"`
}

// TransferSettings legacy nested settings (migration only).
type TransferSettings struct {
	ConnectionType       ConnectionType `json:"connection_type"`
	Host                 string         `json:"host"`
	Port                 string         `json:"port"`
	SSHUser              string         `json:"ssh_user"`
	SSHPassword          string         `json:"ssh_password,omitempty"`
	AuthType             AuthType       `json:"auth_type"`
	AuthKeyPath          string         `json:"auth_key_path,omitempty"`
	AuthKeyPEM           string         `json:"auth_key_pem,omitempty"`
	JumpHost             string         `json:"jump_host,omitempty"`
	JumpPort             string         `json:"jump_port,omitempty"`
	JumpUser             string         `json:"jump_user,omitempty"`
	JumpPassword         string         `json:"jump_password,omitempty"`
	JumpAuthType         AuthType       `json:"jump_auth_type,omitempty"`
	JumpAuthKeyPath      string         `json:"jump_auth_key_path,omitempty"`
	JumpAuthKeyPEM       string         `json:"jump_auth_key_pem,omitempty"`
	WPUrl                string         `json:"wp_url,omitempty"`
	WPKey                string         `json:"wp_key,omitempty"`
	DBHost               string         `json:"db_host"`
	DBPort               string         `json:"db_port"`
	DBUser               string         `json:"db_user"`
	DBPassword           string         `json:"db_password,omitempty"`
	DBType               DBType         `json:"db_type"`
	IsDocker             bool           `json:"is_docker"`
	ContainerID          string         `json:"container_id,omitempty"`
	TargetDBName         string         `json:"target_db_name"`
	Destination          string         `json:"destination,omitempty"`
	PreImportQuery       string         `json:"pre_import_query,omitempty"`
	RunQueryBeforeImport bool           `json:"run_query_before_import,omitempty"`
	PostImportQuery      string         `json:"post_import_query,omitempty"`
	RunQueryAfterImport  bool           `json:"run_query_after_import,omitempty"`

	legacyPostExportQuery     string `json:"post_export_query,omitempty"`
	legacyRunQueryAfterExport bool   `json:"run_query_after_export,omitempty"`
}

func (s *TransferSettings) MigrateQueryFields() {
	if s == nil {
		return
	}
	if s.PostImportQuery == "" && s.legacyPostExportQuery != "" {
		s.PostImportQuery = s.legacyPostExportQuery
	}
	if !s.RunQueryAfterImport && s.legacyRunQueryAfterExport {
		s.RunQueryAfterImport = true
	}
}

// SQLTemplate is a reusable SQL snippet with placeholder support.
type SQLTemplate struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TemplateBundle struct {
	Version   int           `json:"version"`
	Templates []SQLTemplate `json:"templates"`
}

// LogEntry represents a single activity log with structured trace fields.
type LogEntry struct {
	ID          string    `json:"id"`
	OperationID string    `json:"operation_id,omitempty"`
	ProfileID   string    `json:"profile_id"`
	ProfileName string    `json:"profile_name"`
	Timestamp   time.Time `json:"timestamp"`
	Action      string    `json:"action"`
	Phase       string    `json:"phase,omitempty"`
	Strategy    string    `json:"strategy,omitempty"`
	Attempt     int       `json:"attempt,omitempty"`
	Level       string    `json:"level,omitempty"`
	Details     string    `json:"details"`
	FilePath    string    `json:"file_path"`
	FileSize    string    `json:"file_size"`
	Status      string    `json:"status"`
	Error       string    `json:"error,omitempty"`
}

type AppConfig struct {
	Version  int       `json:"version,omitempty"`
	Profiles []Profile `json:"profiles"`
}

type ExportRecord struct {
	ID             string         `json:"id"`
	OperationID    string         `json:"operation_id,omitempty"`
	ProfileID      string         `json:"profile_id"`
	ProfileName    string         `json:"profile_name"`
	DatabaseName   string         `json:"database_name"`
	ExportDate     time.Time      `json:"export_date"`
	FilePath       string         `json:"file_path"`
	FileSize       string         `json:"file_size"`
	FileSizeBytes  int64          `json:"file_size_bytes"`
	ConnectionType ConnectionType `json:"connection_type,omitempty"`
}

type ProfileBundle struct {
	Version   int       `json:"version"`
	Encrypted bool      `json:"encrypted,omitempty"`
	Salt      string    `json:"salt,omitempty"`
	Nonce     string    `json:"nonce,omitempty"`
	Profiles  []Profile `json:"profiles,omitempty"`
	// EncryptedPayload holds AES-GCM ciphertext when Encrypted is true.
	EncryptedPayload string `json:"encrypted_payload,omitempty"`
}

// AppVaultFile is the on-disk encrypted application vault (single file storage).
type AppVaultFile struct {
	Version          int       `json:"version"`
	Salt             string    `json:"salt"`
	Nonce            string    `json:"nonce"`
	UpdatedAt        time.Time `json:"updated_at"`
	EncryptedPayload string    `json:"encrypted_payload"`
}

// AppVaultPayload is the decrypted contents of the internal vault.
type AppVaultPayload struct {
	Version   int            `json:"version"`
	Profiles  []Profile      `json:"profiles"`
	Templates []SQLTemplate  `json:"templates"`
	History   []ExportRecord `json:"history"`
	Logs      []LogEntry     `json:"logs"`
}

// AppBundle exports hosts, templates, backup history metadata, and activity logs.
// Backup .sql.gz files are not included.
type AppBundle struct {
	Version          int            `json:"version"`
	ExportedAt       time.Time      `json:"exported_at"`
	Encrypted        bool           `json:"encrypted,omitempty"`
	Salt             string         `json:"salt,omitempty"`
	Nonce            string         `json:"nonce,omitempty"`
	Profiles         []Profile      `json:"profiles,omitempty"`
	Templates        []SQLTemplate  `json:"templates,omitempty"`
	History          []ExportRecord `json:"history,omitempty"`
	Logs             []LogEntry     `json:"logs,omitempty"`
	EncryptedPayload string         `json:"encrypted_payload,omitempty"`
}

type BackupHistory struct {
	Version int            `json:"version"`
	Records []ExportRecord `json:"records"`
}

type ActivityLog struct {
	Version int        `json:"version"`
	Entries []LogEntry `json:"entries"`
}

func SettingsFromProfile(p Profile) TransferSettings {
	return TransferSettings{
		ConnectionType:       p.ConnectionType,
		Host:                 p.Host,
		Port:                 p.Port,
		SSHUser:              p.SSHUser,
		SSHPassword:          p.SSHPassword,
		AuthType:             p.AuthType,
		AuthKeyPath:          p.AuthKeyPath,
		AuthKeyPEM:           p.AuthKeyPEM,
		JumpHost:             p.JumpHost,
		JumpPort:             p.JumpPort,
		JumpUser:             p.JumpUser,
		JumpPassword:         p.JumpPassword,
		JumpAuthType:         p.JumpAuthType,
		JumpAuthKeyPath:      p.JumpAuthKeyPath,
		JumpAuthKeyPEM:       p.JumpAuthKeyPEM,
		DBHost:               p.DBHost,
		DBPort:               p.DBPort,
		DBUser:               p.DBUser,
		DBPassword:           p.DBPassword,
		DBType:               p.DBType,
		IsDocker:             p.IsDocker,
		ContainerID:          p.ContainerID,
		TargetDBName:         p.TargetDBName,
		Destination:          p.Destination,
		PreImportQuery:       p.PreImportQuery,
		RunQueryBeforeImport: p.RunQueryBeforeImport,
		PostImportQuery:      p.PostImportQuery,
		RunQueryAfterImport:  p.RunQueryAfterImport,
	}
}

func (p Profile) ApplySettings(s *TransferSettings) Profile {
	return p.withSettings(s)
}

func (p Profile) withSettings(s *TransferSettings) Profile {
	if s == nil {
		return p
	}
	p.ConnectionType = s.ConnectionType
	p.Host = s.Host
	p.Port = s.Port
	p.SSHUser = s.SSHUser
	p.SSHPassword = s.SSHPassword
	p.AuthType = s.AuthType
	p.AuthKeyPath = s.AuthKeyPath
	p.AuthKeyPEM = s.AuthKeyPEM
	p.JumpHost = s.JumpHost
	p.JumpPort = s.JumpPort
	p.JumpUser = s.JumpUser
	p.JumpPassword = s.JumpPassword
	p.JumpAuthType = s.JumpAuthType
	p.JumpAuthKeyPath = s.JumpAuthKeyPath
	p.JumpAuthKeyPEM = s.JumpAuthKeyPEM
	p.DBHost = s.DBHost
	p.DBPort = s.DBPort
	p.DBUser = s.DBUser
	p.DBPassword = s.DBPassword
	p.DBType = normalizeDBType(s.DBType)
	p.IsDocker = s.IsDocker
	p.ContainerID = s.ContainerID
	p.TargetDBName = s.TargetDBName
	p.Destination = s.Destination
	p.PreImportQuery = s.PreImportQuery
	p.RunQueryBeforeImport = s.RunQueryBeforeImport
	p.PostImportQuery = s.PostImportQuery
	p.RunQueryAfterImport = s.RunQueryAfterImport
	return p
}

func normalizeDBType(t DBType) DBType {
	switch t {
	case DBTypeMariaDB:
		return DBTypeMariaDB
	default:
		return DBTypeMySQL
	}
}

// QueryVars holds placeholder values for SQL template substitution.
type QueryVars struct {
	DatabaseName string
	Host         string
	Profile      string
	DBUser       string
}

func SubstituteQuery(query string, vars QueryVars) string {
	if query == "" {
		return query
	}
	repl := strings.NewReplacer(
		"{databasename}", vars.DatabaseName,
		"{host}", vars.Host,
		"{profile}", vars.Profile,
		"{dbuser}", vars.DBUser,
	)
	return repl.Replace(query)
}

func SubstituteQueryDBName(query, dbName string) string {
	return SubstituteQuery(query, QueryVars{DatabaseName: dbName})
}

func (p Profile) QueryVars() QueryVars {
	return QueryVars{
		DatabaseName: p.TargetDBName,
		Host:         p.Host,
		Profile:      p.Name,
		DBUser:       p.DBUser,
	}
}

func (p Profile) SupportsSQLQuery() bool {
	return mysqlOrMariaDB(p)
}

func mysqlOrMariaDB(p Profile) bool {
	return p.DBType == DBTypeMySQL || p.DBType == DBTypeMariaDB
}

func (s TransferSettings) SupportsSQLQuery() bool {
	return s.DBType == DBTypeMySQL || s.DBType == DBTypeMariaDB
}

func (p Profile) SupportsImportSQLQuery() bool {
	return p.SupportsSQLQuery()
}

func SettingsEqual(a, b *TransferSettings) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	aa := *a
	bb := *b
	aa.SSHPassword = ""
	aa.DBPassword = ""
	aa.AuthKeyPEM = ""
	aa.JumpPassword = ""
	aa.JumpAuthKeyPEM = ""
	bb.SSHPassword = ""
	bb.DBPassword = ""
	bb.AuthKeyPEM = ""
	bb.JumpPassword = ""
	bb.JumpAuthKeyPEM = ""
	return aa == bb
}
