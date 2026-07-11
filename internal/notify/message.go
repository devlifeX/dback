package notify

import (
	"fmt"
	"strings"

	"dback/internal/event"
	"dback/internal/operation"
)

func MessageFromEvent(ev event.Event) (Message, bool) {
	env := ev.Metadata()
	switch e := ev.(type) {
	case event.OperationCompleted:
		return operationMessage(env, e.Result, "succeeded", "info"), true
	case event.OperationFailed:
		return operationMessage(env, e.Result, "failed", "error"), true
	case event.OperationCanceled:
		return operationMessage(env, e.Result, "canceled", "warn"), true
	case event.TaskSkipped:
		return Message{
			Title:  "Task skipped",
			Body:   SanitizeText(fmt.Sprintf("task %s profile %s: %s", e.TaskID, env.ProfileID, e.Reason)),
			Event:  env.Type,
			Level:  "warn",
			Fields: map[string]string{"task_id": e.TaskID, "profile_id": env.ProfileID},
		}, true
	default:
		return Message{}, false
	}
}

func operationMessage(env event.Envelope, res operation.Result, label, level string) Message {
	errText := SanitizeError(res.Error)
	body := fmt.Sprintf("operation %s (%s) %s for profile %s", env.OperationID, env.Kind, label, env.ProfileID)
	if errText != "" {
		body += ": " + errText
	}
	return Message{
		Title: fmt.Sprintf("Operation %s", label),
		Body:  SanitizeText(body),
		Event: env.Type,
		Level: level,
		Fields: map[string]string{
			"operation_id": env.OperationID,
			"kind":         string(env.Kind),
			"profile_id":   env.ProfileID,
			"status":       string(res.Status),
		},
	}
}

func TestMessage() Message {
	return Message{
		Title: "DBack test notification",
		Body:  SanitizeText("This is a test message from dback notify."),
		Event: event.TypeOperationCompleted,
		Level: "info",
	}
}

func FormatPlain(msg Message) string {
	var b strings.Builder
	if msg.Title != "" {
		b.WriteString(msg.Title)
		b.WriteString("\n")
	}
	if msg.Body != "" {
		b.WriteString(msg.Body)
	}
	return b.String()
}
