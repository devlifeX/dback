package metrics_test

import (
	"context"
	"testing"
	"time"

	"dback/internal/event"
	"dback/internal/metrics"
	"dback/internal/operation"
)

func TestCollectorIncrementsOnOperationCompleted(t *testing.T) {
	bus := event.NewMemoryBus(8)
	c := metrics.NewCollector()
	c.Start(bus)

	_ = bus.Publish(context.Background(), event.OperationStarted{
		Envelope: event.Envelope{
			Type:        event.TypeOperationStarted,
			OperationID: "op1",
			Kind:        operation.KindBackupDB,
			ProfileID:   "p1",
			Timestamp:   time.Now(),
		},
		Spec: operation.Spec{Kind: operation.KindBackupDB},
	})
	_ = bus.Publish(context.Background(), event.OperationCompleted{
		Envelope: event.Envelope{
			Type:        event.TypeOperationCompleted,
			OperationID: "op1",
			Kind:        operation.KindBackupDB,
			ProfileID:   "p1",
			Timestamp:   time.Now(),
		},
		Result: operation.Result{Status: operation.StatusSucceeded},
	})

	time.Sleep(20 * time.Millisecond)
	c.Stop()
}
