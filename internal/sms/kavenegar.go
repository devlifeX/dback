package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type KavenegarConfig struct {
	APIKey string `json:"api_key"`
	Line   string `json:"line"`
}

type Kavenegar struct {
	Client HTTPDoer
}

func (Kavenegar) Validate(raw json.RawMessage) error {
	cfg, err := decodeKavenegar(raw)
	if err != nil {
		return err
	}
	if cfg.APIKey == "" {
		return fmt.Errorf("kavenegar api_key is required")
	}
	if cfg.Line == "" {
		return fmt.Errorf("kavenegar line is required")
	}
	return nil
}

func (Kavenegar) Redact(raw json.RawMessage) json.RawMessage {
	cfg, err := decodeKavenegar(raw)
	if err != nil {
		return raw
	}
	if cfg.APIKey != "" {
		cfg.APIKey = ""
	}
	out, _ := json.Marshal(cfg)
	return out
}

func (k Kavenegar) Send(ctx context.Context, raw json.RawMessage, to, body string) error {
	cfg, err := decodeKavenegar(raw)
	if err != nil {
		return err
	}
	if to == "" {
		return fmt.Errorf("recipient phone is required")
	}
	if body == "" {
		return fmt.Errorf("message body is required")
	}
	base := fmt.Sprintf("https://api.kavenegar.com/v1/%s/sms/send.json", url.PathEscape(cfg.APIKey))
	q := url.Values{}
	q.Set("sender", cfg.Line)
	q.Set("receptor", to)
	q.Set("message", body)
	reqURL := base + "?" + q.Encode()
	return doGETRequest(ctx, k.client(), reqURL)
}

func (k Kavenegar) client() HTTPDoer {
	if k.Client != nil {
		return k.Client
	}
	return DefaultHTTPClient()
}

func decodeKavenegar(raw json.RawMessage) (KavenegarConfig, error) {
	var cfg KavenegarConfig
	if len(raw) == 0 {
		return cfg, fmt.Errorf("kavenegar config is required")
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// KavenegarNotifyConfig extends the base config with a default recipient for notifications.
type KavenegarNotifyConfig struct {
	KavenegarConfig
	Receptor string `json:"receptor"`
}

func ValidateKavenegarNotify(raw json.RawMessage) error {
	cfg, err := DecodeKavenegarNotify(raw)
	if err != nil {
		return err
	}
	if err := (Kavenegar{}).Validate(raw); err != nil {
		return err
	}
	if cfg.Receptor == "" {
		return fmt.Errorf("kavenegar receptor is required")
	}
	return nil
}

func DecodeKavenegarNotify(raw json.RawMessage) (KavenegarNotifyConfig, error) {
	return decodeKavenegarNotify(raw)
}

func decodeKavenegarNotify(raw json.RawMessage) (KavenegarNotifyConfig, error) {
	var cfg KavenegarNotifyConfig
	if len(raw) == 0 {
		return cfg, fmt.Errorf("kavenegar config is required")
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func RedactKavenegarNotify(raw json.RawMessage) json.RawMessage {
	return (Kavenegar{}).Redact(raw)
}

// Ensure Kavenegar uses GET (not POST) for the public API.
var _ = http.MethodGet
