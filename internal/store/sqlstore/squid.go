package sqlstore

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"dback/internal/storemodel"
	"dback/models"
)

func (s *Store) ListSquidProxies() ([]models.SquidProxy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return nil, err
	}
	return append([]models.SquidProxy(nil), s.squidProxies...), nil
}

func (s *Store) GetSquidProxy(id string) (models.SquidProxy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return models.SquidProxy{}, err
	}
	for _, p := range s.squidProxies {
		if p.ID == id {
			return p, nil
		}
	}
	return models.SquidProxy{}, storemodel.ErrSquidProxyNotFound
}

func (s *Store) SaveSquidProxy(p models.SquidProxy) error {
	if err := models.ValidateSquidProxy(p); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	p.Name = strings.TrimSpace(p.Name)
	p.Country = strings.TrimSpace(p.Country)
	p.CountryCode = strings.ToUpper(strings.TrimSpace(p.CountryCode))
	if p.ID == "" {
		p.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	for i, existing := range s.squidProxies {
		if existing.ID == p.ID {
			s.squidProxies[i] = p.Clone()
			s.bumpRevisionLocked()
			return s.persistAllLocked()
		}
	}
	s.squidProxies = append(s.squidProxies, p.Clone())
	s.bumpRevisionLocked()
	return s.persistAllLocked()
}

func (s *Store) DeleteSquidProxy(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	for i, p := range s.squidProxies {
		if p.ID == id {
			s.squidProxies = append(s.squidProxies[:i], s.squidProxies[i+1:]...)
			s.bumpRevisionLocked()
			return s.persistAllLocked()
		}
	}
	return storemodel.ErrSquidProxyNotFound
}

func (s *Store) GetSquidSettings() (models.SquidSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return models.SquidSettings{}, err
	}
	return s.squidSettings, nil
}

func (s *Store) SaveSquidSettings(settings models.SquidSettings) error {
	if err := models.ValidateSquidSettings(settings); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	settings.PrimaryHostCountry = strings.TrimSpace(settings.PrimaryHostCountry)
	settings.PrimaryHostCountryCode = strings.ToUpper(strings.TrimSpace(settings.PrimaryHostCountryCode))
	s.squidSettings = settings
	s.bumpRevisionLocked()
	return s.persistAllLocked()
}

func (s *Store) loadSquidProxiesLocked() ([]models.SquidProxy, error) {
	if err := s.ensureDB(); err != nil {
		return nil, err
	}
	if !tableExists(s.db, s.driver, "squid_proxies") {
		return nil, nil
	}
	rows, err := s.db.Query(`SELECT data_json FROM squid_proxies`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.SquidProxy
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var p models.SquidProxy
		if err := json.Unmarshal([]byte(raw), &p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) loadSquidSettingsLocked() (models.SquidSettings, error) {
	settings := models.SquidSettings{PrimaryHostCountry: "Local server"}
	if err := s.ensureDB(); err != nil {
		return settings, err
	}
	var raw string
	err := s.db.QueryRow(`SELECT squid_settings_json FROM app_settings WHERE id = 1`).Scan(&raw)
	if err != nil {
		return settings, nil
	}
	if strings.TrimSpace(raw) == "" || raw == "{}" {
		return settings, nil
	}
	_ = json.Unmarshal([]byte(raw), &settings)
	if strings.TrimSpace(settings.PrimaryHostCountry) == "" {
		settings.PrimaryHostCountry = "Local server"
	}
	return settings, nil
}
