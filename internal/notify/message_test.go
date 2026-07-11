package notify

import (
	"strings"
	"testing"
	"time"

	"dback/internal/event"
	"dback/internal/operation"
)

type stubNamer struct {
	hosts map[string]string
	tasks map[string]string
}

func (s stubNamer) HostName(id string) string {
	if n := s.hosts[id]; n != "" {
		return n
	}
	return id
}

func (s stubNamer) TaskName(id string) string {
	if n := s.tasks[id]; n != "" {
		return n
	}
	return id
}

func TestMessageFromEventUsesHostNameAndEmoji(t *testing.T) {
	namer := stubNamer{hosts: map[string]string{"p1": "Parned"}}
	msg, ok := MessageFromEvent(event.OperationStarted{
		Envelope: event.Envelope{
			Type:        event.TypeOperationStarted,
			OperationID: "op-1",
			Kind:        operation.KindBackupDB,
			ProfileID:   "p1",
			Timestamp:   time.Now(),
		},
	}, namer)
	if !ok {
		t.Fatal("expected message")
	}
	if !strings.Contains(msg.Title, "▶️") || !strings.Contains(msg.Title, "💾") {
		t.Fatalf("title missing emoji: %q", msg.Title)
	}
	if !strings.Contains(msg.Title, "Database backup started") {
		t.Fatalf("title: %q", msg.Title)
	}
	if msg.Body != "Host: Parned" {
		t.Fatalf("body: %q", msg.Body)
	}
	if strings.Contains(msg.Body, "p1") {
		t.Fatalf("body should not contain profile id: %q", msg.Body)
	}
}

func TestMessageFromEventSucceeded(t *testing.T) {
	namer := stubNamer{hosts: map[string]string{"p1": "Rade-Localhost"}}
	msg, ok := MessageFromEvent(event.OperationCompleted{
		Envelope: event.Envelope{
			Type:        event.TypeOperationCompleted,
			OperationID: "op-2",
			Kind:        operation.KindBackupDB,
			ProfileID:   "p1",
			Timestamp:   time.Now(),
		},
		Result: operation.Result{Status: operation.StatusSucceeded},
	}, namer)
	if !ok {
		t.Fatal("expected message")
	}
	if !strings.Contains(msg.Title, "✅") {
		t.Fatalf("title: %q", msg.Title)
	}
	if msg.Body != "Host: Rade-Localhost" {
		t.Fatalf("body: %q", msg.Body)
	}
}

func TestMessageFromEventFailedIncludesError(t *testing.T) {
	msg, _ := MessageFromEvent(event.OperationFailed{
		Envelope: event.Envelope{
			Type:        event.TypeOperationFailed,
			Kind:        operation.KindUpload,
			ProfileID:   "p9",
			Timestamp:   time.Now(),
		},
		Result: operation.Result{Status: operation.StatusFailed, Error: "timeout"},
	}, stubNamer{hosts: map[string]string{"p9": "MyHost"}})
	if !strings.Contains(msg.Title, "❌") || !strings.Contains(msg.Title, "☁️") {
		t.Fatalf("title: %q", msg.Title)
	}
	if !strings.Contains(msg.Body, "Host: MyHost") || !strings.Contains(msg.Body, "Error: timeout") {
		t.Fatalf("body: %q", msg.Body)
	}
}

func TestMessageTaskSkipped(t *testing.T) {
	msg, ok := MessageFromEvent(event.TaskSkipped{
		Envelope: event.Envelope{Type: event.TypeTaskSkipped, ProfileID: "p1", Timestamp: time.Now()},
		TaskID:   "t1",
		Reason:   "overlap",
	}, stubNamer{
		hosts: map[string]string{"p1": "Parned"},
		tasks: map[string]string{"t1": "parned-db"},
	})
	if !ok {
		t.Fatal("expected message")
	}
	if msg.Title != "⏭️ Task skipped" {
		t.Fatalf("title: %q", msg.Title)
	}
	if !strings.Contains(msg.Body, "Task: parned-db") || !strings.Contains(msg.Body, "Host: Parned") {
		t.Fatalf("body: %q", msg.Body)
	}
}
