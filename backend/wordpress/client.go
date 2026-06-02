package wordpress

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"dback/backend/db"
	"dback/models"
)

const restNamespace = "dback/v1"

type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func NewClient(p models.Profile) (*Client, error) {
	if err := db.ValidateProfileForWordPress(p); err != nil {
		return nil, err
	}
	base := normalizeSiteURL(p.WPUrl)
	if base == "" {
		base = normalizeSiteURL(p.Host)
	}
	return &Client{
		baseURL: base,
		apiKey:  strings.TrimSpace(p.WPKey),
		http: &http.Client{
			Timeout: 0,
		},
	}, nil
}

func normalizeSiteURL(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimRight(raw, "/")
	return raw
}

func (c *Client) endpoint(path string) string {
	return c.baseURL + "/wp-json/" + restNamespace + path
}

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint(path), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-DBACK-KEY", c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (c *Client) Ping(ctx context.Context) (map[string]interface{}, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/ping", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return decodeJSONResponse(resp)
}

func (c *Client) Preflight(ctx context.Context) (PreflightResult, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/preflight", nil)
	if err != nil {
		return PreflightResult{}, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return PreflightResult{}, err
	}
	defer resp.Body.Close()
	data, err := decodeJSONResponse(resp)
	if err != nil {
		return PreflightResult{}, err
	}
	return parsePreflightResult(data), nil
}

func (c *Client) Export(ctx context.Context) (io.ReadCloser, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/export", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/gzip")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		return nil, parseHTTPError(resp)
	}
	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "json") {
		defer resp.Body.Close()
		return nil, parseHTTPError(resp)
	}
	return resp.Body, nil
}

func (c *Client) applyDatabaseHeader(req *http.Request, database string) error {
	database = strings.TrimSpace(database)
	if database == "" {
		return nil
	}
	if err := db.ValidateWordPressDatabaseName(database); err != nil {
		return err
	}
	req.Header.Set("X-DBACK-DATABASE", database)
	return nil
}

func (c *Client) Import(ctx context.Context, r io.Reader, size int64, database string) error {
	database = strings.TrimSpace(database)
	if err := db.ValidateWordPressDatabaseName(database); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint("/import"), r)
	if err != nil {
		return err
	}
	req.Header.Set("X-DBACK-KEY", c.apiKey)
	req.Header.Set("Content-Type", "application/gzip")
	if err := c.applyDatabaseHeader(req, database); err != nil {
		return err
	}
	if size > 0 {
		req.ContentLength = size
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return parseHTTPError(resp)
	}
	data, err := decodeJSONResponse(resp)
	if err != nil {
		return err
	}
	return validateImportResponse(data)
}

func validateImportResponse(data map[string]interface{}) error {
	executed, ok := data["statements_executed"].(float64)
	if !ok {
		return fmt.Errorf("wordpress import response missing statements_executed")
	}
	if executed <= 0 {
		bytesReceived, _ := data["bytes_received"].(float64)
		if bytesReceived > 0 {
			return fmt.Errorf("wordpress import received %.0f bytes but executed 0 SQL statements", bytesReceived)
		}
		return fmt.Errorf("wordpress import executed 0 SQL statements")
	}
	return nil
}

func (c *Client) Query(ctx context.Context, sql, database string) (db.QueryResult, error) {
	payload, err := json.Marshal(map[string]string{"sql": sql})
	if err != nil {
		return db.QueryResult{}, err
	}
	req, err := c.newRequest(ctx, http.MethodPost, "/query", bytes.NewReader(payload))
	if err != nil {
		return db.QueryResult{}, err
	}
	if err := c.applyDatabaseHeader(req, database); err != nil {
		return db.QueryResult{}, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return db.QueryResult{}, err
	}
	defer resp.Body.Close()
	data, err := decodeJSONResponse(resp)
	if err != nil {
		return db.QueryResult{}, err
	}
	return queryResultFromJSON(data), nil
}

func decodeJSONResponse(resp *http.Response) (map[string]interface{}, error) {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, parseErrorBody(resp.StatusCode, body)
	}
	var data map[string]interface{}
	if len(body) == 0 {
		return map[string]interface{}{"success": true}, nil
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("wordpress API returned invalid JSON (HTTP %d): %s", resp.StatusCode, truncate(string(body), 300))
	}
	if code, ok := data["code"].(string); ok && code != "" {
		msg, _ := data["message"].(string)
		if msg == "" {
			msg = code
		}
		return nil, fmt.Errorf("wordpress API error (%s): %s", code, msg)
	}
	return data, nil
}

func parseHTTPError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return parseErrorBody(resp.StatusCode, body)
}

func parseErrorBody(status int, body []byte) error {
	if len(body) == 0 {
		return fmt.Errorf("wordpress API HTTP %d", status)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err == nil {
		if code, ok := data["code"].(string); ok {
			msg, _ := data["message"].(string)
			if msg == "" {
				msg = code
			}
			return fmt.Errorf("wordpress API error (%s): %s", code, msg)
		}
	}
	return fmt.Errorf("wordpress API HTTP %d: %s", status, truncate(string(body), 300))
}

func queryResultFromJSON(data map[string]interface{}) db.QueryResult {
	queryType, _ := data["type"].(string)
	if queryType == "command" {
		affected := formatNumber(data["affected_rows"])
		return db.QueryResult{
			Columns: []string{"Affected rows"},
			Rows:    [][]string{{affected}},
			Message: fmt.Sprintf("Query executed (%s)", data["query_type"]),
		}
	}

	columns := stringSliceFromJSON(data["columns"])
	rowsRaw, _ := data["rows"].([]interface{})
	var rows [][]string
	for _, row := range rowsRaw {
		rowMap, ok := row.(map[string]interface{})
		if !ok {
			continue
		}
		line := make([]string, len(columns))
		for i, col := range columns {
			if v, ok := rowMap[col]; ok {
				line[i] = fmt.Sprint(v)
			}
		}
		rows = append(rows, line)
	}
	return db.QueryResult{
		Columns: columns,
		Rows:    rows,
		Message: fmt.Sprintf("%d row(s)", len(rows)),
	}
}

func stringSliceFromJSON(v interface{}) []string {
	raw, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		out = append(out, fmt.Sprint(item))
	}
	return out
}

func formatNumber(v interface{}) string {
	switch n := v.(type) {
	case float64:
		return fmt.Sprintf("%.0f", n)
	case json.Number:
		return n.String()
	default:
		return fmt.Sprint(v)
	}
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
