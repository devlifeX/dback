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
