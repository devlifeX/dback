package notify

import (
	"fmt"
	"strings"

	"dback/internal/event"
	"dback/internal/operation"
)

func MessageFromEvent(ev event.Event, namer HostNamer) (Message, bool) {
	n := resolveNamer(namer)
	env := ev.Metadata()
	host := n.HostName(env.ProfileID)

	switch e := ev.(type) {
	case event.OperationStarted:
		return operationMessage(env, host, operation.Result{Status: operation.StatusRunning}, "started", "info"), true
	case event.OperationCompleted:
		return operationMessage(env, host, e.Result, "succeeded", "info"), true
	case event.OperationFailed:
		return operationMessage(env, host, e.Result, "failed", "error"), true
	case event.OperationCanceled:
		return operationMessage(env, host, e.Result, "canceled", "warn"), true
	case event.TaskSkipped:
		task := n.TaskName(e.TaskID)
		return Message{
			Title: "⏭️ Task skipped",
			Body: SanitizeText(fmt.Sprintf(
				"Task: %s\nHost: %s\nReason: %s",
				task, host, e.Reason,
			)),
			Event: env.Type,
			Level: "warn",
			Fields: map[string]string{
				"task_id":    e.TaskID,
				"task_name":  task,
				"profile_id": env.ProfileID,
				"host_name":  host,
			},
		}, true
	default:
		return Message{}, false
	}
}

func operationMessage(env event.Envelope, host string, res operation.Result, label, level string) Message {
	kind := env.Kind
	action := kindLabel(kind)
	emoji := eventEmoji(label)
	kindIcon := kindEmoji(kind)

	title := fmt.Sprintf("%s %s %s", emoji, kindIcon, action+" "+statusWord(label))
	body := fmt.Sprintf("Host: %s", host)
	if errText := SanitizeError(res.Error); errText != "" {
		body += "\nError: " + errText
	}

	return Message{
		Title: SanitizeText(title),
		Body:  SanitizeText(body),
		Event: env.Type,
		Level: level,
		Fields: map[string]string{
			"operation_id": env.OperationID,
			"kind":         string(kind),
			"profile_id":   env.ProfileID,
			"host_name":    host,
			"status":       string(res.Status),
		},
	}
}

func eventEmoji(label string) string {
	switch label {
	case "started":
		return "▶️"
	case "succeeded":
		return "✅"
	case "failed":
		return "❌"
	case "canceled":
		return "⏹️"
	default:
		return "ℹ️"
	}
}

func kindEmoji(kind operation.Kind) string {
	switch kind {
	case operation.KindBackupDB:
		return "💾"
	case operation.KindBackupFiles:
		return "📁"
	case operation.KindUpload:
		return "☁️"
	case operation.KindRestore:
		return "♻️"
	case operation.KindDeepVerify:
		return "🔍"
	default:
		return "⚙️"
	}
}

func kindLabel(kind operation.Kind) string {
	switch kind {
	case operation.KindBackupDB:
		return "Database backup"
	case operation.KindBackupFiles:
		return "File backup"
	case operation.KindUpload:
		return "Remote upload"
	case operation.KindRestore:
		return "Restore"
	case operation.KindDeepVerify:
		return "Deep verify"
	default:
		return strings.ReplaceAll(string(kind), "_", " ")
	}
}

func statusWord(label string) string {
	switch label {
	case "started":
		return "started"
	case "succeeded":
		return "succeeded"
	case "failed":
		return "failed"
	case "canceled":
		return "canceled"
	default:
		return label
	}
}

func TestMessage() Message {
	return Message{
		Title: "🔔 DBack test",
		Body:  SanitizeText("Test notification — your channel is configured correctly."),
		Event: event.TypeOperationCompleted,
		Level: "info",
	}
}

func FormatPlain(msg Message) string {
	var b strings.Builder
	if msg.Title != "" {
		b.WriteString(msg.Title)
	}
	if msg.Body != "" {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(msg.Body)
	}
	return b.String()
}
