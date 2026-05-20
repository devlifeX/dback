package ui

import (
	"context"
	"fmt"
	"time"
)

type operationJob struct {
	ID          string
	Kind        string
	ProfileName string
	Status      string
	Progress    float64
	Done        bool
	Err         string
	Cancel      context.CancelFunc
}

func (u *UI) addJob(kind, profileName string, cancel context.CancelFunc) *operationJob {
	job := &operationJob{
		ID:          fmt.Sprintf("%d", time.Now().UnixNano()),
		Kind:        kind,
		ProfileName: profileName,
		Status:      "Starting...",
		Cancel:      cancel,
	}
	u.jobsMu.Lock()
	u.jobs = append([]*operationJob{job}, u.jobs...)
	u.jobsMu.Unlock()
	u.invalidate()
	return job
}

func (u *UI) updateJob(id, status string, progress float64, errText string) {
	u.jobsMu.Lock()
	for _, job := range u.jobs {
		if job.ID == id {
			job.Status = status
			job.Progress = progress
			if errText != "" {
				job.Err = truncateError(errText, maxErrorMessageLen)
			}
			break
		}
	}
	u.jobsMu.Unlock()
	u.requestBackupsRefresh(false)
}

func (u *UI) finishJob(id, status string, err error) {
	u.jobsMu.Lock()
	for _, job := range u.jobs {
		if job.ID == id {
			job.Done = true
			job.Status = status
			job.Progress = 1
			if err != nil {
				job.Err = sanitizeError(err)
				job.Progress = 0
			}
			break
		}
	}
	u.jobsMu.Unlock()
	u.requestBackupsRefresh(true)
}

func (u *UI) currentJobs() []*operationJob {
	u.jobsMu.Lock()
	defer u.jobsMu.Unlock()
	jobs := make([]*operationJob, len(u.jobs))
	copy(jobs, u.jobs)
	return jobs
}

func (u *UI) requestBackupsRefresh(force bool) {
	if u.section != SectionBackups {
		return
	}
	u.jobsUIMu.Lock()
	now := time.Now()
	if !force && now.Sub(u.lastBackupsRefresh) < 300*time.Millisecond {
		u.jobsUIMu.Unlock()
		return
	}
	u.lastBackupsRefresh = now
	u.jobsUIMu.Unlock()
	u.invalidate()
}
