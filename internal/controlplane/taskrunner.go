package controlplane

import (
	"context"
	"fmt"
	"sync"
	"time"

	"dback/internal/app"
	"dback/internal/event"
	"dback/internal/operation"
	"dback/models"

	"github.com/google/uuid"
)

type TaskRunner struct {
	app        *app.App
	dispatcher *Dispatcher
	bus        event.Bus
}

func NewTaskRunner(application *app.App, dispatcher *Dispatcher, bus event.Bus) *TaskRunner {
	return &TaskRunner{
		app:        application,
		dispatcher: dispatcher,
		bus:        bus,
	}
}

func (r *TaskRunner) RunTask(ctx context.Context, task models.Task, profileID, triggerRef string) (models.TaskRunRecord, error) {
	run := models.TaskRunRecord{
		ID:          uuid.NewString(),
		TaskID:      task.ID,
		ProfileID:   profileID,
		TriggerType: task.Trigger.Type,
		StartedAt:   time.Now(),
		Status:      string(operation.StatusRunning),
	}
	specs, err := r.specsForTask(task, profileID, triggerRef)
	if err != nil {
		run.Status = string(operation.StatusFailed)
		run.FinishedAt = time.Now()
		run.ActionResults = []models.ActionResult{{
			Kind:   string(task.Actions[0].Operation),
			Status: string(operation.StatusFailed),
			Error:  err.Error(),
		}}
		return run, err
	}

	chain, err := r.dispatcher.RunChain(ctx, specs)
	run.FinishedAt = time.Now()
	if chain != nil {
		for _, res := range chain.Results {
			run.ActionResults = append(run.ActionResults, models.ActionResult{
				OperationID: res.OperationID,
				Kind:        string(res.Kind),
				Status:      string(res.Status),
				Error:       res.Error,
			})
		}
		run.Status = string(chain.Status)
		if chain.Error != "" && run.Status == string(operation.StatusFailed) {
			err = fmt.Errorf("%s", chain.Error)
		}
	}
	if err != nil && run.Status == string(operation.StatusRunning) {
		run.Status = string(operation.StatusFailed)
	}
	return run, err
}

func (r *TaskRunner) RunTaskAllProfiles(ctx context.Context, task models.Task, triggerRef string) ([]models.TaskRunRecord, error) {
	max := task.EffectiveMaxConcurrentProfiles()
	sem := make(chan struct{}, max)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var runs []models.TaskRunRecord
	var firstErr error

	for _, profileID := range task.ProfileIDs {
		wg.Add(1)
		go func(pid string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			run, err := r.RunTask(ctx, task, pid, triggerRef)
			mu.Lock()
			runs = append(runs, run)
			if err != nil && firstErr == nil {
				firstErr = err
			}
			mu.Unlock()
		}(profileID)
	}
	wg.Wait()
	return runs, firstErr
}

func (r *TaskRunner) ValidateForRun(task models.Task) error {
	if err := r.app.ValidateTask(task); err != nil {
		return err
	}
	for _, profileID := range task.ProfileIDs {
		if !r.app.ProfileExists(profileID) {
			return fmt.Errorf("profile %q not found", profileID)
		}
	}
	return nil
}

func (r *TaskRunner) PublishSkipped(ctx context.Context, taskID, profileID, reason string) {
	_ = r.bus.Publish(ctx, event.TaskSkipped{
		Envelope: event.Envelope{
			ID:        uuid.NewString(),
			Type:      event.TypeTaskSkipped,
			ProfileID: profileID,
			Timestamp: time.Now(),
		},
		TaskID: taskID,
		Reason: reason,
	})
}

func (r *TaskRunner) specsForTask(task models.Task, profileID, triggerRef string) ([]operation.Spec, error) {
	specs := make([]operation.Spec, 0, len(task.Actions))
	for _, action := range task.Actions {
		spec, err := operation.SpecFromAction(profileID, triggerRef, operation.Kind(action.Operation), action.Params)
		if err != nil {
			return nil, err
		}
		specs = append(specs, spec)
	}
	return specs, nil
}
