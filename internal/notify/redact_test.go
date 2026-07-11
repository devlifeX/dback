package notify

import (
	"testing"
)

func TestSanitizeErrorRedactsSecrets(t *testing.T) {
	in := "connect failed: password=supersecret and token=abc123"
	out := SanitizeError(in)
	if out == in {
		t.Fatalf("expected redaction, got %q", out)
	}
	if contains(out, "supersecret") || contains(out, "abc123") {
		t.Fatalf("secrets leaked: %q", out)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && searchSubstring(s, sub))
}

func searchSubstring(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
