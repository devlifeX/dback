package sqlstore

import (
	"database/sql"
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
	if sample.SourceLabel == "" {
		if sample.ViaProxy {
			sample.SourceLabel = "Proxy"
		} else {
			sample.SourceLabel = "Direct"
		}
	}
	_, err := s.db.Exec(
		`INSERT INTO url_check_samples (id, profile_id, url, ts, ttfb_ms, status_code, ok, via_proxy, error_text, proxy_id, source_label, country_code) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sample.ID,
		sample.ProfileID,
		sample.URL,
		sample.TS,
		sample.TTFBMs,
		sample.StatusCode,
		boolToInt(sample.OK),
		boolToInt(sample.ViaProxy),
		sample.Error,
		sample.ProxyID,
		sample.SourceLabel,
		sample.CountryCode,
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

	hasSourceCols := columnExists(s.db, s.driver, "url_check_samples", "source_label")

	var query string
	if hasSourceCols {
		query = `SELECT url, ts, ttfb_ms, status_code, ok, proxy_id, source_label, country_code FROM url_check_samples WHERE profile_id = ?`
	} else {
		query = `SELECT url, ts, ttfb_ms, status_code, ok, via_proxy, '', '' FROM url_check_samples WHERE profile_id = ?`
	}
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
		sumTTFB    int64
		minTTFB    int64
		maxTTFB    int64
		samples    int
		okCount    int
		failCount  int
		lastStatus int
		proxyID    string
		country    string
	}
	// hour -> url -> source_label -> acc
	buckets := map[string]map[string]map[string]*acc{}

	for rows.Next() {
		var url string
		var ts string
		var ttfb int64
		var status int
		var ok int
		var proxyID, sourceLabel, countryCode string
		if hasSourceCols {
			if err := rows.Scan(&url, &ts, &ttfb, &status, &ok, &proxyID, &sourceLabel, &countryCode); err != nil {
				return nil, err
			}
		} else {
			var viaProxy int
			if err := rows.Scan(&url, &ts, &ttfb, &status, &ok, &viaProxy, &proxyID, &countryCode); err != nil {
				return nil, err
			}
			if viaProxy != 0 {
				sourceLabel = "Proxy"
			} else {
				sourceLabel = "Direct"
			}
		}
		if strings.TrimSpace(sourceLabel) == "" {
			sourceLabel = "Direct"
		}
		parsed, err := time.Parse(time.RFC3339Nano, ts)
		if err != nil {
			continue
		}
		hour := parsed.UTC().Truncate(time.Hour).Format(time.RFC3339)
		if buckets[hour] == nil {
			buckets[hour] = map[string]map[string]*acc{}
		}
		if buckets[hour][url] == nil {
			buckets[hour][url] = map[string]*acc{}
		}
		a := buckets[hour][url][sourceLabel]
		if a == nil {
			a = &acc{minTTFB: ttfb, maxTTFB: ttfb, proxyID: proxyID, country: countryCode}
			buckets[hour][url][sourceLabel] = a
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
		for url, bySource := range byURL {
			for sourceLabel, a := range bySource {
				avg := float64(0)
				if a.samples > 0 {
					avg = float64(a.sumTTFB) / float64(a.samples)
				}
				out = append(out, models.URLCheckHourlyBucket{
					Hour:        hour,
					URL:         url,
					SourceLabel: sourceLabel,
					ProxyID:     a.proxyID,
					CountryCode: a.country,
					AvgTTFBMs:   avg,
					MinTTFBMs:   a.minTTFB,
					MaxTTFBMs:   a.maxTTFB,
					Samples:     a.samples,
					OKCount:     a.okCount,
					FailCount:   a.failCount,
					LastStatus:  a.lastStatus,
				})
			}
		}
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if strings.Compare(out[i].Hour, out[j].Hour) > 0 ||
				(out[i].Hour == out[j].Hour && strings.Compare(out[i].SourceLabel, out[j].SourceLabel) > 0) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

func columnExists(db *sql.DB, driver, table, column string) bool {
	if driver == "mysql" {
		var count int
		err := db.QueryRow(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`, table, column).Scan(&count)
		return err == nil && count > 0
	}
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, table, column).Scan(&count)
	return err == nil && count > 0
}
