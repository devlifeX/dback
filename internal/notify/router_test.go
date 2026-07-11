package notify

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"dback/internal/event"
	"dback/internal/operation"
	"dback/models"
)

type memChannelStore struct {
	channels []models.NotifyChannel
}

func (m *memChannelStore) ListNotifyChannels() ([]models.NotifyChannel, error) {
	return append([]models.NotifyChannel(nil), m.channels...), nil
}

func (m *memChannelStore) GetNotifyChannel(id string) (models.NotifyChannel, error) {
	for _, ch := range m.channels {
		if ch.ID == id {
			return ch, nil
		}
	}
	return models.NotifyChannel{}, errors.New("not found")
}

type stubSender struct {
	hits *int32
}

func (s stubSender) Send(context.Context, json.RawMessage, Message) error {
	atomic.AddInt32(s.hits, 1)
	return nil
}

func (s stubSender) Validate(json.RawMessage) error { return nil }

func (s stubSender) Redact(raw json.RawMessage) json.RawMessage { return raw }

func TestRouterDeliversOnOperationFailed(t *testing.T) {
	var hits int32
	store := &memChannelStore{channels: []models.NotifyChannel{{
		ID:       "ch1",
		Name:     "ops",
		Provider: models.NotifyProviderWebhook,
		Enabled:  true,
		Events:   []string{string(event.TypeOperationFailed)},
		Config:   json.RawMessage(`{}`),
	}}}

	bus := event.NewMemoryBus(4)
	reg := NewRegistry()
	reg.Register(models.NotifyProviderWebhook, stubSender{hits: &hits})
	router := NewRouter(bus, store, reg, nil)
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
		Result: operation.Result{Status: operation.StatusFailed, Error: "password=secret"},
	})

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&hits) >= 1 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("expected delivery, hits=%d", hits)
}

func TestRouterFiltersByEvent(t *testing.T) {
	var hits int32
	store := &memChannelStore{channels: []models.NotifyChannel{{
		ID:       "ch1",
		Provider: models.NotifyProviderWebhook,
		Enabled:  true,
		Events:   []string{string(event.TypeOperationCompleted)},
		Config:   json.RawMessage(`{}`),
	}}}

	bus := event.NewMemoryBus(4)
	reg := NewRegistry()
	reg.Register(models.NotifyProviderWebhook, stubSender{hits: &hits})
	router := NewRouter(bus, store, reg, nil)
	router.Start()
	defer router.Stop()

	_ = bus.Publish(context.Background(), event.OperationFailed{
		Envelope: event.Envelope{Type: event.TypeOperationFailed, Timestamp: time.Now()},
		Result:   operation.Result{Status: operation.StatusFailed},
	})
	time.Sleep(100 * time.Millisecond)
	if atomic.LoadInt32(&hits) != 0 {
		t.Fatalf("expected no delivery for filtered event, hits=%d", hits)
	}
}

func TestSendWithRetry(t *testing.T) {
	var attempts int
	err := SendWithRetry(context.Background(), func(context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary")
		}
		return nil
	}, []time.Duration{1 * time.Millisecond, 1 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestRouterDeliversOnOperationCompleted(t *testing.T) {
	var hits int32
	store := &memChannelStore{channels: []models.NotifyChannel{{
		ID:       "ch1",
		Provider: models.NotifyProviderWebhook,
		Enabled:  true,
		Events:   []string{string(event.TypeOperationCompleted)},
		Config:   json.RawMessage(`{}`),
	}}}

	bus := event.NewMemoryBus(4)
	reg := NewRegistry()
	reg.Register(models.NotifyProviderWebhook, stubSender{hits: &hits})
	router := NewRouter(bus, store, reg, nil)
	router.Start()
	defer router.Stop()

	publishCtx, cancel := context.WithCancel(context.Background())
	cancel()
	_ = bus.Publish(publishCtx, event.OperationCompleted{
		Envelope: event.Envelope{
			Type:        event.TypeOperationCompleted,
			OperationID: "op1",
			Kind:        operation.KindBackupDB,
			ProfileID:   "p1",
			Timestamp:   time.Now(),
		},
		Result: operation.Result{Status: operation.StatusSucceeded},
	})

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&hits) >= 1 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("expected delivery despite canceled publish context, hits=%d", hits)
}
