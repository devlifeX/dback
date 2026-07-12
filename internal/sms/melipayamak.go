package sms

import (
	"context"
	"encoding/json"
	"fmt"
)

type MeliPayamakConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
}

type MeliPayamak struct {
	Client HTTPDoer
}

func (MeliPayamak) Validate(raw json.RawMessage) error {
	cfg, err := decodeMeliPayamak(raw)
	if err != nil {
		return err
	}
	if cfg.Username == "" {
		return fmt.Errorf("melipayamak username is required")
	}
	if cfg.Password == "" {
		return fmt.Errorf("melipayamak password is required")
	}
	if cfg.From == "" {
		return fmt.Errorf("melipayamak from is required")
	}
	return nil
}

func (MeliPayamak) Redact(raw json.RawMessage) json.RawMessage {
	cfg, err := decodeMeliPayamak(raw)
	if err != nil {
		return raw
	}
	if cfg.Password != "" {
		cfg.Password = ""
	}
	out, _ := json.Marshal(cfg)
	return out
}

func (m MeliPayamak) Send(ctx context.Context, raw json.RawMessage, to, body string) error {
	cfg, err := decodeMeliPayamak(raw)
	if err != nil {
		return err
	}
	if to == "" {
		return fmt.Errorf("recipient phone is required")
	}
	if body == "" {
		return fmt.Errorf("message body is required")
	}
	payload := map[string]string{
		"username": cfg.Username,
		"password": cfg.Password,
		"to":       to,
		"from":     cfg.From,
		"text":     body,
	}
	return doJSONRequest(ctx, m.client(), "POST", "https://rest.payamak-panel.com/api/SendSMS/SendSMS", nil, payload)
}

func (m MeliPayamak) client() HTTPDoer {
	if m.Client != nil {
		return m.Client
	}
	return DefaultHTTPClient()
}

func decodeMeliPayamak(raw json.RawMessage) (MeliPayamakConfig, error) {
	var cfg MeliPayamakConfig
	if len(raw) == 0 {
		return cfg, fmt.Errorf("melipayamak config is required")
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

type MeliPayamakNotifyConfig struct {
	MeliPayamakConfig
	To string `json:"to"`
}

func ValidateMeliPayamakNotify(raw json.RawMessage) error {
	cfg, err := decodeMeliPayamakNotify(raw)
	if err != nil {
		return err
	}
	if err := (MeliPayamak{}).Validate(raw); err != nil {
		return err
	}
	if cfg.To == "" {
		return fmt.Errorf("melipayamak to is required")
	}
	return nil
}

func DecodeMeliPayamakNotify(raw json.RawMessage) (MeliPayamakNotifyConfig, error) {
	return decodeMeliPayamakNotify(raw)
}

func decodeMeliPayamakNotify(raw json.RawMessage) (MeliPayamakNotifyConfig, error) {
	var cfg MeliPayamakNotifyConfig
	if len(raw) == 0 {
		return cfg, fmt.Errorf("melipayamak config is required")
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func RedactMeliPayamakNotify(raw json.RawMessage) json.RawMessage {
	return (MeliPayamak{}).Redact(raw)
}
