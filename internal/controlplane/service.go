package controlplane

import (
	"context"

	"dback/internal/app"
	"dback/internal/event"
)

type Service struct {
	App        *app.App
	Bus        event.Bus
	Registry   *Registry
	Locks      *LockTable
	Records    *RecordStore
	Engine     *Engine
	Dispatcher *Dispatcher
	opCtx      context.Context
	opCancel   context.CancelFunc
}

func NewService(application *app.App, queueCapacity, workers int) *Service {
	opCtx, opCancel := context.WithCancel(context.Background())
	bus := event.NewMemoryBus(64)
	registry := NewRegistry()
	locks := NewLockTable()
	records := NewRecordStore(1000)
	engine := NewEngine(application, bus, registry, locks)
	dispatcher := NewDispatcher(engine, records, queueCapacity, workers)
	RegisterAppHandlers(registry, application)
	return &Service{
		App:        application,
		Bus:        bus,
		Registry:   registry,
		Locks:      locks,
		Records:    records,
		Engine:     engine,
		Dispatcher: dispatcher,
		opCtx:      opCtx,
		opCancel:   opCancel,
	}
}

// OperationContext outlives individual HTTP requests for async queue work.
func (s *Service) OperationContext() context.Context {
	return s.opCtx
}

func (s *Service) StopOperations() {
	if s.opCancel != nil {
		s.opCancel()
	}
}
