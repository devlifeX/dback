package v1_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apiv1 "dback/internal/api/v1"
	"dback/internal/app"
	"dback/internal/config"
	"dback/internal/controlplane"
	"dback/models"
)

func testHandler(t *testing.T, token string) *apiv1.Handler {
	t.Helper()
	dir := t.TempDir()
	a, err := app.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.CreateVault("test-pass"); err != nil {
		t.Fatal(err)
	}
	cp := controlplane.NewService(a, 32, 2)
	return &apiv1.Handler{
		App: a,
		CP:  cp,
		Cfg: config.Config{APIToken: token},
	}
}

func TestAuthRequired(t *testing.T) {
	h := testHandler(t, "secret-token")
	srv := httptest.NewServer(h.Router())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/operations")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestVersionPublic(t *testing.T) {
	h := testHandler(t, "secret-token")
	srv := httptest.NewServer(h.Router())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/version")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestListHostsRedactsSecrets(t *testing.T) {
	h := testHandler(t, "secret-token")
	if err := h.App.SaveProfile(models.Profile{
		ID:             "p1",
		Name:           "local",
		ConnectionType: models.ConnectionTypeLocalhost,
		SSHPassword:    "secret",
		Host:           "localhost",
	}); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h.Router())
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/hosts", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 {
		t.Fatalf("expected 1 host, got %d", len(body.Items))
	}
	if pw, _ := body.Items[0]["ssh_password"].(string); pw != "" {
		t.Fatalf("expected redacted ssh_password, got %q", pw)
	}
}

func TestPreconditionFailed(t *testing.T) {
	h := testHandler(t, "secret-token")
	srv := httptest.NewServer(h.Router())
	defer srv.Close()

	body := []byte(`{"name":"t1","profile_ids":["p1"],"trigger":{"type":"interval","interval":{"every":3600000000000}},"actions":[{"operation":"backup_db"}]}`)
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/tasks", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", `W/"99999"`)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected 412, got %d", resp.StatusCode)
	}
}

func TestSystemServerInfo(t *testing.T) {
	h := testHandler(t, "secret-token")
	srv := httptest.NewServer(h.Router())
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/system/server-info", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if cpu, ok := body["cpu_count"].(float64); !ok || cpu < 1 {
		t.Fatalf("cpu_count = %v", body["cpu_count"])
	}
	if _, ok := body["disk_free_bytes"]; !ok {
		t.Fatal("missing disk_free_bytes")
	}
	if host, ok := body["internet_host"].(string); !ok || host != "google.com" {
		t.Fatalf("internet_host = %v", body["internet_host"])
	}
}
