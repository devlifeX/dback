package ui

import (
	"testing"
	"time"

	"dback/models"
)

func TestFilterProfilesByGroupAndSearch(t *testing.T) {
	profiles := []models.Profile{
		{ID: "1", Name: "Prod", Group: "Production", Host: "10.0.0.1", TargetDBName: "app"},
		{ID: "2", Name: "Staging", Group: "Default", Host: "10.0.0.2", TargetDBName: "stage"},
		{ID: "3", Name: "Prod Replica", Group: "Production", Host: "10.0.0.3", TargetDBName: "app2"},
	}

	got := filterProfiles(profiles, "", "Production")
	if len(got) != 2 {
		t.Fatalf("expected 2 production hosts, got %d", len(got))
	}

	got = filterProfiles(profiles, "replica", groupFilterAll)
	if len(got) != 1 || got[0].Name != "Prod Replica" {
		t.Fatalf("expected single replica match, got %#v", got)
	}
}

func TestSortBackupsNewestFirst(t *testing.T) {
	now := time.Now()
	records := []models.ExportRecord{
		{ID: "old", ExportDate: now.Add(-2 * time.Hour)},
		{ID: "new", ExportDate: now},
		{ID: "mid", ExportDate: now.Add(-1 * time.Hour)},
	}
	got := sortBackupsNewestFirst(records)
	if got[0].ID != "new" || got[1].ID != "mid" || got[2].ID != "old" {
		t.Fatalf("unexpected order: %#v", got)
	}
}

func TestFilterBackupsByHost(t *testing.T) {
	records := []models.ExportRecord{
		{ID: "a", ProfileID: "host-a"},
		{ID: "b", ProfileID: "host-b"},
	}
	got := filterBackupsByHost(records, "host-a")
	if len(got) != 1 || got[0].ID != "a" {
		t.Fatalf("expected host-a backup only, got %#v", got)
	}
	got = filterBackupsByHost(records, backupFilterAll)
	if len(got) != 2 {
		t.Fatalf("expected all backups, got %d", len(got))
	}
}

func TestFilterBackupsByType(t *testing.T) {
	records := []models.ExportRecord{
		{ID: "db", ExportType: models.ExportTypeDatabase},
		{ID: "files", ExportType: models.ExportTypeFiles},
		{ID: "legacy"},
	}
	got := filterBackupsByType(records, string(models.ExportTypeFiles))
	if len(got) != 1 || got[0].ID != "files" {
		t.Fatalf("expected files only, got %#v", got)
	}
	got = filterBackupsByType(records, string(models.ExportTypeDatabase))
	if len(got) != 2 {
		t.Fatalf("expected database + legacy, got %d", len(got))
	}
}

func TestExportTypeLabel(t *testing.T) {
	if exportTypeLabel(models.ExportTypeFiles) != "Files" {
		t.Fatal("expected Files label")
	}
	if exportTypeLabel(models.ExportTypeDatabase) != "DB" {
		t.Fatal("expected DB label")
	}
}

func TestCollectGroups(t *testing.T) {
	profiles := []models.Profile{
		{Group: "Beta"},
		{Group: "Alpha"},
		{Group: ""},
	}
	groups := collectGroups(profiles)
	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %#v", groups)
	}
	if groups[0] != "Alpha" || groups[1] != "Beta" || groups[2] != "Default" {
		t.Fatalf("unexpected sorted groups: %#v", groups)
	}
}
