package app

import (
	"testing"
	"time"

	"dback/models"
)

func TestSortLogsNewestFirst(t *testing.T) {
	now := time.Now()
	logs := []models.LogEntry{
		{ID: "1", Timestamp: now.Add(-2 * time.Hour)},
		{ID: "2", Timestamp: now},
		{ID: "3", Timestamp: now.Add(-time.Hour)},
	}
	sortLogsNewestFirst(logs)
	if logs[0].ID != "2" || logs[1].ID != "3" || logs[2].ID != "1" {
		t.Fatalf("unexpected order: %#v", logs)
	}
}

func TestSortHistoryNewestFirst(t *testing.T) {
	now := time.Now()
	history := []models.ExportRecord{
		{ID: "a", ExportDate: now.Add(-48 * time.Hour)},
		{ID: "b", ExportDate: now},
		{ID: "c", ExportDate: now.Add(-24 * time.Hour)},
	}
	sortHistoryNewestFirst(history)
	if history[0].ID != "b" || history[1].ID != "c" || history[2].ID != "a" {
		t.Fatalf("unexpected order: %#v", history)
	}
}
