package connector

import (
	"context"

	"dback/backend/shell"
)

// Connector executes plans on a target host transport.
type Connector interface {
	Run(ctx context.Context, plan shell.ExecutionPlan) (*shell.StreamResult, error)
	Close() error
}
