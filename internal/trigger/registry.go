package trigger

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"dback/internal/app"
	"dback/internal/operation"
	"dback/models"
)

type TaskStore interface {
	ListTasks() ([]models.Task, error)
	UpdateTaskState(id string, fn func(*models.Task) error) error
	AppendTaskRun(run models.TaskRunRecord) error
	ValidateTask(task models.Task) error
}

type Runner interface {
	RunTaskAllProfiles(ctx context.Context, task models.Task, triggerRef string) ([]models.TaskRunRecord, error)
	ValidateForRun(task models.Task) error
	PublishSkipped(ctx context.Context, taskID, profileID, reason string)
}

type Registry struct {
	store    TaskStore
	runner   Runner
	clock    Clock
	interval time.Duration

	mu       sync.Mutex
	tasks    map[string]models.Task
	running  map[string]struct{}
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

func NewRegistry(store TaskStore, runner Runner, clock Clock, pollInterval time.Duration) *Registry {
	if pollInterval <= 0 {
		pollInterval = 10 * time.Second
	}
	if clock == nil {
		clock = RealClock{}
	}
	return &Registry{
		store:    store,
		runner:   runner,
		clock:    clock,
		interval: pollInterval,
		tasks:    map[string]models.Task{},
		running:  map[string]struct{}{},
	}
}

func (r *Registry) Start(ctx context.Context) error {
	if err := r.Resync(ctx); err != nil {
		return err
	}
	runCtx, cancel := context.WithCancel(ctx)
	r.mu.Lock()
	r.cancel = cancel
	r.mu.Unlock()

	r.wg.Add(1)
	go r.loop(runCtx)
	return nil
}

func (r *Registry) Stop(ctx context.Context) error {
	r.mu.Lock()
	cancel := r.cancel
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Registry) Resync(ctx context.Context) error {
	tasks, err := r.store.ListTasks()
	if err != nil {
		return err
	}
	now := r.clock.Now()
	updated := make(map[string]models.Task, len(tasks))
	for _, task := range tasks {
		t := task
		if t.Enabled {
			switch t.Trigger.Type {
			case models.TriggerCron, models.TriggerInterval:
				next, err := InitNextRun(t, now)
				if err != nil {
					return err
				}
				if !next.IsZero() {
					t.State.NextRunAt = next
				}
			case models.TriggerOneShot:
				if t.State.LastFiredAt.IsZero() {
					next, err := InitNextRun(t, now)
					if err != nil {
						return err
					}
					if !next.IsZero() {
						t.State.NextRunAt = next
					}
				}
			}
			if err := r.store.UpdateTaskState(t.ID, func(st *models.Task) error {
				st.State = t.State
				return nil
			}); err != nil {
				return err
			}
		}
		updated[t.ID] = t
	}
	r.mu.Lock()
	r.tasks = updated
	r.mu.Unlock()

	for _, task := range updated {
		if task.Enabled && task.Trigger.Type == models.TriggerOnBoot && !task.State.OnBootFired {
			r.scheduleOnBoot(ctx, task)
		}
	}
	return nil
}

func (r *Registry) FireNow(ctx context.Context, taskID string, profileIDs []string) error {
	task, ok := r.taskByID(taskID)
	if !ok {
		tasks, err := r.store.ListTasks()
		if err != nil {
			return err
		}
		for _, t := range tasks {
			if t.ID == taskID {
				task = t
				ok = true
				break
			}
		}
	}
	if !ok {
		return fmt.Errorf("task %q not found", taskID)
	}
	if len(profileIDs) > 0 {
		task.ProfileIDs = append([]string(nil), profileIDs...)
	}
	return r.fire(ctx, task, "manual")
}

func (r *Registry) taskByID(id string) (models.Task, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tasks[id]
	return t, ok
}

func (r *Registry) loop(ctx context.Context) {
	defer r.wg.Done()
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.tick(ctx)
		}
	}
}

func (r *Registry) tick(ctx context.Context) {
	r.mu.Lock()
	tasks := make([]models.Task, 0, len(r.tasks))
	for _, t := range r.tasks {
		if t.Enabled {
			tasks = append(tasks, t)
		}
	}
	r.mu.Unlock()

	now := r.clock.Now()
	for _, task := range tasks {
		if task.Trigger.Type == models.TriggerOnBoot {
			continue
		}
		if ShouldFire(task, now) {
			if err := r.fire(ctx, task, string(task.Trigger.Type)); err != nil {
				log.Printf("trigger: task %s fire failed: %v", task.ID, err)
			}
		}
	}
}

func (r *Registry) scheduleOnBoot(ctx context.Context, task models.Task) {
	delay := task.Trigger.OnBoot.Delay
	r.wg.Add(1)
	go func(t models.Task) {
		defer r.wg.Done()
		select {
		case <-ctx.Done():
			return
		case <-r.clock.After(delay):
		}
		current, ok := r.taskByID(t.ID)
		if !ok || !current.Enabled || current.State.OnBootFired {
			return
		}
		if err := r.fire(ctx, current, string(models.TriggerOnBoot)); err != nil {
			log.Printf("trigger: on_boot task %s failed: %v", t.ID, err)
		}
	}(task)
}

func (r *Registry) fire(ctx context.Context, task models.Task, triggerRef string) error {
	if task.EffectiveOverlapPolicy() == models.OverlapPolicySkip {
		r.mu.Lock()
		if _, ok := r.running[task.ID]; ok {
			r.mu.Unlock()
			for _, profileID := range task.ProfileIDs {
				r.runner.PublishSkipped(ctx, task.ID, profileID, "task already running")
			}
			return nil
		}
		r.running[task.ID] = struct{}{}
		r.mu.Unlock()
		defer func() {
			r.mu.Lock()
			delete(r.running, task.ID)
			r.mu.Unlock()
		}()
	}

	if err := r.runner.ValidateForRun(task); err != nil {
		for _, profileID := range task.ProfileIDs {
			r.runner.PublishSkipped(ctx, task.ID, profileID, err.Error())
		}
		return err
	}

	runs, err := r.runner.RunTaskAllProfiles(ctx, task, triggerRef)
	for _, run := range runs {
		if appendErr := r.store.AppendTaskRun(run); appendErr != nil {
			log.Printf("trigger: append task run: %v", appendErr)
		}
	}

	status := string(operation.StatusSucceeded)
	if err != nil {
		status = string(operation.StatusFailed)
	}
	for _, run := range runs {
		if run.Status == string(operation.StatusFailed) {
			status = string(operation.StatusFailed)
			break
		}
	}

	now := r.clock.Now()
	disable := task.Trigger.Type == models.TriggerOneShot && status == string(operation.StatusSucceeded)
	next, nextErr := NextRunAfter(task, now)
	if nextErr != nil {
		return nextErr
	}

	updateErr := r.store.UpdateTaskState(task.ID, func(st *models.Task) error {
		st.State.LastFiredAt = now
		st.State.LastRunStatus = status
		if task.Trigger.Type == models.TriggerOnBoot {
			st.State.OnBootFired = true
		}
		if disable {
			st.Enabled = false
			st.State.NextRunAt = time.Time{}
		} else if task.Trigger.Type == models.TriggerOneShot {
			st.State.NextRunAt = time.Time{}
		} else if task.Trigger.Type != models.TriggerOnBoot {
			st.State.NextRunAt = next
		}
		return nil
	})
	if updateErr != nil {
		return updateErr
	}

	r.mu.Lock()
	if st, ok := r.tasks[task.ID]; ok {
		st.State.LastFiredAt = now
		st.State.LastRunStatus = status
		if task.Trigger.Type == models.TriggerOnBoot {
			st.State.OnBootFired = true
		}
		if disable {
			st.Enabled = false
			st.State.NextRunAt = time.Time{}
		} else if task.Trigger.Type == models.TriggerOneShot {
			st.State.NextRunAt = time.Time{}
		} else if task.Trigger.Type != models.TriggerOnBoot {
			st.State.NextRunAt = next
		}
		r.tasks[task.ID] = st
	}
	r.mu.Unlock()
	return err
}

var _ TaskStore = (*appStoreAdapter)(nil)

type appStoreAdapter struct {
	app *app.App
}

func AppStore(application *app.App) TaskStore {
	return &appStoreAdapter{app: application}
}

func (a *appStoreAdapter) ListTasks() ([]models.Task, error) {
	return a.app.ListTasks()
}

func (a *appStoreAdapter) UpdateTaskState(id string, fn func(*models.Task) error) error {
	return a.app.UpdateTaskState(id, fn)
}

func (a *appStoreAdapter) AppendTaskRun(run models.TaskRunRecord) error {
	return a.app.AppendTaskRun(run)
}

func (a *appStoreAdapter) ValidateTask(task models.Task) error {
	return a.app.ValidateTask(task)
}
