package transfer

import "testing"

func TestProgressTotal(t *testing.T) {
	if got := progressTotal(500, 1000); got != 1000 {
		t.Fatalf("expected unchanged estimate, got %d", got)
	}
	if got := progressTotal(1200, 1000); got != 1440 {
		t.Fatalf("expected scaled estimate, got %d", got)
	}
	if got := progressTotal(100, 0); got != 0 {
		t.Fatalf("expected zero when no estimate, got %d", got)
	}
}
