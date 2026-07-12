package operation

import (
	"fmt"
	"strings"
	"time"
)

type Kind string

const (
	KindBackupDB    Kind = "backup_db"
	KindBackupFiles Kind = "backup_files"
	KindUpload      Kind = "upload"
	KindRestore     Kind = "restore"
	KindDeepVerify  Kind = "deep_verify"
	KindUrlChecker  Kind = "url_checker"
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

type RestoreParams struct {
	RecordID             string `json:"record_id"`
	DestinationProfileID string `json:"destination_profile_id"`
}

func (RestoreParams) Kind() Kind { return KindRestore }

func (p RestoreParams) Validate() error {
	if strings.TrimSpace(p.RecordID) == "" {
		return fmt.Errorf("record_id is required")
	}
	if strings.TrimSpace(p.DestinationProfileID) == "" {
		return fmt.Errorf("destination_profile_id is required")
	}
	return nil
}

type DeepVerifyParams struct {
	RecordID             string `json:"record_id"`
	DestinationProfileID string `json:"destination_profile_id"`
}

func (DeepVerifyParams) Kind() Kind { return KindDeepVerify }

func (p DeepVerifyParams) Validate() error {
	if strings.TrimSpace(p.RecordID) == "" {
		return fmt.Errorf("record_id is required")
	}
	if strings.TrimSpace(p.DestinationProfileID) == "" {
		return fmt.Errorf("destination_profile_id is required")
	}
	return nil
}

type UrlCheckerParams struct {
	URLIndex *int `json:"url_index"`
	UseProxy bool `json:"use_proxy"`
	Timeout  int  `json:"timeout_seconds"`
}

func (UrlCheckerParams) Kind() Kind { return KindUrlChecker }

func (p UrlCheckerParams) Validate() error {
	if p.Timeout < 0 {
		return fmt.Errorf("timeout_seconds must be >= 0")
	}
	if p.URLIndex != nil && *p.URLIndex < -1 {
		return fmt.Errorf("url_index must be >= -1")
	}
	return nil
}

type Spec struct {
	ID         string
	Kind       Kind
	ProfileID  string
	TaskID     string
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
	Details     string
	Artifacts   []Artifact
}

type ChainResult struct {
	OperationID string
	Results     []Result
	Status      Status
	Error       string
}
