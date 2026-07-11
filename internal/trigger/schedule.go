package trigger

import (
	"fmt"
	"time"

	"dback/models"

	"github.com/robfig/cron/v3"
)

func NextRunAfter(task models.Task, after time.Time) (time.Time, error) {
	switch task.Trigger.Type {
	case models.TriggerCron:
		return nextCronRun(task.Trigger.Cron, after)
	case models.TriggerInterval:
		return after.Add(task.Trigger.Interval.Every), nil
	case models.TriggerOneShot:
		at := task.Trigger.OneShot.At
		if !after.Before(at) {
			return time.Time{}, nil
		}
		return at, nil
	case models.TriggerOnBoot:
		return time.Time{}, nil
	default:
		return time.Time{}, fmt.Errorf("unsupported trigger type %q", task.Trigger.Type)
	}
}

func InitNextRun(task models.Task, now time.Time) (time.Time, error) {
	switch task.Trigger.Type {
	case models.TriggerCron, models.TriggerInterval:
		return NextRunAfter(task, now)
	case models.TriggerOneShot:
		if !task.State.LastFiredAt.IsZero() {
			return time.Time{}, nil
		}
		at := task.Trigger.OneShot.At
		if now.Before(at) {
			return at, nil
		}
		return now, nil
	case models.TriggerOnBoot:
		return time.Time{}, nil
	default:
		return time.Time{}, fmt.Errorf("unsupported trigger type %q", task.Trigger.Type)
	}
}

func ShouldFire(task models.Task, now time.Time) bool {
	if !task.Enabled {
		return false
	}
	switch task.Trigger.Type {
	case models.TriggerOnBoot:
		return false
	case models.TriggerOneShot:
		if !task.State.LastFiredAt.IsZero() {
			return false
		}
		if task.State.NextRunAt.IsZero() {
			return false
		}
		return !now.Before(task.State.NextRunAt)
	default:
		if task.State.NextRunAt.IsZero() {
			return false
		}
		return !now.Before(task.State.NextRunAt)
	}
}

func nextCronRun(cfg *models.CronTrigger, after time.Time) (time.Time, error) {
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timezone %q: %w", cfg.Timezone, err)
	}
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(cfg.Expr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid cron expr %q: %w", cfg.Expr, err)
	}
	inLoc := after.In(loc)
	next := schedule.Next(inLoc)
	return next.UTC(), nil
}
