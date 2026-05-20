package ssh

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"dback/models"
)

func TestLoadPrivateKeyPEMFirst(t *testing.T) {
	pem := "-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----"
	key, err := loadPrivateKey(pem, "/nonexistent/path")
	if err != nil {
		t.Fatal(err)
	}
	if string(key) != pem {
		t.Fatalf("expected PEM content, got %q", key)
	}
}

func TestLoadPrivateKeyFromPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "id_rsa")
	content := []byte("-----BEGIN RSA PRIVATE KEY-----\nabc\n-----END RSA PRIVATE KEY-----")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	key, err := loadPrivateKey("", path)
	if err != nil {
		t.Fatal(err)
	}
	if string(key) != string(content) {
		t.Fatalf("expected file content, got %q", key)
	}
}

func TestLoadPrivateKeyMissing(t *testing.T) {
	_, err := loadPrivateKey("", "")
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestIsRetryableError(t *testing.T) {
	if !isRetryableError(fmt.Errorf("connection reset by peer")) {
		t.Fatal("expected connection reset to be retryable")
	}
	if isRetryableError(fmt.Errorf("permission denied")) {
		t.Fatal("expected permission denied to be non-retryable")
	}
}

func TestSSHConfigPasswordAuth(t *testing.T) {
	SetKnownHostsFile(filepath.Join(t.TempDir(), "known_hosts"))
	cfg, err := sshConfig("user", "pass", models.AuthTypePassword, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Auth) != 1 {
		t.Fatalf("expected 1 auth method, got %d", len(cfg.Auth))
	}
}
