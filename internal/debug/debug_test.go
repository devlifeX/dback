package debug

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestEnabledFromEnv(t *testing.T) {
	os.Setenv("DBACK_DEBUG", "1")
	if !EnabledFromEnv() {
		t.Fatal("expected DBACK_DEBUG=1 to enable")
	}
	os.Setenv("DBACK_DEBUG", "true")
	if !EnabledFromEnv() {
		t.Fatal("expected DBACK_DEBUG=true to enable")
	}
	os.Setenv("DBACK_DEBUG", "0")
	if EnabledFromEnv() {
		t.Fatal("expected DBACK_DEBUG=0 to disable")
	}
}

func TestLogFormat(t *testing.T) {
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	Enabled = true

	Log("Info", "Export", "Started", "Starting backup", "myhost", "op123", "")

	w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	os.Stderr = old
	Enabled = false

	out := buf.String()
	if !strings.Contains(out, "[Info] Export Started") {
		t.Fatalf("unexpected log line: %q", out)
	}
	if !strings.Contains(out, "profile=myhost") {
		t.Fatalf("missing profile: %q", out)
	}
}
