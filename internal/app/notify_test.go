package app

import (
	"encoding/json"
	"testing"

	"dback/models"
)

func TestNotifyChannelSecretMerge(t *testing.T) {
	dir := t.TempDir()
	a, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.CreateVault("test-pass"); err != nil {
		t.Fatal(err)
	}
	cfg, _ := json.Marshal(map[string]string{
		"token":   "secret-token",
		"chat_id": "123",
	})
	ch := models.NotifyChannel{
		Name:     "ops",
		Provider: models.NotifyProviderTelegram,
		Enabled:  true,
		Config:   cfg,
	}
	if err := a.SaveNotifyChannel(ch); err != nil {
		t.Fatal(err)
	}
	channels, err := a.ListNotifyChannels()
	if err != nil {
		t.Fatal(err)
	}
	if len(channels) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(channels))
	}
	if string(channels[0].Config) == string(cfg) {
		t.Fatal("expected redacted config in list")
	}

	updateCfg, _ := json.Marshal(map[string]string{"chat_id": "456"})
	channels[0].Config = updateCfg
	if err := a.SaveNotifyChannel(channels[0]); err != nil {
		t.Fatal(err)
	}
	full, err := a.GetNotifyChannelFull(channels[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]string
	if err := json.Unmarshal(full.Config, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["token"] != "secret-token" {
		t.Fatalf("expected token preserved, got %q", decoded["token"])
	}
	if decoded["chat_id"] != "456" {
		t.Fatalf("expected chat_id updated, got %q", decoded["chat_id"])
	}
}

func TestNotifyChannelVaultRoundTrip(t *testing.T) {
	dir := t.TempDir()
	a, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.CreateVault("test-pass"); err != nil {
		t.Fatal(err)
	}
	cfg, _ := json.Marshal(map[string]string{"url": "https://example.com/hook", "method": "POST"})
	if err := a.SaveNotifyChannel(models.NotifyChannel{
		Name:     "webhook",
		Provider: models.NotifyProviderWebhook,
		Enabled:  true,
		Config:   cfg,
	}); err != nil {
		t.Fatal(err)
	}
	a.Lock()
	if err := a.Unlock("test-pass"); err != nil {
		t.Fatal(err)
	}
	channels, err := a.ListNotifyChannelsFull()
	if err != nil {
		t.Fatal(err)
	}
	if len(channels) != 1 || channels[0].Name != "webhook" {
		t.Fatalf("unexpected channels: %+v", channels)
	}
}
