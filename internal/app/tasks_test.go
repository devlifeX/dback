package app

import (
	"testing"
	"time"

	"dback/internal/operation"
	"dback/models"
)

func TestValidateTaskRejectsWordPressWithBackupFiles(t *testing.T) {
	dir := t.TempDir()
	a, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.CreateVault("test-pass"); err != nil {
		t.Fatal(err)
	}
	profile := models.Profile{
		ID:             "wp1",
		Name:           "wordpress",
		ConnectionType: models.ConnectionTypeWordPress,
	}
	if err := a.SaveProfile(profile); err != nil {
		t.Fatal(err)
	}
	task := models.Task{
		Name:       "bad",
		ProfileIDs: []string{"wp1"},
		Trigger: models.TriggerSpec{
			Type: models.TriggerInterval,
			Interval: &models.IntervalTrigger{
				Every: time.Hour,
			},
		},
		Actions: []models.ActionSpec{{
			Operation: string(operation.KindBackupFiles),
		}},
	}
	if err := a.ValidateTask(task); err == nil {
		t.Fatal("expected validation error for backup_files + wordpress")
	}
}

func TestValidateTaskRequiresProfiles(t *testing.T) {
	dir := t.TempDir()
	a, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.CreateVault("test-pass"); err != nil {
		t.Fatal(err)
	}
	task := models.Task{
		Name: "empty",
		Trigger: models.TriggerSpec{
			Type: models.TriggerInterval,
			Interval: &models.IntervalTrigger{
				Every: time.Hour,
			},
		},
		Actions: []models.ActionSpec{{
			Operation: string(operation.KindBackupDB),
		}},
	}
	if err := a.ValidateTask(task); err == nil {
		t.Fatal("expected validation error for missing profiles")
	}
}

func TestTaskVaultRoundTrip(t *testing.T) {
	dir := t.TempDir()
	a, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.CreateVault("test-pass"); err != nil {
		t.Fatal(err)
	}
	if err := a.SaveProfile(models.Profile{ID: "p1", Name: "host", ConnectionType: models.ConnectionTypeLocalhost}); err != nil {
		t.Fatal(err)
	}
	task := models.Task{
		Name:       "nightly",
		Enabled:    true,
		ProfileIDs: []string{"p1"},
		Trigger: models.TriggerSpec{
			Type: models.TriggerCron,
			Cron: &models.CronTrigger{
				Expr:     "0 2 * * *",
				Timezone: "UTC",
			},
		},
		Actions: []models.ActionSpec{
			{Operation: string(operation.KindBackupDB)},
			{Operation: string(operation.KindUpload), Params: []byte(`{"stale_policy":"new_only"}`)},
		},
	}
	if err := a.SaveTask(task); err != nil {
		t.Fatal(err)
	}
	a.Lock()
	if err := a.Unlock("test-pass"); err != nil {
		t.Fatal(err)
	}
	tasks, err := a.ListTasks()
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].Name != "nightly" {
		t.Fatalf("unexpected tasks: %+v", tasks)
	}
}
