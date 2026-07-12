package models

import (
	"fmt"
	"net/url"
	"strings"
)

const MaxSecondaryURLs = 5

// URLTarget is a monitored HTTP(S) endpoint.
type URLTarget struct {
	URL string `json:"url"`
}

// URLCheck holds primary and secondary URLs monitored for a host.
type URLCheck struct {
	Primary          URLTarget `json:"primary,omitempty"`
	Secondary        []URLTarget `json:"secondary,omitempty"`
	ProxyIDs         []string  `json:"proxy_ids,omitempty"` // selected Squid proxy IDs; direct check always runs
}

// BackupPolicy configures per-host backup retention limits (0 = unlimited).
type BackupPolicy struct {
	Enabled         bool `json:"enabled,omitempty"`
	DBLocalKeep     int  `json:"db_local_keep,omitempty"`
	DBRemoteKeep    int  `json:"db_remote_keep,omitempty"`
	FilesLocalKeep  int  `json:"files_local_keep,omitempty"`
	FilesRemoteKeep int  `json:"files_remote_keep,omitempty"`
}

// URLCheckSample is one url_checker measurement.
type URLCheckSample struct {
	ID          string `json:"id"`
	ProfileID   string `json:"profile_id"`
	URL         string `json:"url"`
	TS          string `json:"ts"`
	TTFBMs      int64  `json:"ttfb_ms"`
	StatusCode  int    `json:"status_code"`
	OK          bool   `json:"ok"`
	ViaProxy    bool   `json:"via_proxy"`
	ProxyID     string `json:"proxy_id,omitempty"`
	SourceLabel string `json:"source_label"`
	CountryCode string `json:"country_code,omitempty"`
	Error       string `json:"error,omitempty"`
}

// URLCheckHourlyBucket aggregates samples per hour for charting.
type URLCheckHourlyBucket struct {
	Hour        string  `json:"hour"`
	URL         string  `json:"url"`
	SourceLabel string  `json:"source_label"`
	ProxyID     string  `json:"proxy_id,omitempty"`
	CountryCode string  `json:"country_code,omitempty"`
	AvgTTFBMs   float64 `json:"avg_ttfb_ms"`
	MinTTFBMs   int64   `json:"min_ttfb_ms"`
	MaxTTFBMs   int64   `json:"max_ttfb_ms"`
	Samples     int     `json:"samples"`
	OKCount     int     `json:"ok_count"`
	FailCount   int     `json:"fail_count"`
	LastStatus  int     `json:"last_status_code"`
}

func ValidateURLTarget(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URL %q: %w", raw, err)
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
	default:
		return fmt.Errorf("URL %q must use http or https scheme", raw)
	}
	if strings.TrimSpace(u.Host) == "" {
		return fmt.Errorf("URL %q is missing host", raw)
	}
	return nil
}

// ValidateURLCheck validates URL check settings on a profile.
func ValidateURLCheck(cfg URLCheck) error {
	if err := ValidateURLTarget(cfg.Primary.URL); err != nil {
		return fmt.Errorf("primary URL: %w", err)
	}
	if len(cfg.Secondary) > MaxSecondaryURLs {
		return fmt.Errorf("at most %d secondary URLs allowed", MaxSecondaryURLs)
	}
	seen := map[string]struct{}{}
	if u := strings.TrimSpace(cfg.Primary.URL); u != "" {
		seen[strings.ToLower(u)] = struct{}{}
	}
	for i, t := range cfg.Secondary {
		if err := ValidateURLTarget(t.URL); err != nil {
			return fmt.Errorf("secondary URL %d: %w", i+1, err)
		}
		u := strings.TrimSpace(t.URL)
		if u == "" {
			continue
		}
		key := strings.ToLower(u)
		if _, dup := seen[key]; dup {
			return fmt.Errorf("duplicate URL %q", u)
		}
		seen[key] = struct{}{}
	}
	return nil
}

// ValidateBackupPolicy validates retention policy fields.
func ValidateBackupPolicy(p BackupPolicy) error {
	for _, n := range []struct {
		name string
		v    int
	}{
		{"db_local_keep", p.DBLocalKeep},
		{"db_remote_keep", p.DBRemoteKeep},
		{"files_local_keep", p.FilesLocalKeep},
		{"files_remote_keep", p.FilesRemoteKeep},
	} {
		if n.v < 0 {
			return fmt.Errorf("%s must be >= 0", n.name)
		}
	}
	return nil
}
