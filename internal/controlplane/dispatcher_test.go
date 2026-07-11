package controlplane_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"dback/internal/controlplane"
	"dback/internal/event"
	"dback/internal/operation"
)

func TestDispatcherSubmitSuccess(t *testing.T) {
	reg := controlplane.NewRegistry()
	reg.Register(operation.KindBackupDB, func(ctx context.Context, spec operation.Spec, publishProgress func(string, int64, int64)) (*operation.Result, error) {
		if publishProgress != nil {
			publishProgress("working", 1, 2)
		}
		return &operation.Result{
			OperationID: spec.ID,
			Kind:        spec.Kind,
			Status:      operation.StatusSucceeded,
		}, nil
	})
	engine := controlplane.NewEngine(nil, event.NewMemoryBus(8), reg, controlplane.NewLockTable())
	records := controlplane.NewRecordStore(10)
	d := controlplane.NewDispatcher(engine, records, 8, 2)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = d.Shutdown(ctx)
	}()

	spec := operation.Spec{
		Kind:       operation.KindBackupDB,
		ProfileID:  "p1",
		TriggerRef: "test",
		Params:     operation.BackupDBParams{},
	}
	rec, err := d.Submit(context.Background(), spec)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if rec.Status != operation.StatusSucceeded {
		t.Fatalf("status = %q, want succeeded", rec.Status)
	}
	if rec.ID == "" {
		t.Fatal("expected generated operation id")
	}
}

func TestDispatcherOverlapSkip(t *testing.T) {
	reg := controlplane.NewRegistry()
	started := make(chan struct{}, 1)
	reg.Register(operation.KindBackupDB, func(ctx context.Context, spec operation.Spec, publishProgress func(string, int64, int64)) (*operation.Result, error) {
		select {
		case started <- struct{}{}:
		default:
		}
		<-ctx.Done()
		return nil, ctx.Err()
	})
	engine := controlplane.NewEngine(nil, event.NewMemoryBus(8), reg, controlplane.NewLockTable())
	records := controlplane.NewRecordStore(10)
	d := controlplane.NewDispatcher(engine, records, 8, 2)
	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	defer func() {
		cancel1()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = d.Shutdown(ctx)
	}()

	spec := operation.Spec{
		Kind:       operation.KindBackupDB,
		ProfileID:  "p1",
		TriggerRef: "test",
		Params:     operation.BackupDBParams{},
	}
	go func() {
		_, _ = d.Submit(ctx1, spec)
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("first operation did not start")
	}

	spec2 := spec
	spec2.ID = "op-2"
	rec, err := d.Submit(context.Background(), spec2)
	if !errors.Is(err, controlplane.ErrOverlapSkip) {
		t.Fatalf("expected overlap skip, got err=%v status=%q", err, rec.Status)
	}
	cancel1()
}

func TestSubmitAsyncOutlivesRequestContext(t *testing.T) {
	reg := controlplane.NewRegistry()
	done := make(chan struct{})
	reg.Register(operation.KindBackupDB, func(ctx context.Context, spec operation.Spec, publishProgress func(string, int64, int64)) (*operation.Result, error) {
		select {
		case <-time.After(50 * time.Millisecond):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		close(done)
		return &operation.Result{
			OperationID: spec.ID,
			Kind:        spec.Kind,
			Status:      operation.StatusSucceeded,
		}, nil
	})
	engine := controlplane.NewEngine(nil, event.NewMemoryBus(8), reg, controlplane.NewLockTable())
	records := controlplane.NewRecordStore(10)
	d := controlplane.NewDispatcher(engine, records, 8, 2)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = d.Shutdown(ctx)
	}()

	_, cancelReq := context.WithCancel(context.Background())
	opCtx, cancelOps := context.WithCancel(context.Background())
	defer cancelOps()
	spec := operation.Spec{
		Kind:       operation.KindBackupDB,
		ProfileID:  "p1",
		TriggerRef: "api",
		Params:     operation.BackupDBParams{},
	}
	rec, err := d.SubmitAsync(opCtx, spec)
	if err != nil {
		t.Fatalf("SubmitAsync: %v", err)
	}
	cancelReq() // HTTP request finished; operation context must stay alive.

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("operation did not finish")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got, err := d.Get(rec.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status == operation.StatusSucceeded {
			return
		}
		if got.Status == operation.StatusCanceled || got.Status == operation.StatusFailed {
			t.Fatalf("status = %q, want succeeded", got.Status)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for succeeded, last status may still be running")
}

func TestChainContextUploadRecordIDs(t *testing.T) {
	var chain controlplane.ChainContext
	chain = chain.WithResult(operation.Result{
		Artifacts: []operation.Artifact{{Type: operation.ArtifactExportRecord, ID: "rec-1"}},
	})
	spec := operation.Spec{
		Kind:      operation.KindUpload,
		ProfileID: "p1",
		Params:    operation.UploadParams{},
	}
	resolved, err := chain.ResolveUploadParams(spec)
	if err != nil {
		t.Fatal(err)
	}
	params := resolved.Params.(operation.UploadParams)
	if len(params.RecordIDs) != 1 || params.RecordIDs[0] != "rec-1" {
		t.Fatalf("record ids = %#v", params.RecordIDs)
	}
}
