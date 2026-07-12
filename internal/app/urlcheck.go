package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"time"

	"dback/internal/operation"
	"dback/models"
)

const defaultURLCheckTimeout = 30 * time.Second

type URLCheckOutcome struct {
	URL        string
	TTFBMs     int64
	StatusCode int
	OK         bool
	ViaProxy   bool
	Error      string
}

func urlsForCheck(profile models.Profile, index *int) ([]string, error) {
	primary := strings.TrimSpace(profile.URLCheck.Primary.URL)
	var secondaries []string
	for _, t := range profile.URLCheck.Secondary {
		u := strings.TrimSpace(t.URL)
		if u != "" {
			secondaries = append(secondaries, u)
		}
	}

	if index == nil {
		if primary == "" {
			return nil, fmt.Errorf("no primary URL configured")
		}
		return []string{primary}, nil
	}

	switch *index {
	case -1:
		var all []string
		if primary != "" {
			all = append(all, primary)
		}
		all = append(all, secondaries...)
		if len(all) == 0 {
			return nil, fmt.Errorf("no URLs configured")
		}
		return all, nil
	case 0:
		if primary == "" {
			return nil, fmt.Errorf("no primary URL configured")
		}
		return []string{primary}, nil
	default:
		i := *index - 1
		if i < 0 || i >= len(secondaries) {
			return nil, fmt.Errorf("secondary URL index %d out of range", *index)
		}
		return []string{secondaries[i]}, nil
	}
}

func (a *App) CheckURLs(ctx context.Context, profileID string, params operation.UrlCheckerParams) error {
	profile, err := a.profileByID(profileID)
	if err != nil {
		return err
	}
	targets, err := urlsForCheck(profile, params.URLIndex)
	if err != nil {
		return err
	}

	timeout := defaultURLCheckTimeout
	if params.Timeout > 0 {
		timeout = time.Duration(params.Timeout) * time.Second
	}
	useProxy := params.UseProxy && a.squidProxy != ""

	var outcomes []URLCheckOutcome
	var failures []error
	for _, target := range targets {
		outcome := probeURL(ctx, target, useProxy, a.squidProxy, timeout)
		outcomes = append(outcomes, outcome)
		sample := models.URLCheckSample{
			ID:         newID(),
			ProfileID:  profileID,
			URL:        target,
			TS:         time.Now().UTC().Format(time.RFC3339Nano),
			TTFBMs:     outcome.TTFBMs,
			StatusCode: outcome.StatusCode,
			OK:         outcome.OK,
			ViaProxy:   outcome.ViaProxy,
			Error:      outcome.Error,
		}
		_ = a.store.AppendURLCheckSample(sample)
		if !outcome.OK {
			msg := outcome.Error
			if msg == "" {
				msg = fmt.Sprintf("%s returned HTTP %d", target, outcome.StatusCode)
			}
			failures = append(failures, fmt.Errorf("%s", msg))
		}
	}
	if len(failures) > 0 {
		return errors.Join(failures...)
	}
	_ = outcomes
	return nil
}

func probeURL(ctx context.Context, rawURL string, useProxy bool, proxyAddr string, timeout time.Duration) URLCheckOutcome {
	out := URLCheckOutcome{URL: rawURL, ViaProxy: useProxy}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var gotFirstByte time.Time
	start := time.Now()
	trace := &httptrace.ClientTrace{
		GotFirstResponseByte: func() {
			if gotFirstByte.IsZero() {
				gotFirstByte = time.Now()
			}
		},
	}
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	client := &http.Client{Timeout: timeout}
	if useProxy && proxyAddr != "" {
		proxyURL, perr := url.Parse(proxyAddr)
		if perr != nil {
			out.Error = perr.Error()
			return out
		}
		client.Transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	}

	resp, err := client.Do(req)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if !gotFirstByte.IsZero() {
		out.TTFBMs = gotFirstByte.Sub(start).Milliseconds()
	} else {
		out.TTFBMs = time.Since(start).Milliseconds()
	}
	out.StatusCode = resp.StatusCode
	out.OK = resp.StatusCode >= 200 && resp.StatusCode < 300
	if !out.OK && out.Error == "" {
		out.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return out
}

func (a *App) ListURLCheckHourly(profileID, url string, from, to time.Time) ([]models.URLCheckHourlyBucket, error) {
	return a.store.ListURLCheckHourly(profileID, url, from, to)
}
