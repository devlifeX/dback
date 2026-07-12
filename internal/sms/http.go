package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultHTTPTimeout = 10 * time.Second
const maxResponseBytes = 1 << 20

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

func DefaultHTTPClient() *http.Client {
	return &http.Client{Timeout: defaultHTTPTimeout}
}

func doJSONRequest(ctx context.Context, client HTTPDoer, method, url string, headers map[string]string, payload any) error {
	if client == nil {
		client = DefaultHTTPClient()
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		if isAllowedHeader(k) {
			req.Header.Set(k, v)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBytes))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}

func doGETRequest(ctx context.Context, client HTTPDoer, url string) error {
	if client == nil {
		client = DefaultHTTPClient()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBytes))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}

func isAllowedHeader(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "content-type", "accept", "user-agent", "x-request-id":
		return true
	default:
		return strings.HasPrefix(strings.ToLower(name), "x-dback-")
	}
}
