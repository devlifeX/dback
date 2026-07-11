package trigger

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"dback/internal/operation"
	"dback/models"
)

type memStore struct {
	mu    sync.Mutex
	tasks map[string]models.Task
	runs  []models.TaskRunRecord
}

func (m *memStore) ListTasks() ([]models.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]models.Task, 0, len(m.tasks))
	for _, t := range m.tasks {
		out = append(out, t)
	}
	return out, nil
}

func (m *memStore) UpdateTaskState(id string, fn func(*models.Task) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[id]
	if !ok {
		return errNotFound
	}
	if err := fn(&t); err != nil {
		return err
	}
	m.tasks[id] = t
	return nil
}

func (m *memStore) AppendTaskRun(run models.TaskRunRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runs = append(m.runs, run)
	return nil
}

func (m *memStore) ValidateTask(task models.Task) error {
	return task.Trigger.Validate()
}

type fakeRunner struct {
	mu    sync.Mutex
	calls int
	block chan struct{}
}

func (f *fakeRunner) RunTaskAllProfiles(ctx context.Context, task models.Task, triggerRef string) ([]models.TaskRunRecord, error) {
	if f.block != nil {
		<-f.block
	}
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
	return []models.TaskRunRecord{{
		ID:        "run-1",
		TaskID:    task.ID,
		ProfileID: task.ProfileIDs[0],
		Status:    string(operation.StatusSucceeded),
	}}, nil
}

func (f *fakeRunner) ValidateForRun(task models.Task) error { return nil }

func (f *fakeRunner) PublishSkipped(ctx context.Context, taskID, profileID, reason string) {}

var errNotFound = errors.New("not found")

func TestRegistryDoubleFireOverlap(t *testing.T) {
	store := &memStore{tasks: map[string]models.Task{
		"t1": {
			ID:         "t1",
			Enabled:    true,
			ProfileIDs: []string{"p1"},
			Trigger: models.TriggerSpec{
				Type:     models.TriggerInterval,
				Interval: &models.IntervalTrigger{Every: time.Minute},
			},
		},
	}}
	runner := &fakeRunner{block: make(chan struct{})}
	reg := NewRegistry(store, runner, NewFakeClock(time.Now()), time.Minute)
	ctx := context.Background()
	done := make(chan struct{})
	go func() {
		_ = reg.fire(ctx, store.tasks["t1"], "test")
		close(done)
	}()
	time.Sleep(20 * time.Millisecond)
	err := reg.fire(ctx, store.tasks["t1"], "test")
	if err != nil {
		t.Fatalf("second fire: %v", err)
	}
	close(runner.block)
	<-done
	runner.mu.Lock()
	calls := runner.calls
	runner.mu.Unlock()
	if calls != 1 {
		t.Fatalf("expected 1 run due to overlap skip, got %d", calls)
	}
}

func TestRegistryResyncNoCatchUp(t *testing.T) {
	store := &memStore{tasks: map[string]models.Task{
		"t1": {
			ID:      "t1",
			Enabled: true,
			Trigger: models.TriggerSpec{
				Type: models.TriggerCron,
				Cron: &models.CronTrigger{
					Expr:     "0 2 * * *",
					Timezone: "UTC",
				},
			},
		},
	}}
	reg := NewRegistry(store, &fakeRunner{}, NewFakeClock(time.Date(2026, 7, 11, 10, 0, 0, 0, time.UTC)), time.Minute)
	if err := reg.Resync(context.Background()); err != nil {
		t.Fatal(err)
	}
	task := store.tasks["t1"]
	if task.State.NextRunAt.IsZero() {
		t.Fatal("expected next run to be set on resync")
	}
	if !task.State.NextRunAt.After(time.Date(2026, 7, 11, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("next run should be in the future, got %v", task.State.NextRunAt)
	}
}
