package ui

import (
	"fmt"
	"time"
)

func formatElapsed(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	if d < time.Second {
		return "0s"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		if s == 0 {
			return fmt.Sprintf("%dm", m)
		}
		return fmt.Sprintf("%dm %ds", m, s)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if m == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}

func formatRelativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		s := int(d.Seconds())
		if s <= 5 {
			return "just now"
		}
		if s == 1 {
			return "1 second ago"
		}
		return fmt.Sprintf("%d seconds ago", s)
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	case d < 7*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	case d < 30*24*time.Hour:
		weeks := int(d.Hours() / (24 * 7))
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	case d < 365*24*time.Hour:
		months := int(d.Hours() / (24 * 30))
		if months < 1 {
			months = 1
		}
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	default:
		years := int(d.Hours() / (24 * 365))
		if years < 1 {
			years = 1
		}
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}

func jobElapsed(job *operationJob) time.Duration {
	if job == nil || job.StartedAt.IsZero() {
		return 0
	}
	end := time.Now()
	if job.Done && !job.FinishedAt.IsZero() {
		end = job.FinishedAt
	}
	return end.Sub(job.StartedAt)
}

func jobStatusLine(job *operationJob) string {
	status := job.Status
	if elapsed := formatElapsed(jobElapsed(job)); elapsed != "" {
		status += " · " + elapsed + " elapsed"
	}
	if job.Err != "" {
		status += " — " + truncateError(job.Err, maxErrorMessageLen)
	}
	return status
}
