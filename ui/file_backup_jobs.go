package ui

import (
	"context"
	"fmt"
	"time"

	coreapp "dback/internal/app"
	"dback/models"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

type jobSubItem struct {
	Name     string
	Status   string
	Progress float64
	RecordID string
	Err      string
}

func (u *UI) addFileBackupJob(p models.Profile, cancel context.CancelFunc) *operationJob {
	subItems := make([]jobSubItem, len(p.FileBackupPaths))
	for i, path := range p.FileBackupPaths {
		subItems[i] = jobSubItem{Name: path.Name, Status: "pending"}
	}
	job := &operationJob{
		ID:          fmt.Sprintf("%d", time.Now().UnixNano()),
		Kind:        "Backup Files",
		ProfileName: p.Name,
		Status:      "Starting...",
		StartedAt:   time.Now(),
		PathTotal:   len(p.FileBackupPaths),
		SubItems:    subItems,
		Cancel:      cancel,
	}
	u.jobsMu.Lock()
	u.jobs = append([]*operationJob{job}, u.jobs...)
	u.jobsMu.Unlock()
	u.ensureJobTicker()
	u.invalidate()
	return job
}

func (u *UI) setFileBackupJobProgress(id string, prog coreapp.FileBackupProgress) {
	u.jobsMu.Lock()
	for _, job := range u.jobs {
		if job.ID != id {
			continue
		}
		job.PathIndex = prog.PathIndex
		job.PathTotal = prog.PathTotal
		if prog.PathIndex > 0 && prog.PathIndex <= len(job.SubItems) {
			idx := prog.PathIndex - 1
			if idx > 0 {
				job.SubItems[idx-1].Status = "done"
				job.SubItems[idx-1].Progress = 1
			}
			cur := &job.SubItems[idx]
			cur.Status = "running"
			if prog.BytesTotal > 0 {
				cur.Progress = float64(prog.BytesDone) / float64(prog.BytesTotal)
			}
		}
		status := fmt.Sprintf("%d / %d · %s", prog.PathIndex, prog.PathTotal, prog.PathName)
		if prog.BytesDone > 0 {
			status += fmt.Sprintf(" · %s", formatBytes(prog.BytesDone))
			if prog.SpeedBps > 0 {
				status += fmt.Sprintf(" · %s/s", formatBytes(prog.SpeedBps))
			}
		}
		job.Status = status
		if prog.PathTotal > 0 {
			base := float64(prog.PathIndex-1) / float64(prog.PathTotal)
			step := 1.0 / float64(prog.PathTotal)
			if prog.BytesTotal > 0 {
				job.Progress = base + step*float64(prog.BytesDone)/float64(prog.BytesTotal)
			} else {
				job.Progress = base + step*0.5
			}
		}
		break
	}
	u.jobsMu.Unlock()
	u.requestBackupsRefresh(false)
}

func (u *UI) setFileBackupJobRecord(id, recordID string) {
	u.jobsMu.Lock()
	for _, job := range u.jobs {
		if job.ID != id {
			continue
		}
		job.RecordID = recordID
		if job.PathIndex > 0 && job.PathIndex <= len(job.SubItems) {
			job.SubItems[job.PathIndex-1].RecordID = recordID
			job.SubItems[job.PathIndex-1].Status = "done"
			job.SubItems[job.PathIndex-1].Progress = 1
		}
		break
	}
	u.jobsMu.Unlock()
}

func (u *UI) finishFileBackupJob(id, status string, err error) {
	u.jobsMu.Lock()
	for _, job := range u.jobs {
		if job.ID != id {
			continue
		}
		job.Done = true
		job.FinishedAt = time.Now()
		job.Status = status
		if err == nil {
			job.Progress = 1
			for i := range job.SubItems {
				if job.SubItems[i].Status == "pending" || job.SubItems[i].Status == "running" {
					job.SubItems[i].Status = "done"
					job.SubItems[i].Progress = 1
				}
			}
		} else {
			job.Err = sanitizeError(err)
			if job.PathIndex > 0 && job.PathIndex <= len(job.SubItems) {
				job.SubItems[job.PathIndex-1].Status = "failed"
				job.SubItems[job.PathIndex-1].Err = job.Err
			}
		}
		break
	}
	u.jobsMu.Unlock()
	u.requestBackupsRefresh(true)
	u.invalidateBackupCache()
}

func formatBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	}
	if n < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(n)/1024/1024)
	}
	return fmt.Sprintf("%.1f GB", float64(n)/1024/1024/1024)
}

func layoutJobsSubItems(gtx layout.Context, th *material.Theme, theme *AppTheme, items []jobSubItem) layout.Dimensions {
	var rows []layout.FlexChild
	for _, item := range items {
		item := item
		label := item.Name
		switch item.Status {
		case "done":
			label = "✓ " + label
		case "failed":
			label = "✗ " + label
		case "running":
			label = "● " + label
		default:
			label = "○ " + label
		}
		rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(16), Top: unit.Dp(2), Bottom: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Caption(th, label)
				lbl.Color = theme.TextMuted
				return lbl.Layout(gtx)
			})
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
}
