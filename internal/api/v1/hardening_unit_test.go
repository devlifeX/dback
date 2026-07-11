package v1

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestIPLimiter(t *testing.T) {
	lim := newIPLimiter(1, 1)
	if !lim.allow("127.0.0.1") {
		t.Fatal("first request should pass")
	}
	if lim.allow("127.0.0.1") {
		t.Fatal("second request should be limited")
	}
}

func TestRateLimitMiddlewareUnit(t *testing.T) {
	mw := rateLimitMiddleware(1, 1)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	srv := httptest.NewServer(h)
	defer srv.Close()

	var wg sync.WaitGroup
	statuses := make([]int, 6)
	for i := range statuses {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resp, err := http.Get(srv.URL)
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
		t.Fatalf("expected 429s, got %v", statuses)
	}
}
