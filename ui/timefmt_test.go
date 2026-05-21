package ui

import (
	"testing"
	"time"
)

func TestFormatElapsed(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{500 * time.Millisecond, "0s"},
		{5 * time.Second, "5s"},
		{90 * time.Second, "1m 30s"},
		{5 * time.Minute, "5m"},
		{2*time.Hour + 15*time.Minute, "2h 15m"},
	}
	for _, tc := range tests {
		if got := formatElapsed(tc.d); got != tc.want {
			t.Fatalf("formatElapsed(%v) = %q, want %q", tc.d, got, tc.want)
		}
	}
}

func TestFormatRelativeTime(t *testing.T) {
	now := time.Now()
	tests := []struct {
		at   time.Time
		want string
	}{
		{now.Add(-3 * time.Second), "just now"},
		{now.Add(-30 * time.Second), "30 seconds ago"},
		{now.Add(-2 * time.Hour), "2 hours ago"},
		{now.Add(-7 * 24 * time.Hour), "1 week ago"},
	}
	for _, tc := range tests {
		if got := formatRelativeTime(tc.at); got != tc.want {
			t.Fatalf("formatRelativeTime(%v) = %q, want %q", tc.at, got, tc.want)
		}
	}
}

func TestJobStatusLineIncludesElapsed(t *testing.T) {
	job := &operationJob{
		Status:    "Streaming backup 10.0%",
		StartedAt: time.Now().Add(-5 * time.Second),
	}
	line := jobStatusLine(job)
	if line == job.Status {
		t.Fatalf("expected elapsed suffix, got %q", line)
	}
}
