package models

import (
	"strings"
	"time"
)

const RemoteProviderS3 RemoteProviderType = "s3"

// RemoteProviderType identifies a remote storage backend.
type RemoteProviderType string

// S3DestinationConfig holds S3-compatible connection settings.
type S3DestinationConfig struct {
	Endpoint    string `json:"endpoint"`
	Region      string `json:"region,omitempty"`
	Bucket      string `json:"bucket"`
	AccessKeyID string `json:"access_key_id"`
	SecretKey   string `json:"secret_key"`
	UseSSL      bool   `json:"use_ssl"`
}

func (c *S3DestinationConfig) Clone() *S3DestinationConfig {
	if c == nil {
		return nil
	}
	cp := *c
	return &cp
}

// RemoteDestination is a named remote storage target.
type RemoteDestination struct {
	ID   string               `json:"id"`
	Name string               `json:"name"`
	Type RemoteProviderType   `json:"type"`
	S3   *S3DestinationConfig `json:"s3,omitempty"`
}

func (d *RemoteDestination) Clone() RemoteDestination {
	if d == nil {
		return RemoteDestination{}
	}
	cp := *d
	cp.S3 = d.S3.Clone()
	return cp
}

// RemoteUploadStatus tracks per-destination upload progress for a backup record.
type RemoteUploadStatus string

const (
	RemoteUploadPending   RemoteUploadStatus = "pending"
	RemoteUploadUploading RemoteUploadStatus = "uploading"
	RemoteUploadDone      RemoteUploadStatus = "done"
	RemoteUploadFailed    RemoteUploadStatus = "failed"
)

// RemoteUploadState records upload outcome for one destination.
type RemoteUploadState struct {
	DestinationID string             `json:"destination_id"`
	Status        RemoteUploadStatus `json:"status"`
	RemoteKey     string             `json:"remote_key,omitempty"`
	UploadedAt    time.Time          `json:"uploaded_at,omitempty"`
	ETag          string             `json:"etag,omitempty"`
	SizeBytes     int64              `json:"size_bytes,omitempty"`
	Error         string             `json:"error,omitempty"`
}

// RemoteDestinationFromSyncSettings builds a destination from legacy sync settings.
func RemoteDestinationFromSyncSettings(id, name string, sync SyncSettings) RemoteDestination {
	return RemoteDestination{
		ID:   id,
		Name: name,
		Type: RemoteProviderS3,
		S3: &S3DestinationConfig{
			Endpoint:    sync.Endpoint,
			Region:      sync.Region,
			Bucket:      sync.Bucket,
			AccessKeyID: sync.AccessKeyID,
			SecretKey:   sync.SecretKey,
			UseSSL:      sync.UseSSL,
		},
	}
}

// ToSyncSettings converts an S3 destination back to legacy sync settings.
func (d RemoteDestination) ToSyncSettings() *SyncSettings {
	if d.Type != RemoteProviderS3 || d.S3 == nil {
		return nil
	}
	return &SyncSettings{
		Endpoint:    d.S3.Endpoint,
		Region:      d.S3.Region,
		Bucket:      d.S3.Bucket,
		AccessKeyID: d.S3.AccessKeyID,
		SecretKey:   d.S3.SecretKey,
		UseSSL:      d.S3.UseSSL,
	}
}

// SyncSettingsConfigured reports whether legacy sync settings have required fields.
func SyncSettingsConfigured(s *SyncSettings) bool {
	if s == nil {
		return false
	}
	return strings.TrimSpace(s.Endpoint) != "" &&
		strings.TrimSpace(s.Bucket) != "" &&
		strings.TrimSpace(s.AccessKeyID) != "" &&
		strings.TrimSpace(s.SecretKey) != ""
}

// ValidateRemoteDestination checks destination fields for save.
func ValidateRemoteDestination(d RemoteDestination) error {
	name := strings.TrimSpace(d.Name)
	if name == "" {
		return errRemoteDestinationNameRequired
	}
	switch d.Type {
	case RemoteProviderS3:
		if d.S3 == nil {
			return errRemoteDestinationS3Required
		}
		if strings.TrimSpace(d.S3.Endpoint) == "" ||
			strings.TrimSpace(d.S3.Bucket) == "" ||
			strings.TrimSpace(d.S3.AccessKeyID) == "" ||
			strings.TrimSpace(d.S3.SecretKey) == "" {
			return errRemoteDestinationS3Incomplete
		}
	default:
		return errRemoteDestinationUnsupportedType
	}
	return nil
}

// RemoteUploadDoneForDestination reports whether upload succeeded for a destination.
func RemoteUploadDoneForDestination(uploads []RemoteUploadState, destinationID string) bool {
	for _, u := range uploads {
		if u.DestinationID == destinationID && u.Status == RemoteUploadDone {
			return true
		}
	}
	return false
}

// RemoteUploadStateForDestination returns upload state for a destination if present.
func RemoteUploadStateForDestination(uploads []RemoteUploadState, destinationID string) (RemoteUploadState, bool) {
	for _, u := range uploads {
		if u.DestinationID == destinationID {
			return u, true
		}
	}
	return RemoteUploadState{}, false
}

type remoteErr string

func (e remoteErr) Error() string { return string(e) }

const (
	errRemoteDestinationNameRequired    remoteErr = "destination name is required"
	errRemoteDestinationS3Required      remoteErr = "s3 configuration is required"
	errRemoteDestinationS3Incomplete    remoteErr = "endpoint, bucket, access key, and secret key are required"
	errRemoteDestinationUnsupportedType remoteErr = "unsupported remote destination type"
)

// ErrRemoteDestinationUnsupportedType returns unsupported type error for external packages.
func ErrRemoteDestinationUnsupportedType() error {
	return errRemoteDestinationUnsupportedType
}
