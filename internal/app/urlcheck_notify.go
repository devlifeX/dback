package app

import (
	"fmt"
	"math"
	"strings"
	"time"
)

func (a *App) FormatURLCheckNotifyReport(profileID string, run []URLCheckOutcome) (string, error) {
	profile, err := a.profileByID(profileID)
	if err != nil {
		return "", err
	}
	hostName := strings.TrimSpace(profile.Name)
	if hostName == "" {
		hostName = profileID
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Host: %s\n", hostName))

	if len(run) > 0 {
		b.WriteString("\n=== This run ===\n")
		for _, o := range run {
			status := "OK"
			if !o.OK {
				status = "FAIL"
			}
			line := fmt.Sprintf("• %s %s → HTTP %d, TTFB %dms %s", o.SourceLabel, o.URL, o.StatusCode, o.TTFBMs, status)
			if o.Error != "" {
				line += " (" + o.Error + ")"
			}
			b.WriteString(line + "\n")
		}
	}

	now := time.Now().UTC()
	from := now.Add(-24 * time.Hour)
	buckets, err := a.ListURLCheckHourly(profileID, "", from, now)
	if err != nil {
		return b.String(), nil
	}
	if len(buckets) == 0 {
		return b.String(), nil
	}

	b.WriteString("\n=== Last 24 hours ===\n")
	type dayKey struct {
		url, source string
	}
	agg := map[dayKey]struct {
		sumTTFB float64
		samples int
		ok      int
		fail    int
	}{}
	for _, bk := range buckets {
		k := dayKey{url: bk.URL, source: bk.SourceLabel}
		a := agg[k]
		a.sumTTFB += bk.AvgTTFBMs * float64(bk.Samples)
		a.samples += bk.Samples
		a.ok += bk.OKCount
		a.fail += bk.FailCount
		agg[k] = a
	}
	for k, a := range agg {
		avg := int64(0)
		if a.samples > 0 {
			avg = int64(math.Round(a.sumTTFB / float64(a.samples)))
		}
		okPct := int64(0)
		if a.samples > 0 {
			okPct = int64(math.Round(float64(a.ok) * 100 / float64(a.samples)))
		}
		b.WriteString(fmt.Sprintf(
			"• %s %s: avg %dms, %d samples, %d%% OK\n",
			k.source, k.url, avg, a.samples, okPct,
		))
	}
	return b.String(), nil
}
