package operation

import (
	"fmt"
	"time"
)

type Kind string

const (
	KindBackupDB    Kind = "backup_db"
	KindBackupFiles Kind = "backup_files"
	KindUpload      Kind = "upload"
)

type Status string

const (
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusCanceled  Status = "canceled"
	StatusSkipped   Status = "skipped"
)

type UploadStalePolicy string

const (
	UploadStaleNewOnly    UploadStalePolicy = "new_only"
	UploadStaleLatestOnly UploadStalePolicy = "latest_only"
	UploadStaleAllStale   UploadStalePolicy = "all_stale"
	UploadStaleFail       UploadStalePolicy = "fail"
)

type Params interface {
	Kind() Kind
	Validate() error
}

type BackupDBParams struct{}

func (BackupDBParams) Kind() Kind { return KindBackupDB }

func (BackupDBParams) Validate() error { return nil }

type BackupFilesParams struct{}

func (BackupFilesParams) Kind() Kind { return KindBackupFiles }

func (BackupFilesParams) Validate() error { return nil }

type UploadParams struct {
	RecordIDs   []string
	UploadAll   bool
	StalePolicy UploadStalePolicy
}

func (UploadParams) Kind() Kind { return KindUpload }

func (p UploadParams) Validate() error {
	switch p.StalePolicy {
	case "", UploadStaleNewOnly, UploadStaleLatestOnly, UploadStaleAllStale, UploadStaleFail:
		return nil
	default:
		return fmt.Errorf("invalid upload stale policy %q", p.StalePolicy)
	}
}

func (p UploadParams) EffectiveStalePolicy() UploadStalePolicy {
	if p.StalePolicy == "" {
		return UploadStaleNewOnly
	}
	return p.StalePolicy
}

type Spec struct {
	ID         string
	Kind       Kind
	ProfileID  string
	TriggerRef string
	Params     Params
}

func (s Spec) Validate() error {
	if s.Kind == "" {
		return fmt.Errorf("operation kind is required")
	}
	if s.ProfileID == "" {
		return fmt.Errorf("profile id is required")
	}
	if s.Params == nil {
		return fmt.Errorf("operation params are required")
	}
	if s.Params.Kind() != s.Kind {
		return fmt.Errorf("params kind %q does not match spec kind %q", s.Params.Kind(), s.Kind)
	}
	return s.Params.Validate()
}

type ArtifactType string

const (
	ArtifactExportRecord ArtifactType = "export_record"
)

type Artifact struct {
	Type ArtifactType
	ID   string
	Path string
}

type Result struct {
	OperationID string
	Kind        Kind
	Status      Status
	StartedAt   time.Time
	FinishedAt  time.Time
	Error       string
	Artifacts   []Artifact
}

type ChainResult struct {
	OperationID string
	Results     []Result
	Status      Status
	Error       string
}
