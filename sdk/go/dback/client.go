package dback

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	apiv1 "dback/internal/api/v1"
)

const defaultTimeout = 30 * time.Second

type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

func (c *Client) Version(ctx context.Context) (map[string]string, error) {
	var out map[string]string
	err := c.get(ctx, "/api/v1/version", &out)
	return out, err
}

func (c *Client) ListOperations(ctx context.Context) (apiv1.Paginated, error) {
	var out apiv1.Paginated
	err := c.get(ctx, "/api/v1/operations", &out)
	return out, err
}

func (c *Client) CreateOperation(ctx context.Context, kind, profileID string, params json.RawMessage) (apiv1.OperationDTO, error) {
	body := map[string]any{
		"kind":       kind,
		"profile_id": profileID,
	}
	if len(params) > 0 {
		body["params"] = json.RawMessage(params)
	}
	var out apiv1.OperationDTO
	err := c.post(ctx, "/api/v1/operations", body, &out)
	return out, err
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *Client) post(ctx context.Context, path string, body any, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, strings.NewReader(string(b)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, out)
}

func (c *Client) do(req *http.Request, out any) error {
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultTimeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		var eb apiv1.ErrorBody
		if json.Unmarshal(raw, &eb) == nil && eb.Message != "" {
			return fmt.Errorf("api %s: %s", eb.Code, eb.Message)
		}
		return fmt.Errorf("api status %d: %s", resp.StatusCode, string(raw))
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return err
		}
	}
	return nil
}
