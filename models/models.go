package models

import (
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
	ConnectionTypeSSH       ConnectionType = "SSH"
	ConnectionTypeJumpHost  ConnectionType = "JumpHost"
	ConnectionTypeWordPress ConnectionType = "WordPress"
)

// DBType defines the database type (MySQL/MariaDB)
type DBType string

const (
	DBTypeMySQL      DBType = "MySQL"
	DBTypeMariaDB    DBType = "MariaDB"
	DBTypePostgreSQL DBType = "PostgreSQL"
	DBTypeCouchDB    DBType = "CouchDB"
)

// Profile represents a saved connection profile
type Profile struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Group          string         `json:"group,omitempty"`
	Host           string         `json:"host"`
	Port           string         `json:"port"`
	ConnectionType ConnectionType `json:"connection_type"` // SSH or WordPress

	// SSH Fields
	SSHUser     string   `json:"ssh_user"`
	SSHPassword string   `json:"ssh_password"`
	AuthType    AuthType `json:"auth_type"`
	AuthKeyPath string   `json:"auth_key_path"`

	// Jump Host Fields (optional SSH bastion)
	JumpHost        string   `json:"jump_host,omitempty"`
	JumpPort        string   `json:"jump_port,omitempty"`
	JumpUser        string   `json:"jump_user,omitempty"`
	JumpPassword    string   `json:"jump_password,omitempty"`
	JumpAuthType    AuthType `json:"jump_auth_type,omitempty"`
	JumpAuthKeyPath string   `json:"jump_auth_key_path,omitempty"`

	// WordPress Fields
	WPUrl      string `json:"wp_url"`      // e.g. https://example.com
	WPKey      string `json:"wp_key"`      // The API key shared with plugin
	PluginPath string `json:"plugin_path"` // Path to save generated plugin

	DBHost       string `json:"db_host"`
	DBPort       string `json:"db_port"`
	DBUser       string `json:"db_user"`
	DBPassword   string `json:"db_password"` // In a real app, should be encrypted
	DBType       DBType `json:"db_type"`
	IsDocker     bool   `json:"is_docker"`
	ContainerID  string `json:"container_id"`
	TargetDBName string `json:"target_db_name"`
	Destination  string `json:"destination"` // Local folder path

	ExportSettings *TransferSettings `json:"export_settings,omitempty"`
	ImportSettings *TransferSettings `json:"import_settings,omitempty"`
}

// LogEntry represents a single activity log
type LogEntry struct {
	ID          string    `json:"id"`
	OperationID string    `json:"operation_id,omitempty"`
	ProfileID   string    `json:"profile_id"`
	ProfileName string    `json:"profile_name"`
	Timestamp   time.Time `json:"timestamp"`
	Action      string    `json:"action"` // e.g., "Export", "Import"
	Level       string    `json:"level,omitempty"`
	Details     string    `json:"details"`
	FilePath    string    `json:"file_path"`
	FileSize    string    `json:"file_size"`
	Status      string    `json:"status"`
	Error       string    `json:"error,omitempty"`
}

// AppConfig holds application-wide configuration (like saved profiles)
type AppConfig struct {
	Version  int       `json:"version,omitempty"`
	Profiles []Profile `json:"profiles"`
}

// ExportRecord represents a successful export entry for the history
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

// TransferSettings keeps export and import settings independent for each profile.
type TransferSettings struct {
	ConnectionType  ConnectionType `json:"connection_type"`
	Host            string         `json:"host"`
	Port            string         `json:"port"`
	SSHUser         string         `json:"ssh_user"`
	SSHPassword     string         `json:"ssh_password,omitempty"`
	AuthType        AuthType       `json:"auth_type"`
	AuthKeyPath     string         `json:"auth_key_path,omitempty"`
	JumpHost        string         `json:"jump_host,omitempty"`
	JumpPort        string         `json:"jump_port,omitempty"`
	JumpUser        string         `json:"jump_user,omitempty"`
	JumpPassword    string         `json:"jump_password,omitempty"`
	JumpAuthType    AuthType       `json:"jump_auth_type,omitempty"`
	JumpAuthKeyPath string         `json:"jump_auth_key_path,omitempty"`
	WPUrl           string         `json:"wp_url,omitempty"`
	WPKey           string         `json:"wp_key,omitempty"`
	DBHost          string         `json:"db_host"`
	DBPort          string         `json:"db_port"`
	DBUser          string         `json:"db_user"`
	DBPassword      string         `json:"db_password,omitempty"`
	DBType          DBType         `json:"db_type"`
	IsDocker        bool           `json:"is_docker"`
	ContainerID     string         `json:"container_id,omitempty"`
	TargetDBName    string         `json:"target_db_name"`
	Destination     string         `json:"destination,omitempty"`
}

type ProfileBundle struct {
	Version  int       `json:"version"`
	Profiles []Profile `json:"profiles"`
}

type BackupHistory struct {
	Version int            `json:"version"`
	Records []ExportRecord `json:"records"`
}

type ActivityLog struct {
	Version int        `json:"version"`
	Entries []LogEntry `json:"entries"`
}

func (p Profile) EffectiveExport() Profile {
	return p.withSettings(p.ExportSettings)
}

func (p Profile) EffectiveImport() Profile {
	if p.ImportSettings != nil {
		return p.withSettings(p.ImportSettings)
	}
	return p.withSettings(p.ExportSettings)
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
	p.JumpHost = s.JumpHost
	p.JumpPort = s.JumpPort
	p.JumpUser = s.JumpUser
	p.JumpPassword = s.JumpPassword
	p.JumpAuthType = s.JumpAuthType
	p.JumpAuthKeyPath = s.JumpAuthKeyPath
	p.WPUrl = s.WPUrl
	p.WPKey = s.WPKey
	p.DBHost = s.DBHost
	p.DBPort = s.DBPort
	p.DBUser = s.DBUser
	p.DBPassword = s.DBPassword
	p.DBType = s.DBType
	p.IsDocker = s.IsDocker
	p.ContainerID = s.ContainerID
	p.TargetDBName = s.TargetDBName
	p.Destination = s.Destination
	return p
}

func SettingsFromProfile(p Profile) TransferSettings {
	return TransferSettings{
		ConnectionType:  p.ConnectionType,
		Host:            p.Host,
		Port:            p.Port,
		SSHUser:         p.SSHUser,
		SSHPassword:     p.SSHPassword,
		AuthType:        p.AuthType,
		AuthKeyPath:     p.AuthKeyPath,
		JumpHost:        p.JumpHost,
		JumpPort:        p.JumpPort,
		JumpUser:        p.JumpUser,
		JumpPassword:    p.JumpPassword,
		JumpAuthType:    p.JumpAuthType,
		JumpAuthKeyPath: p.JumpAuthKeyPath,
		WPUrl:           p.WPUrl,
		WPKey:           p.WPKey,
		DBHost:          p.DBHost,
		DBPort:          p.DBPort,
		DBUser:          p.DBUser,
		DBPassword:      p.DBPassword,
		DBType:          p.DBType,
		IsDocker:        p.IsDocker,
		ContainerID:     p.ContainerID,
		TargetDBName:    p.TargetDBName,
		Destination:     p.Destination,
	}
}
