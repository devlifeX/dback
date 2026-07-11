package models

import "encoding/json"

type NotifyProvider string

const (
	NotifyProviderTelegram NotifyProvider = "telegram"
	NotifyProviderSlack    NotifyProvider = "slack"
	NotifyProviderBale     NotifyProvider = "bale"
	NotifyProviderWebhook  NotifyProvider = "webhook"
)

type NotifyChannel struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Provider NotifyProvider  `json:"provider"`
	Enabled  bool            `json:"enabled"`
	Events   []string        `json:"events,omitempty"`
	Config   json.RawMessage `json:"config,omitempty"`
}

func (c NotifyChannel) SubscribesTo(eventType string) bool {
	if len(c.Events) == 0 {
		return true
	}
	for _, e := range c.Events {
		if e == eventType {
			return true
		}
	}
	return false
}
