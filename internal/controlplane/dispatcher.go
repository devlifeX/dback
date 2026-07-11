package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"dback/internal/operation"

	"github.com/google/uuid"
)

type Dispatcher struct {
	engine   *Engine
	records  *RecordStore
	queueCap int
	workers  int

	mu      sync.Mutex
	sem     chan struct{}
	pending chan *submitJob
	wg      sync.WaitGroup
	closed  bool
}

type submitJob struct {
	ctx    context.Context
	rec    *OperationRecord
	spec   operation.Spec
	done   chan submitResult
}

type submitResult struct {
	result *operation.Result
	err    error
}

func NewDispatcher(engine *Engine, records *RecordStore, queueCapacity, workers int) *Dispatcher {
	if queueCapacity <= 0 {
		queueCapacity = 256
	}
	if workers <= 0 {
		workers = 4
	}
	d := &Dispatcher{
		engine:   engine,
		records:  records,
		queueCap: queueCapacity,
		workers:  workers,
		sem:      make(chan struct{}, workers),
		pending:  make(chan *submitJob, queueCapacity),
	}
	for i := 0; i < workers; i++ {
		d.wg.Add(1)
		go d.worker()
	}
	return d
}

func (d *Dispatcher) Submit(ctx context.Context, spec operation.Spec) (*OperationRecord, error) {
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	if spec.ID == "" {
		spec.ID = uuid.NewString()
	}

	rec := &OperationRecord{
		ID:         spec.ID,
		Kind:       spec.Kind,
		ProfileID:  spec.ProfileID,
		TriggerRef: spec.TriggerRef,
		Params:     encodeSpecParams(spec),
		Status:     operation.StatusQueued,
	}
	d.records.Put(rec)

	job := &submitJob{
		ctx:  ctx,
		rec:  rec,
		spec: spec,
		done: make(chan submitResult, 1),
	}

	select {
	case d.pending <- job:
	default:
		d.records.Update(rec.ID, func(r *OperationRecord) {
			r.Status = operation.StatusFailed
			r.Error = ErrQueueFull.Error()
			r.FinishedAt = time.Now()
		})
		return rec, ErrQueueFull
	}

	select {
	case <-ctx.Done():
		return rec, ctx.Err()
	case res := <-job.done:
		return rec, res.err
	}
}

func (d *Dispatcher) SubmitAsync(ctx context.Context, spec operation.Spec) (*OperationRecord, error) {
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	if spec.ID == "" {
		spec.ID = uuid.NewString()
	}
	rec := &OperationRecord{
		ID:         spec.ID,
		Kind:       spec.Kind,
		ProfileID:  spec.ProfileID,
		TriggerRef: spec.TriggerRef,
		Params:     encodeSpecParams(spec),
		Status:     operation.StatusQueued,
	}
	d.records.Put(rec)

	job := &submitJob{
		ctx:  ctx,
		rec:  rec,
		spec: spec,
		done: make(chan submitResult, 1),
	}

	select {
	case d.pending <- job:
		return rec, nil
	default:
		d.records.Update(rec.ID, func(r *OperationRecord) {
			r.Status = operation.StatusFailed
			r.Error = ErrQueueFull.Error()
			r.FinishedAt = time.Now()
		})
		return rec, ErrQueueFull
	}
}

// encodeSpecParams serializes the typed params so a persisted OperationRecord
// can be replayed (e.g. retry) with the original filters/policy intact.
func encodeSpecParams(spec operation.Spec) json.RawMessage {
	if spec.Params == nil {
		return nil
	}
	raw, err := json.Marshal(spec.Params)
	if err != nil {
		return nil
	}
	return raw
}

func (d *Dispatcher) Get(id string) (*OperationRecord, error) {
	rec, ok := d.records.Get(id)
	if !ok {
		return nil, ErrOperationNotFound
	}
	return rec, nil
}

func (d *Dispatcher) Cancel(id string) error {
	rec, ok := d.records.Get(id)
	if !ok {
		return ErrOperationNotFound
	}
	if rec.cancel != nil {
		rec.cancel()
		return nil
	}
	if rec.Status == operation.StatusQueued {
		d.records.Update(id, func(r *OperationRecord) {
			r.Status = operation.StatusCanceled
			r.Error = "canceled before start"
			r.FinishedAt = time.Now()
		})
	}
	return nil
}

func (d *Dispatcher) RunChain(ctx context.Context, specs []operation.Spec) (*operation.ChainResult, error) {
	if len(specs) == 0 {
		return nil, errors.New("empty operation chain")
	}
	chainID := uuid.NewString()
	var results []operation.Result
	var chainCtx ChainContext

	for i, spec := range specs {
		if spec.ID == "" {
			spec.ID = uuid.NewString()
		}
		resolved, err := chainCtx.ResolveUploadParams(spec)
		if err != nil {
			return &operation.ChainResult{
				OperationID: chainID,
				Results:     results,
				Status:      operation.StatusFailed,
				Error:       err.Error(),
			}, err
		}
		rec, err := d.Submit(ctx, resolved)
		if rec != nil {
			if rec.Status == operation.StatusSucceeded || rec.Status == operation.StatusFailed || rec.Status == operation.StatusCanceled || rec.Status == operation.StatusSkipped {
				results = append(results, operation.Result{
					OperationID: rec.ID,
					Kind:        rec.Kind,
					Status:      rec.Status,
					StartedAt:   rec.StartedAt,
					FinishedAt:  rec.FinishedAt,
					Error:       rec.Error,
					Artifacts:   rec.Artifacts,
				})
				if rec.Status == operation.StatusSucceeded {
					chainCtx = chainCtx.WithResult(results[len(results)-1])
				}
			}
		}
		if err != nil {
			if errors.Is(err, ErrOverlapSkip) {
				continue
			}
			if i == len(specs)-1 || !errors.Is(err, context.Canceled) {
				status := operation.StatusFailed
				if errors.Is(err, context.Canceled) {
					status = operation.StatusCanceled
				}
				return &operation.ChainResult{
					OperationID: chainID,
					Results:     results,
					Status:      status,
					Error:       err.Error(),
				}, err
			}
		}
		if rec != nil && rec.Status == operation.StatusFailed {
			return &operation.ChainResult{
				OperationID: chainID,
				Results:     results,
				Status:      operation.StatusFailed,
				Error:       rec.Error,
			}, fmt.Errorf("chain stopped at step %d: %s", i+1, rec.Error)
		}
	}

	status := operation.StatusSucceeded
	for _, r := range results {
		if r.Status != operation.StatusSucceeded && r.Status != operation.StatusSkipped {
			status = r.Status
			break
		}
	}
	return &operation.ChainResult{
		OperationID: chainID,
		Results:     results,
		Status:      status,
	}, nil
}

func (d *Dispatcher) Shutdown(ctx context.Context) error {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil
	}
	d.closed = true
	d.mu.Unlock()
	close(d.pending)
	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (d *Dispatcher) worker() {
	defer d.wg.Done()
	for job := range d.pending {
		d.runJob(job)
	}
}

func (d *Dispatcher) runJob(job *submitJob) {
	d.sem <- struct{}{}
	defer func() { <-d.sem }()

	runCtx, cancel := context.WithCancel(job.ctx)
	d.records.Update(job.rec.ID, func(r *OperationRecord) {
		r.Status = operation.StatusRunning
		r.StartedAt = time.Now()
		r.cancel = cancel
	})

	result, err := d.engine.Execute(runCtx, job.rec, job.spec)

	d.records.Update(job.rec.ID, func(r *OperationRecord) {
		r.cancel = nil
		r.FinishedAt = time.Now()
		if result != nil {
			r.Status = result.Status
			r.Error = result.Error
			r.Artifacts = result.Artifacts
		} else if err != nil {
			r.Status = operation.StatusFailed
			r.Error = err.Error()
		}
	})

	job.done <- submitResult{result: result, err: err}
}
