package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Asset struct {
	Name string
	URL  string
	Size int64
}

type Release struct {
	TagName string
	Version string
	Name    string
	Body    string
	HTMLURL string
	Assets  []Asset
}

type Info struct {
	Available      bool
	CurrentVersion string
	LatestVersion  string
	ReleaseNotes   string
	ReleaseURL     string
	Asset          Asset
}

func Check(ctx context.Context, client *http.Client, userAgent, currentVersion string) (Info, error) {
	currentVersion = strings.TrimSpace(currentVersion)
	if currentVersion == "" {
		currentVersion = "0.0.0"
	}

	release, err := FetchLatest(ctx, client, userAgent)
	if err != nil {
		return Info{CurrentVersion: currentVersion}, err
	}

	info := Info{
		CurrentVersion: currentVersion,
		LatestVersion:  release.Version,
		ReleaseNotes:   release.Body,
		ReleaseURL:     release.HTMLURL,
		Available:      IsNewer(currentVersion, release.Version),
	}

	if !info.Available {
		return info, nil
	}

	asset, err := PickAsset(release)
	if err != nil {
		return info, err
	}
	info.Asset = asset
	return info, nil
}

func FetchLatest(ctx context.Context, client *http.Client, userAgent string) (Release, error) {
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, LatestReleaseAPIURL(), nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Release{}, err
	}
	if resp.StatusCode >= 400 {
		return Release{}, fmt.Errorf("github releases API HTTP %d: %s", resp.StatusCode, truncate(string(body), 300))
	}

	var payload struct {
		TagName     string `json:"tag_name"`
		Name        string `json:"name"`
		Body        string `json:"body"`
		HTMLURL     string `json:"html_url"`
		Assets      []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
			Size               int64  `json:"size"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Release{}, fmt.Errorf("parse github release: %w", err)
	}

	release := Release{
		TagName: payload.TagName,
		Version: NormalizeVersion(payload.TagName),
		Name:    payload.Name,
		Body:    strings.TrimSpace(payload.Body),
		HTMLURL: payload.HTMLURL,
	}
	for _, asset := range payload.Assets {
		release.Assets = append(release.Assets, Asset{
			Name: asset.Name,
			URL:  asset.BrowserDownloadURL,
			Size: asset.Size,
		})
	}
	if release.Version == "0.0.0" && release.TagName == "" {
		return Release{}, fmt.Errorf("github release response missing tag_name")
	}
	return release, nil
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
