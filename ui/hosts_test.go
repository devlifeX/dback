package ui

import (
	"testing"

	"dback/models"
)

func TestNextDuplicateName(t *testing.T) {
	profiles := []models.Profile{
		{Name: "Production"},
		{Name: "Production 1"},
		{Name: "Production 2"},
	}
	if got := nextDuplicateName("Production", profiles); got != "Production 3" {
		t.Fatalf("expected Production 3, got %q", got)
	}
}
