package models

import (
	"fmt"
	"net/url"
	"strings"
)

// SquidProxy is an HTTP proxy endpoint used for multi-region URL checks.
type SquidProxy struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Country     string `json:"country"`
	CountryCode string `json:"country_code,omitempty"`
	Enabled     bool   `json:"enabled"`
}

// SquidSettings holds global URL-check settings (direct / primary host country).
type SquidSettings struct {
	PrimaryHostCountry     string `json:"primary_host_country"`
	PrimaryHostCountryCode string `json:"primary_host_country_code,omitempty"`
}

func ValidateSquidProxy(p SquidProxy) error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("proxy name is required")
	}
	if strings.TrimSpace(p.Country) == "" {
		return fmt.Errorf("country is required")
	}
	raw := strings.TrimSpace(p.URL)
	if raw == "" {
		return fmt.Errorf("proxy URL is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid proxy URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("proxy URL must use http or https")
	}
	if strings.TrimSpace(u.Host) == "" {
		return fmt.Errorf("proxy URL is missing host")
	}
	cc := strings.TrimSpace(p.CountryCode)
	if cc != "" && len(cc) != 2 {
		return fmt.Errorf("country_code must be a 2-letter ISO code")
	}
	return nil
}

func ValidateSquidSettings(s SquidSettings) error {
	if strings.TrimSpace(s.PrimaryHostCountry) == "" {
		return fmt.Errorf("primary host country is required")
	}
	cc := strings.TrimSpace(s.PrimaryHostCountryCode)
	if cc != "" && len(cc) != 2 {
		return fmt.Errorf("primary_host_country_code must be a 2-letter ISO code")
	}
	return nil
}

func (p SquidProxy) Clone() SquidProxy {
	return p
}
