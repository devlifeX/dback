package store

import (
	"errors"
	"fmt"

	"dback/models"

	"github.com/google/uuid"
)

const maxTaskRuns = 500

var (
	ErrTaskNotFound = errors.New("task not found")
)

func (s *Store) ListTasks() ([]models.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return nil, err
	}
	return append([]models.Task(nil), s.tasks...), nil
}

func (s *Store) GetTask(id string) (models.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return models.Task{}, err
	}
	for _, t := range s.tasks {
		if t.ID == id {
			return t, nil
		}
	}
	return models.Task{}, ErrTaskNotFound
}

func (s *Store) SaveTask(task models.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	if task.ID == "" {
		task.ID = uuid.NewString()
	}
	for i, existing := range s.tasks {
		if existing.ID == task.ID {
			s.tasks[i] = task
			s.bumpRevisionLocked()
			return s.persistVaultLocked()
		}
	}
	s.tasks = append(s.tasks, task)
	s.bumpRevisionLocked()
	return s.persistVaultLocked()
}

func (s *Store) SetTaskEnabled(id string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	for i, t := range s.tasks {
		if t.ID == id {
			s.tasks[i].Enabled = enabled
			s.bumpRevisionLocked()
			return s.persistVaultLocked()
		}
	}
	return ErrTaskNotFound
}

func (s *Store) DeleteTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	for i, t := range s.tasks {
		if t.ID == id {
			s.tasks = append(s.tasks[:i], s.tasks[i+1:]...)
			s.bumpRevisionLocked()
			return s.persistVaultLocked()
		}
	}
	return ErrTaskNotFound
}

func (s *Store) ListTaskRuns(taskID string, limit int) ([]models.TaskRunRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}
	var out []models.TaskRunRecord
	for i := len(s.taskRuns) - 1; i >= 0 && len(out) < limit; i-- {
		if taskID == "" || s.taskRuns[i].TaskID == taskID {
			out = append(out, s.taskRuns[i])
		}
	}
	return out, nil
}

func (s *Store) AppendTaskRun(run models.TaskRunRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	if run.ID == "" {
		run.ID = uuid.NewString()
	}
	s.taskRuns = append(s.taskRuns, run)
	if len(s.taskRuns) > maxTaskRuns {
		s.taskRuns = s.taskRuns[len(s.taskRuns)-maxTaskRuns:]
	}
	s.bumpRevisionLocked()
	return s.persistVaultLocked()
}

func (s *Store) UpdateTaskState(id string, fn func(*models.Task) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	for i := range s.tasks {
		if s.tasks[i].ID != id {
			continue
		}
		if err := fn(&s.tasks[i]); err != nil {
			return err
		}
		s.bumpRevisionLocked()
		return s.persistVaultLocked()
	}
	return fmt.Errorf("%w: %s", ErrTaskNotFound, id)
}
