package ssh

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHostKeyCallbackRequiresPath(t *testing.T) {
	SetKnownHostsFile("")
	_, err := hostKeyCallback()
	if err == nil {
		t.Fatal("expected error when known_hosts path is empty")
	}
}

func TestHostKeyCallbackCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ssh_known_hosts")
	SetKnownHostsFile(path)
	cb, err := hostKeyCallback()
	if err != nil {
		t.Fatal(err)
	}
	if cb == nil {
		t.Fatal("expected callback")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected known_hosts file to exist: %v", err)
	}
}
