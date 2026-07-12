package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"dback/internal/app"
	"dback/internal/event"
	"dback/internal/operation"
)

var (
	ErrOperationNotFound = errors.New("operation not found")
	ErrQueueFull         = errors.New("operation queue is full")
	ErrOverlapSkip       = errors.New("operation skipped due to overlap")
)

type Handler func(ctx context.Context, spec operation.Spec, publishProgress func(message string, current, total int64)) (*operation.Result, error)

type Registry struct {
	mu       sync.RWMutex
	handlers map[operation.Kind]Handler
}

func NewRegistry() *Registry {
	return &Registry{handlers: map[operation.Kind]Handler{}}
}

func (r *Registry) Register(kind operation.Kind, h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[kind] = h
}

func (r *Registry) Handler(kind operation.Kind) (Handler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[kind]
	return h, ok
}

type LockTable struct {
	mu    sync.Mutex
	keys  map[string]struct{}
}

func NewLockTable() *LockTable {
	return &LockTable{keys: map[string]struct{}{}}
}

func lockKey(profileID string, kind operation.Kind) string {
	return profileID + ":" + string(kind)
}

func (t *LockTable) TryAcquire(profileID string, kind operation.Kind) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	key := lockKey(profileID, kind)
	if _, ok := t.keys[key]; ok {
		return false
	}
	t.keys[key] = struct{}{}
	return true
}

func (t *LockTable) Release(profileID string, kind operation.Kind) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.keys, lockKey(profileID, kind))
}

type RecordStore struct {
	mu      sync.RWMutex
	records map[string]*OperationRecord
	order   []string
	cap     int
}

type OperationRecord struct {
	ID          string
	Kind        operation.Kind
	ProfileID   string
	TaskID      string
	TriggerRef  string
	Params      json.RawMessage
	Status      operation.Status
	StartedAt   time.Time
	FinishedAt  time.Time
	Error       string
	Progress    string
	Artifacts   []operation.Artifact
	cancel      context.CancelFunc
}

func NewRecordStore(capacity int) *RecordStore {
	if capacity <= 0 {
		capacity = 1000
	}
	return &RecordStore{
		records: map[string]*OperationRecord{},
		cap:     capacity,
	}
}

func (s *RecordStore) Put(rec *OperationRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.records[rec.ID]; !exists {
		s.order = append(s.order, rec.ID)
	}
	s.records[rec.ID] = rec
	for len(s.order) > s.cap {
		oldest := s.order[0]
		s.order = s.order[1:]
		delete(s.records, oldest)
	}
}

func (s *RecordStore) Get(id string) (*OperationRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.records[id]
	return rec, ok
}

func (s *RecordStore) Update(id string, fn func(*OperationRecord)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[id]
	if !ok {
		return false
	}
	fn(rec)
	return true
}

func (s *RecordStore) List(limit int) []*OperationRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 {
		limit = 50
	}
	start := len(s.order) - limit
	if start < 0 {
		start = 0
	}
	out := make([]*OperationRecord, 0, limit)
	for i := len(s.order) - 1; i >= start; i-- {
		if rec, ok := s.records[s.order[i]]; ok {
			cp := *rec
			out = append(out, &cp)
		}
	}
	return out
}

type Engine struct {
	app      *app.App
	bus      event.Bus
	registry *Registry
	locks    *LockTable
}

func NewEngine(application *app.App, bus event.Bus, registry *Registry, locks *LockTable) *Engine {
	return &Engine{
		app:      application,
		bus:      bus,
		registry: registry,
		locks:    locks,
	}
}

func (e *Engine) Execute(ctx context.Context, rec *OperationRecord, spec operation.Spec) (*operation.Result, error) {
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	handler, ok := e.registry.Handler(spec.Kind)
	if !ok {
		return nil, fmt.Errorf("no handler registered for kind %q", spec.Kind)
	}
	if !e.locks.TryAcquire(spec.ProfileID, spec.Kind) {
		return nil, ErrOverlapSkip
	}
	defer e.locks.Release(spec.ProfileID, spec.Kind)

	started := time.Now()
	env := event.Envelope{
		ID:          newEventID(),
		Type:        event.TypeOperationStarted,
		OperationID: rec.ID,
		Kind:        spec.Kind,
		ProfileID:   spec.ProfileID,
		TaskID:      spec.TaskID,
		Timestamp:   started,
	}
	_ = e.bus.Publish(ctx, event.OperationStarted{Envelope: env, Spec: spec})

	publishProgress := func(message string, current, total int64) {
		_ = e.bus.Publish(ctx, event.OperationProgress{
			Envelope: operationEnvelope(event.TypeOperationProgress, rec, spec, time.Now()),
			Phase:   "progress",
			Current: current,
			Total:   total,
			Message: message,
		})
	}

	result, err := handler(ctx, spec, publishProgress)
	finished := time.Now()
	if result == nil {
		result = &operation.Result{
			OperationID: rec.ID,
			Kind:        spec.Kind,
			StartedAt:   started,
			FinishedAt:  finished,
		}
	}
	result.OperationID = rec.ID
	result.StartedAt = started
	result.FinishedAt = finished

	if err != nil {
		if errors.Is(err, context.Canceled) {
			result.Status = operation.StatusCanceled
			result.Error = "canceled"
			_ = e.bus.Publish(ctx, event.OperationCanceled{
				Envelope: operationEnvelope(event.TypeOperationCanceled, rec, spec, finished),
				Result: *result,
			})
			return result, err
		}
		if errors.Is(err, ErrOverlapSkip) {
			result.Status = operation.StatusSkipped
			result.Error = err.Error()
			return result, err
		}
		result.Status = operation.StatusFailed
		if result.Error == "" {
			result.Error = err.Error()
		}
		_ = e.bus.Publish(ctx, event.OperationFailed{
			Envelope: operationEnvelope(event.TypeOperationFailed, rec, spec, finished),
			Result: *result,
			Cause:  err,
		})
		return result, err
	}

	result.Status = operation.StatusSucceeded
	_ = e.bus.Publish(ctx, event.OperationCompleted{
		Envelope: operationEnvelope(event.TypeOperationCompleted, rec, spec, finished),
		Result: *result,
	})
	return result, err
}

func operationEnvelope(typ event.Type, rec *OperationRecord, spec operation.Spec, ts time.Time) event.Envelope {
	return event.Envelope{
		ID:          newEventID(),
		Type:        typ,
		OperationID: rec.ID,
		Kind:        spec.Kind,
		ProfileID:   spec.ProfileID,
		TaskID:      spec.TaskID,
		Timestamp:   ts,
	}
}

func newEventID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
