package app

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"dback/models"
)

func TestShouldRunPreImportQuery(t *testing.T) {
	t.Parallel()

	wp := models.Profile{
		ConnectionType: models.ConnectionTypeWordPress,
		PreImportQuery: "DROP DATABASE test;",
	}
	if !shouldRunPreImportQuery(wp) {
		t.Fatal("expected wordpress pre-import query to run when SQL text is set")
	}

	wp.RunQueryBeforeImport = false
	if !shouldRunPreImportQuery(wp) {
		t.Fatal("expected pre-import query to run even when run flag is false but SQL text exists")
	}

	wp.PreImportQuery = "   "
	if shouldRunPreImportQuery(wp) {
		t.Fatal("expected empty pre-import query to be skipped")
	}
}

func TestRestoreWordPressRunsPreImportBeforeImport(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	order := make([]string, 0, 2)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-DBACK-KEY"); got != "secret-key" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		mu.Lock()
		defer mu.Unlock()

		switch r.URL.Path {
		case "/wp-json/dback/v1/preflight/":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"checks":  []map[string]string{{"name": "database", "status": "ok"}},
			})
		case "/wp-json/dback/v1/query/":
			order = append(order, "query")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"type":    "command",
			})
		case "/wp-json/dback/v1/import/":
			order = append(order, "import")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success":             true,
				"statements_executed": float64(1),
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	backupPath := writeTestGzipBackup(t, "-- test\nSELECT 1;\n")

	dir := t.TempDir()
	a := openApp(t, dir)
	profile := models.Profile{
		ID:                   "wp1",
		Name:                 "WP Site",
		ConnectionType:       models.ConnectionTypeWordPress,
		WPUrl:                server.URL,
		WPKey:                "secret-key",
		TargetDBName:         "test",
		PreImportQuery:       "DROP DATABASE IF EXISTS test; CREATE DATABASE test;",
		RunQueryBeforeImport: true,
	}
	if err := a.SaveProfile(profile); err != nil {
		t.Fatal(err)
	}

	record := models.ExportRecord{
		FilePath:      backupPath,
		FileSizeBytes: fileSize(t, backupPath),
	}

	dest := a.Profiles()[0]
	if err := a.Restore(context.Background(), record, dest, nil); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	mu.Lock()
	got := append([]string(nil), order...)
	mu.Unlock()

	if len(got) != 2 || got[0] != "query" || got[1] != "import" {
		t.Fatalf("expected [query import], got %#v", got)
	}
}

func TestRestoreWordPressSkipsImportWhenPreImportFails(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	order := make([]string, 0, 2)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wp-json/dback/v1/preflight/":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
		case "/wp-json/dback/v1/query/":
			mu.Lock()
			order = append(order, "query")
			mu.Unlock()
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"code":    "dback_query_failed",
				"message": "query failed",
			})
		case "/wp-json/dback/v1/import/":
			mu.Lock()
			order = append(order, "import")
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success":             true,
				"statements_executed": float64(1),
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	backupPath := writeTestGzipBackup(t, "-- test\nSELECT 1;\n")
	dir := t.TempDir()
	a := openApp(t, dir)
	profile := models.Profile{
		ID:                   "wp1",
		Name:                 "WP Site",
		ConnectionType:       models.ConnectionTypeWordPress,
		WPUrl:                server.URL,
		WPKey:                "secret-key",
		PreImportQuery:       "DROP DATABASE test;",
		RunQueryBeforeImport: true,
	}
	if err := a.SaveProfile(profile); err != nil {
		t.Fatal(err)
	}

	record := models.ExportRecord{
		FilePath:      backupPath,
		FileSizeBytes: fileSize(t, backupPath),
	}

	err := a.Restore(context.Background(), record, a.Profiles()[0], nil)
	if err == nil || !strings.Contains(err.Error(), "pre-import query failed") {
		t.Fatalf("expected pre-import failure, got: %v", err)
	}

	mu.Lock()
	got := append([]string(nil), order...)
	mu.Unlock()

	if len(got) != 1 || got[0] != "query" {
		t.Fatalf("expected only query call, got %#v", got)
	}
}

func writeTestGzipBackup(t *testing.T, sql string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "backup.sql.gz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gw := gzip.NewWriter(f)
	if _, err := gw.Write([]byte(sql)); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() < 128 {
		padding := make([]byte, 128-info.Size())
		if err := os.WriteFile(path, append(mustReadFile(t, path), padding...), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func fileSize(t *testing.T, path string) int64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Size()
}
