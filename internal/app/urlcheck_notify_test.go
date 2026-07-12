package app

import (
	"strings"
	"testing"
	"time"

	"dback/models"
)

func TestFormatURLCheckNotifyReportIncludesRunAndDailyStats(t *testing.T) {
	dir := t.TempDir()
	a, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.CreateVault("test-pass"); err != nil {
		t.Fatal(err)
	}
	profile := models.Profile{ID: "p1", Name: "Example Host", ConnectionType: models.ConnectionTypeLocalhost}
	if err := a.SaveProfile(profile); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		if err := a.store.AppendURLCheckSample(models.URLCheckSample{
			ProfileID:   "p1",
			URL:         "https://example.com",
			TS:          now.Add(-time.Duration(i) * time.Hour).Format(time.RFC3339Nano),
			TTFBMs:      100 + int64(i*10),
			StatusCode:  200,
			OK:          true,
			SourceLabel: "Direct",
		}); err != nil {
			t.Fatal(err)
		}
	}

	run := []URLCheckOutcome{{
		URL:         "https://example.com",
		TTFBMs:      95,
		StatusCode:  200,
		OK:          true,
		SourceLabel: "Direct",
	}}
	report, err := a.FormatURLCheckNotifyReport("p1", run)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(report, "Host: Example Host") {
		t.Fatalf("missing host name: %q", report)
	}
	if !strings.Contains(report, "=== This run ===") || !strings.Contains(report, "HTTP 200") {
		t.Fatalf("missing this run section: %q", report)
	}
	if !strings.Contains(report, "=== Last 24 hours ===") || !strings.Contains(report, "samples") {
		t.Fatalf("missing daily stats: %q", report)
	}
}
