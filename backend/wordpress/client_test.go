package wordpress

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dback/models"
)

func TestClientPingAndQuery(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-DBACK-KEY"); got != "secret-key" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		switch r.URL.Path {
		case "/wp-json/dback/v1/ping":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"message": "pong",
			})
		case "/wp-json/dback/v1/query":
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["sql"] != "SELECT 1" {
				http.Error(w, "bad sql", http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"type":    "result",
				"columns": []string{"ok"},
				"rows": []map[string]interface{}{
					{"ok": 1},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(models.Profile{
		ConnectionType: models.ConnectionTypeWordPress,
		WPUrl:          server.URL,
		WPKey:          "secret-key",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	data, err := client.Ping(context.Background())
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if data["success"] != true {
		t.Fatalf("expected success ping, got %#v", data)
	}

	result, err := client.Query(context.Background(), "SELECT 1", "")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(result.Columns) != 1 || result.Columns[0] != "ok" {
		t.Fatalf("unexpected columns: %#v", result.Columns)
	}
	if len(result.Rows) != 1 || result.Rows[0][0] != "1" {
		t.Fatalf("unexpected rows: %#v", result.Rows)
	}
}

func TestClientExport(t *testing.T) {
	t.Parallel()

	payload := []byte{0x1f, 0x8b, 0x08}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wp-json/dback/v1/export" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	client, err := NewClient(models.Profile{
		ConnectionType: models.ConnectionTypeWordPress,
		WPUrl:          server.URL,
		WPKey:          "secret-key",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	body, err := client.Export(context.Background())
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(data) != string(payload) {
		t.Fatalf("unexpected export payload: %#v", data)
	}
}

func TestClientQueryWithDatabaseHeader(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-DBACK-DATABASE"); got != "test" {
			http.Error(w, "missing database header", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  true,
			"type":     "result",
			"columns":  []string{"db"},
			"rows":     []map[string]interface{}{{"db": "test"}},
			"database": "test",
		})
	}))
	defer server.Close()

	client, err := NewClient(models.Profile{
		ConnectionType: models.ConnectionTypeWordPress,
		WPUrl:          server.URL,
		WPKey:          "secret-key",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	result, err := client.Query(context.Background(), "SELECT DATABASE() AS db", "test")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(result.Columns) != 1 || result.Columns[0] != "db" {
		t.Fatalf("unexpected columns: %#v", result.Columns)
	}
}

func TestClientImport(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wp-json/dback/v1/import" {
			http.NotFound(w, r)
			return
		}
		body, _ := io.ReadAll(r.Body)
		if len(body) < 2 || body[0] != 0x1f || body[1] != 0x8b {
			http.Error(w, "bad gzip", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success":             true,
			"statements_executed": 12,
			"bytes_received":      len(body),
		})
	}))
	defer server.Close()

	client, err := NewClient(models.Profile{
		ConnectionType: models.ConnectionTypeWordPress,
		WPUrl:          server.URL,
		WPKey:          "secret-key",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	payload := []byte{0x1f, 0x8b, 0x08, 0x00}
	if err := client.Import(context.Background(), bytes.NewReader(payload), int64(len(payload)), ""); err != nil {
		t.Fatalf("Import: %v", err)
	}
}

func TestClientImportWithDatabaseHeader(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-DBACK-DATABASE"); got != "staging_db" {
			http.Error(w, "missing database header", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success":             true,
			"statements_executed": float64(3),
			"database":            "staging_db",
		})
	}))
	defer server.Close()

	client, err := NewClient(models.Profile{
		ConnectionType: models.ConnectionTypeWordPress,
		WPUrl:          server.URL,
		WPKey:          "secret-key",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	payload := []byte{0x1f, 0x8b, 0x08}
	if err := client.Import(context.Background(), bytes.NewReader(payload), int64(len(payload)), "staging_db"); err != nil {
		t.Fatalf("Import: %v", err)
	}
}

func TestValidateImportResponseRejectsZeroStatements(t *testing.T) {
	t.Parallel()
	if err := validateImportResponse(map[string]interface{}{
		"success":             true,
		"statements_executed": float64(0),
		"bytes_received":      float64(128),
	}); err == nil {
		t.Fatal("expected zero-statement import to fail")
	}
}

func TestBuildPluginZip(t *testing.T) {
	t.Parallel()

	data, filename, err := BuildPluginZip("https://shop.example.com", "generated-token")
	if err != nil {
		t.Fatalf("BuildPluginZip: %v", err)
	}
	if !strings.HasPrefix(filename, "dback-shop.example.com-") || !strings.HasSuffix(filename, ".zip") {
		t.Fatalf("unexpected filename: %q", filename)
	}
	if len(data) < 128 {
		t.Fatalf("zip too small: %d bytes", len(data))
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("NewReader: %v", err)
	}
	var mainPHP string
	for _, f := range zr.File {
		if f.Name == "dback-db-tools/dback-db-tools.php" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("open plugin main file: %v", err)
			}
			body, err := io.ReadAll(rc)
			_ = rc.Close()
			if err != nil {
				t.Fatalf("read plugin main file: %v", err)
			}
			mainPHP = string(body)
			break
		}
	}
	if !strings.Contains(mainPHP, "generated-token") {
		t.Fatalf("expected generated token in plugin main file")
	}
}

func TestHostnameFromSiteURL(t *testing.T) {
	t.Parallel()
	if got := hostnameFromSiteURL("https://My.Site.com/wp"); got != "my.site.com" {
		t.Fatalf("got %q", got)
	}
}
