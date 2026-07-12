package sqlstore

import (
	"fmt"
	"strings"
	"time"

	"dback/models"
)

const urlCheckRetentionDays = 30

func (s *Store) AppendURLCheckSample(sample models.URLCheckSample) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return err
	}
	if err := s.ensureDB(); err != nil {
		return err
	}
	if sample.ID == "" {
		sample.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	_, err := s.db.Exec(
		`INSERT INTO url_check_samples (id, profile_id, url, ts, ttfb_ms, status_code, ok, via_proxy, error_text) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sample.ID,
		sample.ProfileID,
		sample.URL,
		sample.TS,
		sample.TTFBMs,
		sample.StatusCode,
		boolToInt(sample.OK),
		boolToInt(sample.ViaProxy),
		sample.Error,
	)
	if err != nil {
		return err
	}
	cutoff := time.Now().UTC().Add(-urlCheckRetentionDays * 24 * time.Hour).Format(time.RFC3339Nano)
	_, _ = s.db.Exec(`DELETE FROM url_check_samples WHERE ts < ?`, cutoff)
	return nil
}

func (s *Store) ListURLCheckHourly(profileID, urlFilter string, from, to time.Time) ([]models.URLCheckHourlyBucket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireUnlocked(); err != nil {
		return nil, err
	}
	if err := s.ensureDB(); err != nil {
		return nil, err
	}
	if !tableExists(s.db, s.driver, "url_check_samples") {
		return nil, nil
	}

	query := `SELECT url, ts, ttfb_ms, status_code, ok FROM url_check_samples WHERE profile_id = ?`
	args := []any{profileID}
	if urlFilter != "" {
		query += ` AND url = ?`
		args = append(args, urlFilter)
	}
	if !from.IsZero() {
		query += ` AND ts >= ?`
		args = append(args, from.UTC().Format(time.RFC3339Nano))
	}
	if !to.IsZero() {
		query += ` AND ts <= ?`
		args = append(args, to.UTC().Format(time.RFC3339Nano))
	}
	query += ` ORDER BY ts`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type acc struct {
		sumTTFB   int64
		minTTFB   int64
		maxTTFB   int64
		samples   int
		okCount   int
		failCount int
		lastStatus int
	}
	buckets := map[string]map[string]*acc{}

	for rows.Next() {
		var url string
		var ts string
		var ttfb int64
		var status int
		var ok int
		if err := rows.Scan(&url, &ts, &ttfb, &status, &ok); err != nil {
			return nil, err
		}
		parsed, err := time.Parse(time.RFC3339Nano, ts)
		if err != nil {
			continue
		}
		hour := parsed.UTC().Truncate(time.Hour).Format(time.RFC3339)
		if buckets[hour] == nil {
			buckets[hour] = map[string]*acc{}
		}
		a := buckets[hour][url]
		if a == nil {
			a = &acc{minTTFB: ttfb, maxTTFB: ttfb}
			buckets[hour][url] = a
		}
		a.sumTTFB += ttfb
		a.samples++
		if ttfb < a.minTTFB {
			a.minTTFB = ttfb
		}
		if ttfb > a.maxTTFB {
			a.maxTTFB = ttfb
		}
		if ok != 0 {
			a.okCount++
		} else {
			a.failCount++
		}
		a.lastStatus = status
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var out []models.URLCheckHourlyBucket
	for hour, byURL := range buckets {
		for url, a := range byURL {
			avg := float64(0)
			if a.samples > 0 {
				avg = float64(a.sumTTFB) / float64(a.samples)
			}
			out = append(out, models.URLCheckHourlyBucket{
				Hour:       hour,
				URL:        url,
				AvgTTFBMs:  avg,
				MinTTFBMs:  a.minTTFB,
				MaxTTFBMs:  a.maxTTFB,
				Samples:    a.samples,
				OKCount:    a.okCount,
				FailCount:  a.failCount,
				LastStatus: a.lastStatus,
			})
		}
	}
	// stable sort by hour then url
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if strings.Compare(out[i].Hour, out[j].Hour) > 0 ||
				(out[i].Hour == out[j].Hour && strings.Compare(out[i].URL, out[j].URL) > 0) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}
