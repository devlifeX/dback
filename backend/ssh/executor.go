package ssh

import (
	"io"

	"dback/models"
)

// Executor runs shell commands locally or over SSH.
type Executor interface {
	RunCommandStream(cmd string) (io.Reader, io.Reader, Session, error)
	RunCommandPipeInput(cmd string) (io.WriteCloser, io.Reader, Session, error)
	RunCommand(cmd string) (string, error)
	Close() error
}

// NewExecutor returns a command executor for the profile connection type.
func NewExecutor(p models.Profile) (Executor, error) {
	if p.ConnectionType == models.ConnectionTypeLocalhost {
		return &LocalClient{}, nil
	}
	return NewClient(p)
}
