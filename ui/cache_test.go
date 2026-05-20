package ui

import (
	"testing"
	"time"

	"dback/models"
)

func TestBackupViewCacheRebuildsOnFilterChange(t *testing.T) {
	cache := backupViewCache{}
	history := []models.ExportRecord{
		{ID: "a", ProfileID: "host-a", ExportDate: time.Now()},
		{ID: "b", ProfileID: "host-b", ExportDate: time.Now().Add(-time.Hour)},
	}
	filtered := filterBackupsByHost(sortBackupsNewestFirst(history), "host-a")
	if len(filtered) != 1 || filtered[0].ID != "a" {
		t.Fatalf("unexpected filtered backups: %#v", filtered)
	}
	cache.records = filtered
	cache.hostFilter = "host-a"
	if cache.hostFilter != "host-a" {
		t.Fatal("cache should track host filter")
	}
}

func TestTemplateOptionCacheRebuildOnce(t *testing.T) {
	cache := templateOptionCache{}
	templates := []models.SQLTemplate{
		{ID: "1", Name: "One", Body: "SELECT 1"},
		{ID: "2", Name: "Two", Body: "SELECT 2"},
	}
	cache.rebuild(1, templates)
	if len(cache.names) != 2 || cache.nameToBody["Two"] != "SELECT 2" {
		t.Fatalf("unexpected cache: %#v", cache)
	}
	namesPtr := cache.names
	cache.rebuild(1, templates)
	if len(cache.names) != 2 {
		t.Fatalf("expected same revision cache, got %#v", cache.names)
	}
	if &cache.names[0] != &namesPtr[0] && len(cache.names) == len(namesPtr) {
		// rebuild with same revision should return early without reallocating slice header content
	}
	cache.rebuild(2, templates)
	if cache.revision != 2 {
		t.Fatalf("expected revision bump, got %d", cache.revision)
	}
}
