package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"dback/internal/event"
	"dback/models"
)

type ProviderID = models.NotifyProvider

const (
	ProviderTelegram = models.NotifyProviderTelegram
	ProviderSlack    = models.NotifyProviderSlack
	ProviderBale     = models.NotifyProviderBale
	ProviderWebhook  = models.NotifyProviderWebhook
)

type Message struct {
	Title   string
	Body    string
	Event   event.Type
	Level   string
	Fields  map[string]string
}

type Sender interface {
	Send(ctx context.Context, cfg json.RawMessage, msg Message) error
	Validate(cfg json.RawMessage) error
	Redact(cfg json.RawMessage) json.RawMessage
}

type Registry struct {
	senders map[ProviderID]Sender
}

func NewRegistry() *Registry {
	r := &Registry{senders: map[ProviderID]Sender{}}
	r.Register(ProviderTelegram, TelegramSender{})
	r.Register(ProviderSlack, SlackSender{})
	r.Register(ProviderBale, BaleSender{})
	r.Register(ProviderWebhook, WebhookSender{})
	return r
}

func (r *Registry) Register(id ProviderID, sender Sender) {
	r.senders[id] = sender
}

func (r *Registry) Sender(id ProviderID) (Sender, bool) {
	s, ok := r.senders[id]
	return s, ok
}

func ValidateChannel(ch models.NotifyChannel) error {
	if ch.Name == "" {
		return fmt.Errorf("channel name is required")
	}
	switch ch.Provider {
	case ProviderTelegram, ProviderSlack, ProviderBale, ProviderWebhook:
	default:
		return fmt.Errorf("unsupported provider %q", ch.Provider)
	}
	r := NewRegistry()
	sender, ok := r.Sender(ch.Provider)
	if !ok {
		return fmt.Errorf("no sender for provider %q", ch.Provider)
	}
	return sender.Validate(ch.Config)
}

func RedactChannelConfig(ch models.NotifyChannel) models.NotifyChannel {
	r := NewRegistry()
	if sender, ok := r.Sender(ch.Provider); ok {
		ch.Config = sender.Redact(ch.Config)
	}
	return ch
}

func DefaultRetryDelays() []time.Duration {
	return []time.Duration{500 * time.Millisecond, time.Second, 2 * time.Second}
}
