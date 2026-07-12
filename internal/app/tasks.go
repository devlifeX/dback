package app

import (
	"errors"
	"fmt"

	"dback/internal/operation"
	"dback/internal/store"
	"dback/models"
)

var ErrInvalidTask = errors.New("invalid task")

func (a *App) ListTasks() ([]models.Task, error) {
	return a.store.ListTasks()
}

func (a *App) GetTask(id string) (models.Task, error) {
	return a.store.GetTask(id)
}

func (a *App) SaveTask(task models.Task) error {
	if err := a.ValidateTask(task); err != nil {
		return err
	}
	return a.store.SaveTask(task)
}

func (a *App) SetTaskEnabled(id string, enabled bool) error {
	return a.store.SetTaskEnabled(id, enabled)
}

func (a *App) DeleteTask(id string) error {
	return a.store.DeleteTask(id)
}

func (a *App) ListTaskRuns(taskID string, limit int) ([]models.TaskRunRecord, error) {
	return a.store.ListTaskRuns(taskID, limit)
}

func (a *App) AppendTaskRun(run models.TaskRunRecord) error {
	return a.store.AppendTaskRun(run)
}

func (a *App) UpdateTaskState(id string, fn func(*models.Task) error) error {
	return a.store.UpdateTaskState(id, fn)
}

func (a *App) ProfileExists(id string) bool {
	for _, p := range a.Profiles() {
		if p.ID == id {
			return true
		}
	}
	return false
}

func (a *App) ValidateTask(task models.Task) error {
	if task.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidTask)
	}
	if len(task.ProfileIDs) == 0 {
		return fmt.Errorf("%w: at least one profile id is required", ErrInvalidTask)
	}
	if len(task.Actions) == 0 {
		return fmt.Errorf("%w: at least one action is required", ErrInvalidTask)
	}
	if err := task.Trigger.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidTask, err)
	}
	hasBackupFiles := task.HasAction(string(operation.KindBackupFiles))
	for _, profileID := range task.ProfileIDs {
		if !a.ProfileExists(profileID) {
			return fmt.Errorf("%w: profile %q not found", ErrInvalidTask, profileID)
		}
		if hasBackupFiles {
			p, err := a.profileByID(profileID)
			if err != nil {
				return fmt.Errorf("%w: %v", ErrInvalidTask, err)
			}
			if p.UsesWordPress() {
				return fmt.Errorf("%w: backup_files is incompatible with WordPress profile %q", ErrInvalidTask, profileID)
			}
		}
	}
	for _, action := range task.Actions {
		kind := operation.Kind(action.Operation)
		switch kind {
		case operation.KindBackupDB, operation.KindBackupFiles, operation.KindUpload, operation.KindUrlChecker:
		default:
			return fmt.Errorf("%w: unsupported action %q", ErrInvalidTask, action.Operation)
		}
		params, err := operation.DecodeParams(kind, action.Params)
		if err != nil {
			return fmt.Errorf("%w: action %q: %v", ErrInvalidTask, action.Operation, err)
		}
		if err := params.Validate(); err != nil {
			return fmt.Errorf("%w: action %q: %v", ErrInvalidTask, action.Operation, err)
		}
	}
	if task.EffectiveOverlapPolicy() != models.OverlapPolicySkip && task.OverlapPolicy != "" {
		return fmt.Errorf("%w: only skip overlap policy is supported in v1", ErrInvalidTask)
	}
	return nil
}

func IsTaskNotFound(err error) bool {
	return errors.Is(err, store.ErrTaskNotFound)
}
