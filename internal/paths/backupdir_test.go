package paths

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestDefaultBackupDestinationUsesHome(t *testing.T) {
	t.Setenv("USERPROFILE", `C:\Users\testuser`)
	t.Setenv("HOME", "/home/testuser")

	got := DefaultBackupDestination()
	if got == "" {
		t.Fatal("expected non-empty path")
	}
	if runtime.GOOS == "windows" {
		want := filepath.Join(`C:\Users\testuser`, "Documents", "dback")
		if got != want {
			t.Fatalf("windows path: got %q want %q", got, want)
		}
		return
	}
	want := filepath.Join("/home/testuser", "dback", "backups")
	if got != want {
		t.Fatalf("unix path: got %q want %q", got, want)
	}
}

func TestEffectiveBackupDestinationMigratesLegacyMjavadPath(t *testing.T) {
	t.Setenv("HOME", "/home/devlife")

	got := EffectiveBackupDestination("/home/mjavad/dback/backups")
	want := filepath.Join("/home/devlife", "dback", "backups")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestEffectiveBackupDestinationKeepsCustomPath(t *testing.T) {
	t.Setenv("HOME", "/home/devlife")

	custom := "/mnt/shared/backups"
	if got := EffectiveBackupDestination(custom); got != custom {
		t.Fatalf("custom path changed: got %q want %q", got, custom)
	}
}

func TestEffectiveBackupDestinationKeepsCurrentUserDefault(t *testing.T) {
	t.Setenv("HOME", "/home/devlife")

	current := filepath.Join("/home/devlife", "dback", "backups")
	if got := EffectiveBackupDestination(current); got != current {
		t.Fatalf("current user default changed: got %q want %q", got, current)
	}
}

func TestMigrateBackupDestination(t *testing.T) {
	t.Setenv("HOME", "/home/devlife")

	if _, changed := MigrateBackupDestination(""); changed {
		t.Fatal("empty destination should not migrate")
	}

	newDest, changed := MigrateBackupDestination("/home/mjavad/dback/backups")
	if !changed {
		t.Fatal("expected legacy path to migrate")
	}
	want := filepath.Join("/home/devlife", "dback", "backups")
	if newDest != want {
		t.Fatalf("got %q want %q", newDest, want)
	}

	_, changed = MigrateBackupDestination("/mnt/shared/backups")
	if changed {
		t.Fatal("custom mount path should not migrate")
	}
}

func TestMigrateBackupDestinationOtherUnixHome(t *testing.T) {
	t.Setenv("HOME", "/home/devlife")

	newDest, changed := MigrateBackupDestination("/home/otheruser/dback/backups")
	if !changed {
		t.Fatal("expected other user's home default to migrate")
	}
	want := filepath.Join("/home/devlife", "dback", "backups")
	if newDest != want {
		t.Fatalf("got %q want %q", newDest, want)
	}
}
