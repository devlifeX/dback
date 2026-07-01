package shell

import (
	"fmt"
	"strings"
)

// Command is one argv-safe process invocation.
type Command struct {
	Binary string
	Args   []string
}

// ExecutionMode selects how a connector runs plan steps.
type ExecutionMode int

const (
	// ModeLocalPipe runs all steps locally with Go io.Pipe between them.
	ModeLocalPipe ExecutionMode = iota
	// ModeRemoteToLocalPipe runs step 0 on remote host and remaining steps locally.
	ModeRemoteToLocalPipe
)

// ExecutionPlan describes a safe multi-step archive pipeline.
type ExecutionPlan struct {
	Steps []Command
	Mode  ExecutionMode
}

// SerializePOSIX renders one command for logging or single-step SSH execution.
func SerializePOSIX(cmd Command) (string, error) {
	if strings.TrimSpace(cmd.Binary) == "" {
		return "", fmt.Errorf("empty command binary")
	}
	parts := []string{quotePOSIX(cmd.Binary)}
	for _, arg := range cmd.Args {
		parts = append(parts, quotePOSIX(arg))
	}
	return strings.Join(parts, " "), nil
}

func quotePOSIX(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
