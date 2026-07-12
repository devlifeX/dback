package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"dback/internal/notify"
	"dback/internal/store"
	"dback/models"
)

var ErrInvalidNotifyChannel = errors.New("invalid notify channel")

func (a *App) ListNotifyChannels() ([]models.NotifyChannel, error) {
	channels, err := a.store.ListNotifyChannels()
	if err != nil {
		return nil, err
	}
	out := make([]models.NotifyChannel, len(channels))
	for i, ch := range channels {
		out[i] = notify.RedactChannelConfig(ch)
	}
	return out, nil
}

func (a *App) GetNotifyChannelFull(id string) (models.NotifyChannel, error) {
	return a.store.GetNotifyChannel(id)
}

func (a *App) GetNotifyChannel(id string) (models.NotifyChannel, error) {
	ch, err := a.store.GetNotifyChannel(id)
	if err != nil {
		return models.NotifyChannel{}, err
	}
	return notify.RedactChannelConfig(ch), nil
}

func (a *App) ListNotifyChannelsFull() ([]models.NotifyChannel, error) {
	return a.store.ListNotifyChannels()
}

func (a *App) SaveNotifyChannel(ch models.NotifyChannel) error {
	if ch.ID != "" {
		existing, err := a.store.GetNotifyChannel(ch.ID)
		if err == nil {
			ch.Config = mergeNotifyConfig(existing.Config, ch.Config, ch.Provider)
		}
	}
	if err := a.ValidateNotifyChannel(ch); err != nil {
		return err
	}
	return a.store.SaveNotifyChannel(ch)
}

func (a *App) DeleteNotifyChannel(id string) error {
	return a.store.DeleteNotifyChannel(id)
}

func (a *App) ValidateNotifyChannel(ch models.NotifyChannel) error {
	if err := notify.ValidateChannel(ch); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidNotifyChannel, err)
	}
	return nil
}

func mergeNotifyConfig(existing, incoming json.RawMessage, provider models.NotifyProvider) json.RawMessage {
	if len(incoming) == 0 {
		return existing
	}
	switch provider {
	case models.NotifyProviderTelegram:
		return mergeSecretField(existing, incoming, "token")
	case models.NotifyProviderSlack:
		return mergeSecretField(existing, incoming, "webhook_url")
	case models.NotifyProviderBale:
		return mergeSecretField(existing, incoming, "token")
	case models.NotifyProviderWebhook:
		return incoming
	case models.NotifyProviderKavenegar:
		return mergeSecretField(existing, incoming, "api_key")
	case models.NotifyProviderMeliPayamak:
		return mergeSecretField(existing, incoming, "password")
	default:
		return incoming
	}
}

func mergeSecretField(existing, incoming json.RawMessage, field string) json.RawMessage {
	var in map[string]json.RawMessage
	if err := json.Unmarshal(incoming, &in); err != nil {
		return incoming
	}
	val, ok := in[field]
	if !ok || len(val) == 0 || string(val) == `""` {
		var ex map[string]json.RawMessage
		if err := json.Unmarshal(existing, &ex); err == nil {
			if old, ok := ex[field]; ok {
				in[field] = old
			}
		}
	}
	out, err := json.Marshal(in)
	if err != nil {
		return incoming
	}
	return out
}

func IsNotifyChannelNotFound(err error) bool {
	return errors.Is(err, store.ErrNotifyChannelNotFound)
}

type notifyChannelStore struct {
	app *App
}

func (a *App) NotifyChannelStore() notify.ChannelStore {
	return notifyChannelStore{app: a}
}

func (a *App) NotifyDeps() (notify.ChannelStore, notify.HostNamer) {
	s := notifyChannelStore{app: a}
	return s, s
}

func (s notifyChannelStore) ListNotifyChannels() ([]models.NotifyChannel, error) {
	return s.app.ListNotifyChannelsFull()
}

func (s notifyChannelStore) GetNotifyChannel(id string) (models.NotifyChannel, error) {
	return s.app.GetNotifyChannelFull(id)
}

func (s notifyChannelStore) HostName(profileID string) string {
	if profileID == "" {
		return ""
	}
	for _, p := range s.app.Profiles() {
		if p.ID == profileID {
			if name := strings.TrimSpace(p.Name); name != "" {
				return name
			}
			break
		}
	}
	return profileID
}

func (s notifyChannelStore) TaskName(taskID string) string {
	if taskID == "" {
		return ""
	}
	task, err := s.app.GetTask(taskID)
	if err == nil {
		if name := strings.TrimSpace(task.Name); name != "" {
			return name
		}
	}
	return taskID
}
