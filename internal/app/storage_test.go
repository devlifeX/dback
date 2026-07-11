package app

import (
	"path/filepath"
	"testing"
)

func TestPathWithinRoot(t *testing.T) {
	root := filepath.Join("/data", "backups")
	tests := []struct {
		path string
		want bool
	}{
		{root, true},
		{filepath.Join(root, "host1"), true},
		{filepath.Join(root, "host1", "dump.sql.gz"), true},
		{"/data/other", false},
	}
	for _, tc := range tests {
		if got := pathWithinRoot(tc.path, root); got != tc.want {
			t.Fatalf("pathWithinRoot(%q, %q) = %v, want %v", tc.path, root, got, tc.want)
		}
	}
}

func TestNormalizeRemotePrefix(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", remoteBackupPrefix},
		{"dback/backups", remoteBackupPrefix},
		{"dback/backups/host1", "dback/backups/host1/"},
	}
	for _, tc := range tests {
		if got := normalizeRemotePrefix(tc.in); got != tc.want {
			t.Fatalf("normalizeRemotePrefix(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestRemoteParentPrefix(t *testing.T) {
	if got := remoteParentPrefix("dback/backups/"); got != "" {
		t.Fatalf("root parent = %q, want empty", got)
	}
	if got := remoteParentPrefix("dback/backups/host1/"); got != remoteBackupPrefix {
		t.Fatalf("host parent = %q, want %q", got, remoteBackupPrefix)
	}
}
