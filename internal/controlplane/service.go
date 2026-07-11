package controlplane

import (
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
}

func NewService(application *app.App, queueCapacity, workers int) *Service {
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
	}
}
