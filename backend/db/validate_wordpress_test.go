package db

import (
	"testing"

	"dback/models"
)

func TestValidateProfileForWordPress(t *testing.T) {
	t.Parallel()

	valid := models.Profile{
		ConnectionType: models.ConnectionTypeWordPress,
		WPUrl:          "https://example.com",
		WPKey:          "token",
	}
	if err := ValidateProfileForWordPress(valid); err != nil {
		t.Fatalf("expected valid profile, got %v", err)
	}

	missingKey := valid
	missingKey.WPKey = ""
	if err := ValidateProfileForWordPress(missingKey); err == nil {
		t.Fatal("expected missing key error")
	}
}

func TestValidateWordPressDatabaseName(t *testing.T) {
	t.Parallel()
	if err := ValidateWordPressDatabaseName(""); err != nil {
		t.Fatalf("empty database name should be allowed: %v", err)
	}
	if err := ValidateWordPressDatabaseName("staging_db"); err != nil {
		t.Fatalf("valid database name rejected: %v", err)
	}
	if err := ValidateWordPressDatabaseName("bad-name"); err == nil {
		t.Fatal("expected invalid database name error")
	}
}

func TestProfileUsesWordPress(t *testing.T) {
	t.Parallel()
	p := models.Profile{ConnectionType: models.ConnectionTypeWordPress}
	if !p.UsesWordPress() {
		t.Fatal("expected UsesWordPress")
	}
	if !p.SupportsSQLQuery() {
		t.Fatal("expected SupportsSQLQuery for wordpress")
	}
}
