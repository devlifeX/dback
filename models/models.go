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

// DBType defines the database type (MySQL/MariaDB)
type DBType string

const (
	DBTypeMySQL   DBType = "MySQL"
	DBTypeMariaDB DBType = "MariaDB"
)

// Profile represents a saved connection profile
type Profile struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Host         string   `json:"host"`
	Port         string   `json:"port"`
	SSHUser      string   `json:"ssh_user"`
	SSHPassword  string   `json:"ssh_password"` // For password auth
	AuthType     AuthType `json:"auth_type"`
	AuthKeyPath  string   `json:"auth_key_path"` // Path to private key
	DBHost       string   `json:"db_host"`
	DBPort       string   `json:"db_port"`
	DBUser       string   `json:"db_user"`
	DBPassword   string   `json:"db_password"` // In a real app, should be encrypted
	DBType       DBType   `json:"db_type"`
	IsDocker     bool     `json:"is_docker"`
	ContainerID  string   `json:"container_id"`
	TargetDBName string   `json:"target_db_name"`
}

// LogEntry represents a single activity log
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"` // e.g., "Export", "Import"
	Details   string    `json:"details"`
	Status    string    `json:"status"` // "Success", "Failed", "In Progress"
	Error     string    `json:"error,omitempty"`
}

// AppConfig holds application-wide configuration (like saved profiles)
type AppConfig struct {
	Profiles []Profile `json:"profiles"`
}
