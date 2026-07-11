package event

import (
	"time"

	"dback/internal/operation"
)

type Type string

const (
	TypeOperationStarted   Type = "operation.started"
	TypeOperationProgress  Type = "operation.progress"
	TypeOperationCompleted Type = "operation.completed"
	TypeOperationFailed    Type = "operation.failed"
	TypeOperationCanceled  Type = "operation.canceled"
	TypeTaskSkipped        Type = "task.skipped"
)

type Envelope struct {
	ID          string
	Type        Type
	OperationID string
	Kind        operation.Kind
	ProfileID   string
	Timestamp   time.Time
}

type Event interface {
	EventType() Type
	Metadata() Envelope
}

type OperationStarted struct {
	Envelope
	Spec operation.Spec
}

func (e OperationStarted) EventType() Type { return TypeOperationStarted }

func (e OperationStarted) Metadata() Envelope { return e.Envelope }

type OperationProgress struct {
	Envelope
	Phase   string
	Current int64
	Total   int64
	Message string
}

func (e OperationProgress) EventType() Type { return TypeOperationProgress }

func (e OperationProgress) Metadata() Envelope { return e.Envelope }

type OperationCompleted struct {
	Envelope
	Result operation.Result
}

func (e OperationCompleted) EventType() Type { return TypeOperationCompleted }

func (e OperationCompleted) Metadata() Envelope { return e.Envelope }

type OperationFailed struct {
	Envelope
	Result operation.Result
	Cause  error
}

func (e OperationFailed) EventType() Type { return TypeOperationFailed }

func (e OperationFailed) Metadata() Envelope { return e.Envelope }

type OperationCanceled struct {
	Envelope
	Result operation.Result
}

func (e OperationCanceled) EventType() Type { return TypeOperationCanceled }

func (e OperationCanceled) Metadata() Envelope { return e.Envelope }

type TaskSkipped struct {
	Envelope
	TaskID string
	Reason string
}

func (e TaskSkipped) EventType() Type { return TypeTaskSkipped }

func (e TaskSkipped) Metadata() Envelope { return e.Envelope }
