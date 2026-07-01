package app

import (
	"testing"

	"dback/models"
)

func TestNormalizeProfileFileBackupDuplicateName(t *testing.T) {
	p := models.Profile{
		FileBackupPaths: []models.FileBackupPath{
			{ID: "1", Name: "Uploads", RemotePath: "/var/a"},
			{ID: "2", Name: "uploads", RemotePath: "/var/b"},
		},
	}
	if err := normalizeProfileFileBackup(&p); err == nil {
		t.Fatal("expected duplicate name error")
	}
}

func TestNormalizeProfileFileBackupInvalidExclude(t *testing.T) {
	p := models.Profile{
		FileBackupExclude: []string{";bad"},
	}
	if err := normalizeProfileFileBackup(&p); err == nil {
		t.Fatal("expected exclude validation error")
	}
}
