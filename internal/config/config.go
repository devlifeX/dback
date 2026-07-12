package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultListen          = "127.0.0.1:14127"
	defaultQueueCapacity   = 256
	defaultMaxConcurrent   = 4
	defaultShutdownTimeout = 30 * time.Second
	defaultRateLimitRPS    = 20.0
	defaultRateLimitBurst  = 40
	defaultAuditCap        = 500
)

type Config struct {
	DataDir         string
	DB              DBConfig
	Listen          string
	PassphraseFile  string
	Passphrase      string
	APITokenFile    string
	APIToken        string
	WebRoot         string
	QueueCapacity   int
	MaxConcurrent   int
	ShutdownTimeout time.Duration
	RateLimitRPS    float64
	RateLimitBurst  int
	MetricsEnabled  bool
	AuditCap        int
	SquidProxy      string
	DefaultAdminPhone    string
	DefaultAdminPassword string
	DefaultAdminName     string
}

func Load() (Config, error) {
	cfg := Config{
		Listen:          envOr("DBACK_LISTEN", defaultListen),
		PassphraseFile:  strings.TrimSpace(os.Getenv("DBACK_PASSPHRASE_FILE")),
		Passphrase:      strings.TrimSpace(os.Getenv("DBACK_PASSPHRASE")),
		APITokenFile:    strings.TrimSpace(os.Getenv("DBACK_API_TOKEN_FILE")),
		APIToken:        strings.TrimSpace(os.Getenv("DBACK_API_TOKEN")),
		WebRoot:         strings.TrimSpace(os.Getenv("DBACK_WEB_ROOT")),
		QueueCapacity:   envIntOr("DBACK_QUEUE_CAPACITY", defaultQueueCapacity),
		MaxConcurrent:   envIntOr("DBACK_MAX_CONCURRENT", defaultMaxConcurrent),
		ShutdownTimeout: envDurationOr("DBACK_SHUTDOWN_TIMEOUT", defaultShutdownTimeout),
		RateLimitRPS:    envFloatOr("DBACK_RATE_LIMIT_RPS", defaultRateLimitRPS),
		RateLimitBurst:  envIntOr("DBACK_RATE_LIMIT_BURST", defaultRateLimitBurst),
		MetricsEnabled:  envBoolOr("DBACK_METRICS", true),
		AuditCap:        envIntOr("DBACK_AUDIT_CAP", defaultAuditCap),
		SquidProxy:      strings.TrimSpace(os.Getenv("DBACK_SQUID_PROXY")),
		DefaultAdminPhone:    envOr("DBACK_DEFAULT_ADMIN_PHONE", "09359922324"),
		DefaultAdminPassword: envOr("DBACK_DEFAULT_ADMIN_PASSWORD", "09359922324"),
		DefaultAdminName:     envOr("DBACK_DEFAULT_ADMIN_NAME", "Admin"),
	}

	dataDir := strings.TrimSpace(os.Getenv("DBACK_DATA_DIR"))
	if dataDir == "" {
		if dir, err := os.UserConfigDir(); err == nil && dir != "" {
			dataDir = filepath.Join(dir, "dback")
		} else if home, err := os.UserHomeDir(); err == nil {
			dataDir = filepath.Join(home, ".config", "dback")
		} else {
			dataDir = "."
		}
	}
	cfg.DataDir = dataDir
	cfg.DB = LoadDBConfig(dataDir)

	if cfg.Passphrase == "" && cfg.PassphraseFile != "" {
		raw, err := os.ReadFile(cfg.PassphraseFile)
		if err != nil {
			return Config{}, fmt.Errorf("read DBACK_PASSPHRASE_FILE: %w", err)
		}
		cfg.Passphrase = strings.TrimSpace(string(raw))
	}
	if cfg.APIToken == "" && cfg.APITokenFile != "" {
		raw, err := os.ReadFile(cfg.APITokenFile)
		if err != nil {
			return Config{}, fmt.Errorf("read DBACK_API_TOKEN_FILE: %w", err)
		}
		cfg.APIToken = strings.TrimSpace(string(raw))
	}

	if cfg.QueueCapacity < 1 {
		return Config{}, fmt.Errorf("DBACK_QUEUE_CAPACITY must be >= 1")
	}
	if cfg.MaxConcurrent < 1 {
		return Config{}, fmt.Errorf("DBACK_MAX_CONCURRENT must be >= 1")
	}
	return cfg, nil
}

func (c Config) LockPath() string {
	return filepath.Join(c.DataDir, "dback.pid.lock")
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envIntOr(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envDurationOr(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func envFloatOr(key string, fallback float64) float64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}

func envBoolOr(key string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
