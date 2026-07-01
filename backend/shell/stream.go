package shell

import (
	"io"
)

// StreamResult is a running command pipeline with a waitable exit status.
type StreamResult struct {
	Reader io.ReadCloser
	Stderr io.Reader
	Wait   func() error
}
