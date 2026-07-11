package models

import (
	"encoding/json"
	"fmt"
	"time"
)

type OverlapPolicy string

const (
	OverlapPolicySkip OverlapPolicy = "skip"
)

type TriggerType string

const (
	TriggerCron     TriggerType = "cron"
	TriggerInterval TriggerType = "interval"
	TriggerOneShot  TriggerType = "one_shot"
	TriggerOnBoot   TriggerType = "on_boot"
)

type CronTrigger struct {
	Expr     string `json:"expr"`
	Timezone string `json:"timezone"`
}

type IntervalTrigger struct {
	Every time.Duration `json:"every"`
}

type OneShotTrigger struct {
	At time.Time `json:"at"`
}

type OnBootTrigger struct {
	Delay time.Duration `json:"delay"`
}

type TriggerSpec struct {
	Type     TriggerType      `json:"type"`
	Cron     *CronTrigger     `json:"cron,omitempty"`
	Interval *IntervalTrigger `json:"interval,omitempty"`
	OneShot  *OneShotTrigger  `json:"one_shot,omitempty"`
	OnBoot   *OnBootTrigger   `json:"on_boot,omitempty"`
}

type TriggerState struct {
	LastFiredAt   time.Time `json:"last_fired_at,omitempty"`
	NextRunAt     time.Time `json:"next_run_at,omitempty"`
	LastRunStatus string    `json:"last_run_status,omitempty"`
	OnBootFired   bool      `json:"on_boot_fired,omitempty"`
}

type ActionSpec struct {
	Operation string          `json:"operation"`
	Params    json.RawMessage `json:"params,omitempty"`
}

type Task struct {
	ID                    string        `json:"id"`
	Name                  string        `json:"name"`
	Enabled               bool          `json:"enabled"`
	Trigger               TriggerSpec   `json:"trigger"`
	Actions               []ActionSpec  `json:"actions"`
	ProfileIDs            []string      `json:"profile_ids"`
	OverlapPolicy         OverlapPolicy `json:"overlap_policy,omitempty"`
	MaxConcurrentProfiles int           `json:"max_concurrent_profiles,omitempty"`
	State                 TriggerState  `json:"state,omitempty"`
}

type ActionResult struct {
	OperationID string `json:"operation_id"`
	Kind        string `json:"kind"`
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
}

type TaskRunRecord struct {
	ID            string         `json:"id"`
	TaskID        string         `json:"task_id"`
	ProfileID     string         `json:"profile_id"`
	TriggerType   TriggerType    `json:"trigger_type"`
	StartedAt     time.Time      `json:"started_at"`
	FinishedAt    time.Time      `json:"finished_at"`
	Status        string         `json:"status"`
	ActionResults []ActionResult `json:"action_results,omitempty"`
}

func (t Task) EffectiveMaxConcurrentProfiles() int {
	if t.MaxConcurrentProfiles <= 0 {
		return 1
	}
	return t.MaxConcurrentProfiles
}

func (t Task) EffectiveOverlapPolicy() OverlapPolicy {
	if t.OverlapPolicy == "" {
		return OverlapPolicySkip
	}
	return t.OverlapPolicy
}

func (t Task) HasAction(kind string) bool {
	for _, a := range t.Actions {
		if a.Operation == kind {
			return true
		}
	}
	return false
}

func (ts TriggerSpec) Validate() error {
	switch ts.Type {
	case TriggerCron:
		if ts.Cron == nil || ts.Cron.Expr == "" {
			return fmt.Errorf("cron trigger requires expr")
		}
		if ts.Cron.Timezone == "" {
			return fmt.Errorf("cron trigger requires IANA timezone")
		}
	case TriggerInterval:
		if ts.Interval == nil || ts.Interval.Every <= 0 {
			return fmt.Errorf("interval trigger requires positive every duration")
		}
	case TriggerOneShot:
		if ts.OneShot == nil || ts.OneShot.At.IsZero() {
			return fmt.Errorf("one_shot trigger requires at time")
		}
	case TriggerOnBoot:
		if ts.OnBoot == nil {
			return fmt.Errorf("on_boot trigger requires delay config")
		}
		if ts.OnBoot.Delay < 0 {
			return fmt.Errorf("on_boot delay must be non-negative")
		}
	default:
		return fmt.Errorf("unsupported trigger type %q", ts.Type)
	}
	return nil
}
