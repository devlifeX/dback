package v1_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	apiv1 "dback/internal/api/v1"
	"dback/internal/app"
	"dback/internal/config"
	"dback/internal/controlplane"
)

func TestSecurityHeaders(t *testing.T) {
	dir := t.TempDir()
	a, err := app.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.CreateVault("test-pass"); err != nil {
		t.Fatal(err)
	}
	h := &apiv1.Handler{App: a, CP: controlplane.NewService(a, 8, 2), Cfg: config.Config{APIToken: "t"}}
	srv := httptest.NewServer(h.Router())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health/live")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing security header")
	}
}

func TestRateLimit(t *testing.T) {
	dir := t.TempDir()
	a, err := app.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.CreateVault("test-pass"); err != nil {
		t.Fatal(err)
	}
	h := &apiv1.Handler{
		App: a,
		CP:  controlplane.NewService(a, 8, 2),
		Cfg: config.Config{APIToken: "secret", RateLimitRPS: 1, RateLimitBurst: 1},
	}
	srv := httptest.NewServer(h.Router())
	defer srv.Close()

	var wg sync.WaitGroup
	statuses := make([]int, 8)
	for i := range statuses {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/hosts", nil)
			req.Header.Set("Authorization", "Bearer secret")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Error(err)
				return
			}
			statuses[i] = resp.StatusCode
			resp.Body.Close()
		}(i)
	}
	wg.Wait()

	var limited int
	for _, code := range statuses {
		if code == http.StatusTooManyRequests {
			limited++
		}
	}
	if limited == 0 {
		t.Fatalf("expected at least one 429, got statuses %v", statuses)
	}
}
