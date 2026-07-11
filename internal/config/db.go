package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DBDriverSQLite = "sqlite"
	DBDriverMySQL  = "mysql"
)

type DBConfig struct {
	Driver   string
	DSN      string
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

func (c DBConfig) DriverOrDefault() string {
	switch strings.ToLower(strings.TrimSpace(c.Driver)) {
	case "", DBDriverSQLite:
		return DBDriverSQLite
	case DBDriverMySQL:
		return DBDriverMySQL
	default:
		return strings.ToLower(strings.TrimSpace(c.Driver))
	}
}

func LoadDBConfig(dataDir string) DBConfig {
	cfg := DBConfig{
		Driver:   strings.TrimSpace(os.Getenv("DBACK_DB_DRIVER")),
		DSN:      strings.TrimSpace(os.Getenv("DBACK_DB_DSN")),
		Host:     strings.TrimSpace(os.Getenv("DBACK_DB_HOST")),
		Port:     strings.TrimSpace(os.Getenv("DBACK_DB_PORT")),
		User:     strings.TrimSpace(os.Getenv("DBACK_DB_USER")),
		Password: os.Getenv("DBACK_DB_PASSWORD"),
		Name:     strings.TrimSpace(os.Getenv("DBACK_DB_NAME")),
	}
	if cfg.DSN == "" && cfg.DriverOrDefault() == DBDriverSQLite {
		dbPath := filepath.Join(dataDir, "dback.db")
		cfg.DSN = dbPath + "?_foreign_keys=on&_busy_timeout=5000"
	}
	return cfg
}

func (c DBConfig) ResolveDSN() (string, error) {
	if strings.TrimSpace(c.DSN) != "" {
		return c.DSN, nil
	}
	if c.DriverOrDefault() != DBDriverMySQL {
		return "", fmt.Errorf("database DSN is required")
	}
	host := c.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := c.Port
	if port == "" {
		port = "3306"
	}
	name := c.Name
	if name == "" {
		name = "dback"
	}
	user := c.User
	if user == "" {
		user = "dback"
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=true", user, c.Password, host, port, name), nil
}
