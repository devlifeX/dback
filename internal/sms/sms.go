package sms

import (
	"context"
	"encoding/json"
	"fmt"
)

type ProviderID string

const (
	ProviderKavenegar   ProviderID = "kavenegar"
	ProviderMeliPayamak ProviderID = "melipayamak"
)

type Provider interface {
	Send(ctx context.Context, cfg json.RawMessage, to, body string) error
	Validate(cfg json.RawMessage) error
	Redact(cfg json.RawMessage) json.RawMessage
}

type Registry struct {
	providers map[ProviderID]Provider
}

func NewRegistry() *Registry {
	r := &Registry{providers: map[ProviderID]Provider{}}
	r.Register(ProviderKavenegar, Kavenegar{})
	r.Register(ProviderMeliPayamak, MeliPayamak{})
	return r
}

func (r *Registry) Register(id ProviderID, p Provider) {
	r.providers[id] = p
}

func (r *Registry) Provider(id ProviderID) (Provider, bool) {
	p, ok := r.providers[id]
	return p, ok
}

func ValidateProvider(id ProviderID, cfg json.RawMessage) error {
	r := NewRegistry()
	p, ok := r.Provider(id)
	if !ok {
		return fmt.Errorf("unsupported sms provider %q", id)
	}
	return p.Validate(cfg)
}

func RedactConfig(id ProviderID, cfg json.RawMessage) json.RawMessage {
	r := NewRegistry()
	if p, ok := r.Provider(id); ok {
		return p.Redact(cfg)
	}
	return cfg
}
