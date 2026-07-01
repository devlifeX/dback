package connector

import (
	"context"
	"fmt"
	"io"
	"sync"

	"dback/backend/shell"
)

type localConnector struct{}

func newLocalConnector() Connector {
	return &localConnector{}
}

func (c *localConnector) Close() error { return nil }

func (c *localConnector) Run(ctx context.Context, plan shell.ExecutionPlan) (*shell.StreamResult, error) {
	if plan.Mode != shell.ModeLocalPipe {
		return nil, fmt.Errorf("local connector requires local pipe mode")
	}
	if len(plan.Steps) == 0 {
		return nil, fmt.Errorf("execution plan has no steps")
	}
	return runLocalPipe(ctx, plan.Steps)
}

func runLocalPipe(ctx context.Context, steps []shell.Command) (*shell.StreamResult, error) {
	stdout, stderr, waitAll, err := startLocalPipeline(ctx, steps, nil)
	if err != nil {
		return nil, err
	}
	return &shell.StreamResult{
		Reader: readCloser{Reader: stdout, closeFn: func() error {
			_ = stdout.Close()
			return waitAll()
		}},
		Stderr: stderr,
		Wait:   waitAll,
	}, nil
}

type readCloser struct {
	io.Reader
	closeFn func() error
}

func (r readCloser) Close() error {
	if r.closeFn != nil {
		return r.closeFn()
	}
	if c, ok := r.Reader.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func onceWait(fn func() error) func() error {
	var once sync.Once
	var err error
	return func() error {
		once.Do(func() {
			err = fn()
		})
		return err
	}
}
