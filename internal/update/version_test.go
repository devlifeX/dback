package update

import (
	"testing"
)

func TestNormalizeVersion(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"v3.6.1": "3.6.1",
		"3.6":    "3.6.0",
		"3":      "3.0.0",
		"":       "0.0.0",
	}
	for input, want := range cases {
		if got := NormalizeVersion(input); got != want {
			t.Fatalf("NormalizeVersion(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	t.Parallel()

	if CompareVersions("3.6.1", "3.6.2") >= 0 {
		t.Fatal("expected 3.6.1 < 3.6.2")
	}
	if CompareVersions("3.6.2", "3.6.1") <= 0 {
		t.Fatal("expected 3.6.2 > 3.6.1")
	}
	if CompareVersions("v3.6.1", "3.6.1") != 0 {
		t.Fatal("expected equal normalized versions")
	}
}

func TestIsNewer(t *testing.T) {
	t.Parallel()

	if !IsNewer("3.6.1", "3.6.2") {
		t.Fatal("expected newer")
	}
	if IsNewer("3.6.2", "3.6.1") {
		t.Fatal("expected not newer")
	}
}
