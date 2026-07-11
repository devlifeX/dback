package integration_test

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"dback/internal/app"
	"dback/internal/controlplane"
	"dback/internal/event"
	"dback/internal/notify"
	"dback/internal/operation"
	"dback/models"
)

type stubSender struct {
	hits *int32
}

func (s stubSender) Send(context.Context, json.RawMessage, notify.Message) error {
	atomic.AddInt32(s.hits, 1)
	return nil
}

func (s stubSender) Validate(json.RawMessage) error { return nil }

func (s stubSender) Redact(raw json.RawMessage) json.RawMessage { return raw }

func TestOperationEventToNotifyPipeline(t *testing.T) {
	var hits int32
	dir := t.TempDir()
	a, err := app.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.CreateVault("test-pass"); err != nil {
		t.Fatal(err)
	}
	cfg, _ := json.Marshal(map[string]string{"url": "https://example.com/hook"})
	if err := a.SaveNotifyChannel(models.NotifyChannel{
		Name:     "stub",
		Provider: models.NotifyProviderWebhook,
		Enabled:  true,
		Events:   []string{string(event.TypeOperationFailed)},
		Config:   cfg,
	}); err != nil {
		t.Fatal(err)
	}

	bus := event.NewMemoryBus(16)
	_ = controlplane.NewService(a, 16, 2)
	reg := notify.NewRegistry()
	reg.Register(models.NotifyProviderWebhook, stubSender{hits: &hits})
	notifyStore, notifyNamer := a.NotifyDeps()
	router := notify.NewRouter(bus, notifyStore, reg, notifyNamer)
	router.Start()
	defer router.Stop()

	_ = bus.Publish(context.Background(), event.OperationFailed{
		Envelope: event.Envelope{
			Type:        event.TypeOperationFailed,
			OperationID: "op1",
			Kind:        operation.KindBackupDB,
			ProfileID:   "p1",
			Timestamp:   time.Now(),
		},
		Result: operation.Result{Status: operation.StatusFailed, Error: "test failure"},
	})

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&hits) >= 1 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("notify pipeline did not deliver")
}
