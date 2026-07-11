package trigger

import (
	"testing"
	"time"

	"dback/models"
)

func TestNextRunCronDST(t *testing.T) {
	task := models.Task{
		Trigger: models.TriggerSpec{
			Type: models.TriggerCron,
			Cron: &models.CronTrigger{
				Expr:     "0 2 * * *",
				Timezone: "America/New_York",
			},
		},
	}
	after := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)
	next, err := NextRunAfter(task, after)
	if err != nil {
		t.Fatal(err)
	}
	if next.Before(after) {
		t.Fatalf("next run %v should be after %v", next, after)
	}
}

func TestInitNextRunOneShotPastDue(t *testing.T) {
	past := time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC)
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	task := models.Task{
		Trigger: models.TriggerSpec{
			Type: models.TriggerOneShot,
			OneShot: &models.OneShotTrigger{
				At: past,
			},
		},
	}
	next, err := InitNextRun(task, now)
	if err != nil {
		t.Fatal(err)
	}
	if !next.Equal(now) {
		t.Fatalf("expected immediate fire at %v, got %v", now, next)
	}
}

func TestInitNextRunOneShotAlreadyFired(t *testing.T) {
	past := time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC)
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	task := models.Task{
		State: models.TriggerState{LastFiredAt: now.Add(-time.Hour)},
		Trigger: models.TriggerSpec{
			Type: models.TriggerOneShot,
			OneShot: &models.OneShotTrigger{
				At: past,
			},
		},
	}
	next, err := InitNextRun(task, now)
	if err != nil {
		t.Fatal(err)
	}
	if !next.IsZero() {
		t.Fatalf("expected zero next run after fired, got %v", next)
	}
}

func TestShouldFireInterval(t *testing.T) {
	now := time.Now()
	task := models.Task{
		Enabled: true,
		Trigger: models.TriggerSpec{Type: models.TriggerInterval},
		State:   models.TriggerState{NextRunAt: now.Add(-time.Minute)},
	}
	if !ShouldFire(task, now) {
		t.Fatal("expected task to fire")
	}
}

func TestIntervalNextRun(t *testing.T) {
	every := 6 * time.Hour
	task := models.Task{
		Trigger: models.TriggerSpec{
			Type:     models.TriggerInterval,
			Interval: &models.IntervalTrigger{Every: every},
		},
	}
	after := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	next, err := NextRunAfter(task, after)
	if err != nil {
		t.Fatal(err)
	}
	want := after.Add(every)
	if !next.Equal(want) {
		t.Fatalf("got %v want %v", next, want)
	}
}
