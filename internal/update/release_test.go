package update

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchLatestAndCheck(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/devlifeX/dback/releases/latest" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"tag_name": "v3.6.2",
			"name":     "v3.6.2",
			"body":     "Bug fixes",
			"html_url": "https://github.com/devlifeX/dback/releases/tag/v3.6.2",
			"assets": []map[string]interface{}{
				{
					"name":                 "dback_3.6.2_amd64.deb",
					"browser_download_url": "https://example.com/dback_3.6.2_amd64.deb",
					"size":                 1234,
				},
			},
		})
	}))
	defer server.Close()

	client := server.Client()
	origURL := latestReleaseAPIURL
	t.Cleanup(func() {
		latestReleaseAPIURL = origURL
	})
	latestReleaseAPIURL = server.URL + "/repos/devlifeX/dback/releases/latest"

	info, err := Check(context.Background(), client, "DBack/3.6.1", "3.6.1")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !info.Available {
		t.Fatal("expected update available")
	}
	if info.LatestVersion != "3.6.2" {
		t.Fatalf("latest=%q", info.LatestVersion)
	}
	if info.Asset.Name != "dback_3.6.2_amd64.deb" {
		t.Fatalf("asset=%q", info.Asset.Name)
	}
}

func TestCheckUpToDate(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"tag_name": "v3.6.1",
			"html_url": "https://github.com/devlifeX/dback/releases/tag/v3.6.1",
			"assets":   []map[string]interface{}{},
		})
	}))
	defer server.Close()

	origURL := latestReleaseAPIURL
	t.Cleanup(func() {
		latestReleaseAPIURL = origURL
	})
	latestReleaseAPIURL = server.URL + "/repos/devlifeX/dback/releases/latest"

	info, err := Check(context.Background(), server.Client(), "DBack/3.6.1", "3.6.1")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if info.Available {
		t.Fatal("expected no update")
	}
}
