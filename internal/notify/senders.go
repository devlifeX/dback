package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"dback/internal/sms"
)

const defaultHTTPTimeout = 10 * time.Second
const maxResponseBytes = 1 << 20

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

func DefaultHTTPClient() *http.Client {
	return &http.Client{Timeout: defaultHTTPTimeout}
}

func doJSONRequest(ctx context.Context, client HTTPDoer, method, url string, headers map[string]string, payload any) error {
	if client == nil {
		client = DefaultHTTPClient()
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		if isAllowedHeader(k) {
			req.Header.Set(k, v)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBytes))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}

func isAllowedHeader(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "content-type", "accept", "user-agent", "x-request-id":
		return true
	default:
		return strings.HasPrefix(strings.ToLower(name), "x-dback-")
	}
}

type TelegramConfig struct {
	Token     string `json:"token"`
	ChatID    string `json:"chat_id"`
	ThreadID  int64  `json:"thread_id,omitempty"`
	ParseMode string `json:"parse_mode,omitempty"`
}

type TelegramSender struct {
	Client HTTPDoer
}

func (TelegramSender) Validate(raw json.RawMessage) error {
	cfg, err := decodeTelegram(raw)
	if err != nil {
		return err
	}
	if cfg.Token == "" {
		return fmt.Errorf("telegram token is required")
	}
	if cfg.ChatID == "" {
		return fmt.Errorf("telegram chat_id is required")
	}
	return nil
}

func (TelegramSender) Redact(raw json.RawMessage) json.RawMessage {
	cfg, err := decodeTelegram(raw)
	if err != nil {
		return raw
	}
	if cfg.Token != "" {
		cfg.Token = ""
	}
	out, _ := json.Marshal(cfg)
	return out
}

func (s TelegramSender) Send(ctx context.Context, raw json.RawMessage, msg Message) error {
	cfg, err := decodeTelegram(raw)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.Token)
	payload := map[string]any{
		"chat_id": cfg.ChatID,
		"text":    FormatPlain(msg),
	}
	if cfg.ThreadID != 0 {
		payload["message_thread_id"] = cfg.ThreadID
	}
	if cfg.ParseMode != "" {
		payload["parse_mode"] = cfg.ParseMode
	}
	return doJSONRequest(ctx, s.client(), http.MethodPost, url, nil, payload)
}

func (s TelegramSender) client() HTTPDoer {
	if s.Client != nil {
		return s.Client
	}
	return DefaultHTTPClient()
}

func decodeTelegram(raw json.RawMessage) (TelegramConfig, error) {
	var cfg TelegramConfig
	if len(raw) == 0 {
		return cfg, fmt.Errorf("telegram config is required")
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

type SlackConfig struct {
	WebhookURL string `json:"webhook_url"`
	Channel    string `json:"channel,omitempty"`
	Username   string `json:"username,omitempty"`
	IconEmoji  string `json:"icon_emoji,omitempty"`
}

type SlackSender struct {
	Client HTTPDoer
}

func (SlackSender) Validate(raw json.RawMessage) error {
	cfg, err := decodeSlack(raw)
	if err != nil {
		return err
	}
	if cfg.WebhookURL == "" {
		return fmt.Errorf("slack webhook_url is required")
	}
	if _, err := ValidateHTTPSURL(cfg.WebhookURL); err != nil {
		return fmt.Errorf("slack webhook_url: %w", err)
	}
	return nil
}

func (SlackSender) Redact(raw json.RawMessage) json.RawMessage {
	cfg, err := decodeSlack(raw)
	if err != nil {
		return raw
	}
	if cfg.WebhookURL != "" {
		cfg.WebhookURL = ""
	}
	out, _ := json.Marshal(cfg)
	return out
}

func (s SlackSender) Send(ctx context.Context, raw json.RawMessage, msg Message) error {
	cfg, err := decodeSlack(raw)
	if err != nil {
		return err
	}
	payload := map[string]any{"text": FormatPlain(msg)}
	if cfg.Channel != "" {
		payload["channel"] = cfg.Channel
	}
	if cfg.Username != "" {
		payload["username"] = cfg.Username
	}
	if cfg.IconEmoji != "" {
		payload["icon_emoji"] = cfg.IconEmoji
	}
	return doJSONRequest(ctx, s.client(), http.MethodPost, cfg.WebhookURL, nil, payload)
}

func (s SlackSender) client() HTTPDoer {
	if s.Client != nil {
		return s.Client
	}
	return DefaultHTTPClient()
}

func decodeSlack(raw json.RawMessage) (SlackConfig, error) {
	var cfg SlackConfig
	if len(raw) == 0 {
		return cfg, fmt.Errorf("slack config is required")
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

type BaleConfig struct {
	Token  string `json:"token"`
	ChatID string `json:"chat_id"`
}

type BaleSender struct {
	Client HTTPDoer
}

func (BaleSender) Validate(raw json.RawMessage) error {
	cfg, err := decodeBale(raw)
	if err != nil {
		return err
	}
	if cfg.Token == "" {
		return fmt.Errorf("bale token is required")
	}
	if cfg.ChatID == "" {
		return fmt.Errorf("bale chat_id is required")
	}
	return nil
}

func (BaleSender) Redact(raw json.RawMessage) json.RawMessage {
	cfg, err := decodeBale(raw)
	if err != nil {
		return raw
	}
	if cfg.Token != "" {
		cfg.Token = ""
	}
	out, _ := json.Marshal(cfg)
	return out
}

func (s BaleSender) Send(ctx context.Context, raw json.RawMessage, msg Message) error {
	cfg, err := decodeBale(raw)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://tapi.bale.ai/bot%s/sendMessage", cfg.Token)
	payload := map[string]any{
		"chat_id": cfg.ChatID,
		"text":    FormatPlain(msg),
	}
	return doJSONRequest(ctx, s.client(), http.MethodPost, url, nil, payload)
}

func (s BaleSender) client() HTTPDoer {
	if s.Client != nil {
		return s.Client
	}
	return DefaultHTTPClient()
}

func decodeBale(raw json.RawMessage) (BaleConfig, error) {
	var cfg BaleConfig
	if len(raw) == 0 {
		return cfg, fmt.Errorf("bale config is required")
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

type WebhookConfig struct {
	URL        string            `json:"url"`
	Method     string            `json:"method,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	TimeoutSec int               `json:"timeout_sec,omitempty"`
	RetryCount int               `json:"retry_count,omitempty"`
}

type WebhookSender struct {
	Client HTTPDoer
}

func (WebhookSender) Validate(raw json.RawMessage) error {
	cfg, err := decodeWebhook(raw)
	if err != nil {
		return err
	}
	if cfg.URL == "" {
		return fmt.Errorf("webhook url is required")
	}
	if _, err := ValidateHTTPSURL(cfg.URL); err != nil {
		return fmt.Errorf("webhook url: %w", err)
	}
	method := cfg.Method
	if method == "" {
		method = http.MethodPost
	}
	if err := ValidateHTTPMethod(method); err != nil {
		return err
	}
	if cfg.TimeoutSec < 0 || cfg.TimeoutSec > 60 {
		return fmt.Errorf("webhook timeout_sec must be between 0 and 60")
	}
	if cfg.RetryCount < 0 || cfg.RetryCount > 5 {
		return fmt.Errorf("webhook retry_count must be between 0 and 5")
	}
	return nil
}

func (WebhookSender) Redact(raw json.RawMessage) json.RawMessage {
	cfg, err := decodeWebhook(raw)
	if err != nil {
		return raw
	}
	for k := range cfg.Headers {
		if strings.Contains(strings.ToLower(k), "authorization") || strings.Contains(strings.ToLower(k), "token") {
			cfg.Headers[k] = ""
		}
	}
	out, _ := json.Marshal(cfg)
	return out
}

func (s WebhookSender) Send(ctx context.Context, raw json.RawMessage, msg Message) error {
	cfg, err := decodeWebhook(raw)
	if err != nil {
		return err
	}
	if _, err := ValidateHTTPSURL(cfg.URL); err != nil {
		return err
	}
	method := cfg.Method
	if method == "" {
		method = http.MethodPost
	}
	payload := map[string]any{
		"title":  msg.Title,
		"body":   msg.Body,
		"level":  msg.Level,
		"event":  string(msg.Event),
		"fields": msg.Fields,
	}
	client := s.client()
	if cfg.TimeoutSec > 0 {
		if c, ok := client.(*http.Client); ok && c.Timeout == defaultHTTPTimeout {
			client = &http.Client{Timeout: time.Duration(cfg.TimeoutSec) * time.Second}
		}
	}
	return doJSONRequest(ctx, client, strings.ToUpper(method), cfg.URL, cfg.Headers, payload)
}

func (s WebhookSender) client() HTTPDoer {
	if s.Client != nil {
		return s.Client
	}
	return DefaultHTTPClient()
}

func decodeWebhook(raw json.RawMessage) (WebhookConfig, error) {
	var cfg WebhookConfig
	if len(raw) == 0 {
		return cfg, fmt.Errorf("webhook config is required")
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

type KavenegarNotifySender struct {
	Client HTTPDoer
	Provider sms.Provider
}

func (KavenegarNotifySender) Validate(raw json.RawMessage) error {
	return sms.ValidateKavenegarNotify(raw)
}

func (KavenegarNotifySender) Redact(raw json.RawMessage) json.RawMessage {
	return sms.RedactKavenegarNotify(raw)
}

func (s KavenegarNotifySender) Send(ctx context.Context, raw json.RawMessage, msg Message) error {
	cfg, err := sms.DecodeKavenegarNotify(raw)
	if err != nil {
		return err
	}
	p := s.Provider
	if p == nil {
		p = sms.Kavenegar{Client: s.Client}
	}
	return p.Send(ctx, raw, cfg.Receptor, FormatPlain(msg))
}

type MeliPayamakNotifySender struct {
	Client HTTPDoer
	Provider sms.Provider
}

func (MeliPayamakNotifySender) Validate(raw json.RawMessage) error {
	return sms.ValidateMeliPayamakNotify(raw)
}

func (MeliPayamakNotifySender) Redact(raw json.RawMessage) json.RawMessage {
	return sms.RedactMeliPayamakNotify(raw)
}

func (s MeliPayamakNotifySender) Send(ctx context.Context, raw json.RawMessage, msg Message) error {
	cfg, err := sms.DecodeMeliPayamakNotify(raw)
	if err != nil {
		return err
	}
	p := s.Provider
	if p == nil {
		p = sms.MeliPayamak{Client: s.Client}
	}
	return p.Send(ctx, raw, cfg.To, FormatPlain(msg))
}
