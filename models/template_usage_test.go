package models

import "testing"

func TestTemplateBodyChanged(t *testing.T) {
	if TemplateBodyChanged("SELECT 1", "SELECT 1") {
		t.Fatal("expected unchanged")
	}
	if !TemplateBodyChanged("SELECT 1", "SELECT 2") {
		t.Fatal("expected changed")
	}
	if TemplateBodyChanged(" SELECT 1 ", "SELECT 1\n") {
		t.Fatal("expected whitespace-only differences to be unchanged")
	}
}

func TestFindProfilesUsingTemplate(t *testing.T) {
	oldBody := "DELETE FROM `{databasename}`.logs;"
	profiles := []Profile{
		{
			ID:             "a",
			Name:           "Prod",
			TargetDBName:   "app",
			PreImportQuery: "DELETE FROM `app`.logs;",
		},
		{
			ID:              "b",
			Name:            "Staging",
			TargetDBName:    "app",
			PostImportQuery: "SELECT 1;",
		},
	}
	usages := FindProfilesUsingTemplate(profiles, oldBody)
	if len(usages) != 1 || usages[0].ProfileID != "a" || !usages[0].InPreImport {
		t.Fatalf("unexpected usages: %+v", usages)
	}
}

func TestReplaceTemplateInProfile(t *testing.T) {
	p := Profile{
		Name:           "Prod",
		TargetDBName:   "app",
		PreImportQuery: "DELETE FROM `app`.logs;",
	}
	updated := ReplaceTemplateInProfile(p, "DELETE FROM `{databasename}`.logs;", "TRUNCATE `{databasename}`.logs;")
	want := "TRUNCATE `app`.logs;"
	if updated.PreImportQuery != want {
		t.Fatalf("got %q want %q", updated.PreImportQuery, want)
	}
}
