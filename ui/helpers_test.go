package ui

import (
	"testing"

	"dback/models"
)

func TestImportableProfiles(t *testing.T) {
	profiles := []models.Profile{
		{ID: "a", Name: "Prod", ImportProtected: true},
		{ID: "b", Name: "Staging", ImportProtected: false},
	}
	got := importableProfiles(profiles)
	if len(got) != 1 || got[0].ID != "b" {
		t.Fatalf("expected only staging host, got %+v", got)
	}
}

func TestAllowsImport(t *testing.T) {
	if (models.Profile{ImportProtected: true}).AllowsImport() {
		t.Fatal("protected host should not allow import")
	}
	if !(models.Profile{ImportProtected: false}).AllowsImport() {
		t.Fatal("unprotected host should allow import")
	}
}
