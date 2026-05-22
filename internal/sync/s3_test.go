package sync

import (
	"testing"

	"dback/models"
)

func TestNormalizeEndpoint(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"s3.amazonaws.com", "s3.amazonaws.com"},
		{"https://s3.amazonaws.com", "s3.amazonaws.com"},
		{"http://minio.local:9000", "minio.local:9000"},
		{"minio.local:9000/path", "minio.local:9000"},
		{"  play.min.io  ", "play.min.io"},
	}
	for _, tc := range tests {
		if got := NormalizeEndpoint(tc.in); got != tc.want {
			t.Fatalf("NormalizeEndpoint(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestValidateSettingsIncomplete(t *testing.T) {
	err := validateSettings(models.SyncSettings{
		Endpoint: "s3.amazonaws.com",
		Bucket:   "bucket",
	})
	if err != ErrSyncIncomplete {
		t.Fatalf("expected ErrSyncIncomplete, got %v", err)
	}
}
