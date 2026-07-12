package app

import (
	"errors"
	"fmt"

	"dback/internal/store"
	"dback/models"
)

var ErrSquidProxyNotFound = errors.New("squid proxy not found")

func (a *App) ListSquidProxies() ([]models.SquidProxy, error) {
	return a.store.ListSquidProxies()
}

func (a *App) GetSquidProxy(id string) (models.SquidProxy, error) {
	p, err := a.store.GetSquidProxy(id)
	if err != nil {
		if errors.Is(err, store.ErrSquidProxyNotFound) {
			return models.SquidProxy{}, ErrSquidProxyNotFound
		}
		return models.SquidProxy{}, err
	}
	return p, nil
}

func (a *App) SaveSquidProxy(p models.SquidProxy) error {
	return a.store.SaveSquidProxy(p)
}

func (a *App) DeleteSquidProxy(id string) error {
	if err := a.store.DeleteSquidProxy(id); err != nil {
		if errors.Is(err, store.ErrSquidProxyNotFound) {
			return ErrSquidProxyNotFound
		}
		return err
	}
	return nil
}

func (a *App) GetSquidSettings() (models.SquidSettings, error) {
	return a.store.GetSquidSettings()
}

func (a *App) SaveSquidSettings(s models.SquidSettings) error {
	return a.store.SaveSquidSettings(s)
}

func (a *App) validateProfileProxyIDs(proxyIDs []string) error {
	if len(proxyIDs) == 0 {
		return nil
	}
	proxies, err := a.store.ListSquidProxies()
	if err != nil {
		return err
	}
	known := map[string]struct{}{}
	for _, p := range proxies {
		if p.Enabled {
			known[p.ID] = struct{}{}
		}
	}
	for _, id := range proxyIDs {
		if id == "" {
			continue
		}
		if _, ok := known[id]; !ok {
			return fmt.Errorf("unknown or disabled squid proxy %q", id)
		}
	}
	return nil
}

func IsSquidProxyNotFound(err error) bool {
	return errors.Is(err, ErrSquidProxyNotFound) || errors.Is(err, store.ErrSquidProxyNotFound)
}
