package app

import (
	"context"
	"path/filepath"
	"testing"
)

func TestDiskFreeBytes(t *testing.T) {
	dir := t.TempDir()
	free, err := diskFreeBytes(dir)
	if err != nil {
		t.Fatal(err)
	}
	if free == 0 {
		t.Fatal("expected non-zero free disk space")
	}
}

func TestServerInfo(t *testing.T) {
	dir := t.TempDir()
	a, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	info := a.ServerInfo(context.Background(), dir)
	if info.CPUCount < 1 {
		t.Fatalf("cpu_count = %d, want >= 1", info.CPUCount)
	}
	if info.DataDir != dir {
		t.Fatalf("data_dir = %q, want %q", info.DataDir, dir)
	}
	if info.DiskFreeBytes == 0 {
		t.Fatal("expected disk_free_bytes > 0")
	}
	if info.InternetHost != "google.com" {
		t.Fatalf("internet_host = %q", info.InternetHost)
	}
	if info.CheckedAt == "" {
		t.Fatal("expected checked_at")
	}
}

func TestDiskFreeBytesEmptyPath(t *testing.T) {
	if _, err := diskFreeBytes(""); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestDiskFreeBytesRealPath(t *testing.T) {
	free, err := diskFreeBytes(filepath.Clean("/"))
	if err != nil {
		t.Fatal(err)
	}
	if free == 0 {
		t.Fatal("expected non-zero free bytes on root")
	}
}
