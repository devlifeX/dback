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
	URL         string
	TTFBMs      int64
	StatusCode  int
	OK          bool
	ViaProxy    bool
	ProxyID     string
	SourceLabel string
	CountryCode string
	Error       string
}

type checkEndpoint struct {
	proxyURL    string
	proxyID     string
	sourceLabel string
	countryCode string
	viaProxy    bool
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

func (a *App) buildCheckEndpoints(profile models.Profile) ([]checkEndpoint, error) {
	settings, err := a.store.GetSquidSettings()
	if err != nil {
		return nil, err
	}
	directLabel := strings.TrimSpace(settings.PrimaryHostCountry)
	if directLabel == "" {
		directLabel = "Local server"
	}
	endpoints := []checkEndpoint{{
		sourceLabel: directLabel,
		countryCode: strings.ToUpper(strings.TrimSpace(settings.PrimaryHostCountryCode)),
		viaProxy:    false,
	}}

	if len(profile.URLCheck.ProxyIDs) == 0 {
		return endpoints, nil
	}

	all, err := a.store.ListSquidProxies()
	if err != nil {
		return nil, err
	}
	byID := map[string]models.SquidProxy{}
	for _, p := range all {
		if p.Enabled {
			byID[p.ID] = p
		}
	}
	for _, id := range profile.URLCheck.ProxyIDs {
		if id == "" {
			continue
		}
		p, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("squid proxy %q not found or disabled", id)
		}
		endpoints = append(endpoints, checkEndpoint{
			proxyURL:    strings.TrimSpace(p.URL),
			proxyID:     p.ID,
			sourceLabel: strings.TrimSpace(p.Country),
			countryCode: strings.ToUpper(strings.TrimSpace(p.CountryCode)),
			viaProxy:    true,
		})
	}
	return endpoints, nil
}

func (a *App) CheckURLs(ctx context.Context, profileID string, operationID string, params operation.UrlCheckerParams) ([]URLCheckOutcome, error) {
	profile, err := a.profileByID(profileID)
	if err != nil {
		return nil, err
	}
	if operationID == "" {
		operationID = newID()
	}
	targets, err := urlsForCheck(profile, params.URLIndex)
	if err != nil {
		return nil, err
	}
	endpoints, err := a.buildCheckEndpoints(profile)
	if err != nil {
		return nil, err
	}

	timeout := defaultURLCheckTimeout
	if params.Timeout > 0 {
		timeout = time.Duration(params.Timeout) * time.Second
	}

	var failures []error
	var outcomes []URLCheckOutcome
	for _, target := range targets {
		for _, ep := range endpoints {
			outcome := probeURL(ctx, target, ep, timeout)
			outcomes = append(outcomes, outcome)
			sample := models.URLCheckSample{
				ID:          newID(),
				ProfileID:   profileID,
				URL:         target,
				TS:          time.Now().UTC().Format(time.RFC3339Nano),
				TTFBMs:      outcome.TTFBMs,
				StatusCode:  outcome.StatusCode,
				OK:          outcome.OK,
				ViaProxy:    outcome.ViaProxy,
				ProxyID:     outcome.ProxyID,
				SourceLabel: outcome.SourceLabel,
				CountryCode: outcome.CountryCode,
				Error:       outcome.Error,
			}
			_ = a.store.AppendURLCheckSample(sample)
			details := fmt.Sprintf("%s %s: HTTP %d, TTFB %dms", outcome.SourceLabel, target, outcome.StatusCode, outcome.TTFBMs)
			status := "Succeeded"
			errText := ""
			if !outcome.OK {
				status = "Failed"
				errText = outcome.Error
				if errText == "" {
					errText = fmt.Sprintf("HTTP %d", outcome.StatusCode)
				}
			}
			a.logPhase(operationID, &profile, "URL check", "probe", "", 0, details, "Info", status, errText)
			if !outcome.OK {
				msg := outcome.Error
				if msg == "" {
					msg = fmt.Sprintf("%s: %s returned HTTP %d", outcome.SourceLabel, target, outcome.StatusCode)
				}
				failures = append(failures, fmt.Errorf("%s", msg))
			}
		}
	}
	if len(failures) > 0 {
		return outcomes, errors.Join(failures...)
	}
	return outcomes, nil
}

func probeURL(ctx context.Context, rawURL string, ep checkEndpoint, timeout time.Duration) URLCheckOutcome {
	out := URLCheckOutcome{
		URL:         rawURL,
		ViaProxy:    ep.viaProxy,
		ProxyID:     ep.proxyID,
		SourceLabel: ep.sourceLabel,
		CountryCode: ep.countryCode,
	}
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
	if ep.viaProxy && ep.proxyURL != "" {
		proxyURL, perr := url.Parse(ep.proxyURL)
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
