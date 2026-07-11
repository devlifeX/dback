package audit

import (
	"context"
	"sync"
	"time"

	"dback/internal/event"
)

type Entry struct {
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"`
	OperationID string    `json:"operation_id,omitempty"`
	ProfileID   string    `json:"profile_id,omitempty"`
	Kind        string    `json:"kind,omitempty"`
	Status      string    `json:"status,omitempty"`
	Message     string    `json:"message,omitempty"`
}

type Writer struct {
	mu      sync.RWMutex
	entries []Entry
	cap     int
	unsubs  []func()
}

func NewWriter(capacity int) *Writer {
	if capacity <= 0 {
		capacity = 500
	}
	return &Writer{cap: capacity}
}

func (w *Writer) Start(bus event.Bus) {
	subscribe := func(t event.Type, fn func(context.Context, event.Event)) {
		w.unsubs = append(w.unsubs, bus.Subscribe(t, func(ctx context.Context, ev event.Event) error {
			fn(ctx, ev)
			return nil
		}))
	}
	subscribe(event.TypeOperationCompleted, func(_ context.Context, ev event.Event) {
		e := ev.(event.OperationCompleted)
		w.append(Entry{
			Timestamp:   e.Timestamp,
			Type:        string(event.TypeOperationCompleted),
			OperationID: e.OperationID,
			ProfileID:   e.ProfileID,
			Kind:        string(e.Kind),
			Status:      string(e.Result.Status),
		})
	})
	subscribe(event.TypeOperationFailed, func(_ context.Context, ev event.Event) {
		e := ev.(event.OperationFailed)
		w.append(Entry{
			Timestamp:   e.Timestamp,
			Type:        string(event.TypeOperationFailed),
			OperationID: e.OperationID,
			ProfileID:   e.ProfileID,
			Kind:        string(e.Kind),
			Status:      string(e.Result.Status),
			Message:     e.Result.Error,
		})
	})
	subscribe(event.TypeTaskSkipped, func(_ context.Context, ev event.Event) {
		e := ev.(event.TaskSkipped)
		w.append(Entry{
			Timestamp:   e.Timestamp,
			Type:        string(event.TypeTaskSkipped),
			ProfileID:   e.ProfileID,
			Message:     e.Reason,
		})
	})
}

func (w *Writer) append(e Entry) {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.entries = append(w.entries, e)
	if len(w.entries) > w.cap {
		w.entries = w.entries[len(w.entries)-w.cap:]
	}
}

func (w *Writer) List(limit int) []Entry {
	if limit <= 0 {
		limit = 100
	}
	w.mu.RLock()
	defer w.mu.RUnlock()
	start := len(w.entries) - limit
	if start < 0 {
		start = 0
	}
	out := make([]Entry, len(w.entries[start:]))
	copy(out, w.entries[start:])
	return out
}

func (w *Writer) Stop() {
	for _, u := range w.unsubs {
		u()
	}
	w.unsubs = nil
}
