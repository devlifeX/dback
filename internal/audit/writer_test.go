package audit_test

import (
	"context"
	"testing"
	"time"

	"dback/internal/audit"
	"dback/internal/event"
	"dback/internal/operation"
)

func TestWriterRecordsOperationFailed(t *testing.T) {
	bus := event.NewMemoryBus(4)
	w := audit.NewWriter(10)
	w.Start(bus)

	_ = bus.Publish(context.Background(), event.OperationFailed{
		Envelope: event.Envelope{
			Type:        event.TypeOperationFailed,
			OperationID: "op-1",
			Kind:        operation.KindUpload,
			ProfileID:   "host-1",
			Timestamp:   time.Now(),
		},
		Result: operation.Result{Status: operation.StatusFailed, Error: "timeout"},
	})
	time.Sleep(20 * time.Millisecond)
	entries := w.List(10)
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}
	if entries[0].Kind != string(operation.KindUpload) {
		t.Fatalf("unexpected kind %q", entries[0].Kind)
	}
	w.Stop()
}

func TestWriterListNewestFirst(t *testing.T) {
	bus := event.NewMemoryBus(8)
	w := audit.NewWriter(10)
	w.Start(bus)

	t1 := time.Now().Add(-2 * time.Minute)
	t2 := time.Now().Add(-time.Minute)
	t3 := time.Now()
	for _, ts := range []time.Time{t1, t2, t3} {
		_ = bus.Publish(context.Background(), event.OperationCompleted{
			Envelope: event.Envelope{
				Type:        event.TypeOperationCompleted,
				OperationID: "op-" + ts.Format("150405"),
				Kind:        operation.KindBackupDB,
				Timestamp:   ts,
			},
			Result: operation.Result{Status: operation.StatusSucceeded},
		})
	}
	time.Sleep(30 * time.Millisecond)

	entries := w.List(10)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if !entries[0].Timestamp.After(entries[1].Timestamp) || !entries[1].Timestamp.After(entries[2].Timestamp) {
		t.Fatalf("expected newest first, got %#v", entries)
	}
	w.Stop()
}
