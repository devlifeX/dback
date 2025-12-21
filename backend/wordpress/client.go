package wordpress

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Client handles communication with the WordPress plugin
type Client struct {
	BaseURL string
	Key     string
	Client  *http.Client
}

// WPError represents an error response from WordPress REST API
type WPError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Status int `json:"status"`
	} `json:"data"`
}

// PingResponse represents the response from the ping endpoint
type PingResponse struct {
	Success           bool   `json:"success"`
	Message           string `json:"message"`
	Version           string `json:"version"`
	PHPVersion        string `json:"php_version"`
	WPVersion         string `json:"wp_version"`
	CanUseShell       bool   `json:"can_use_shell"`
	DBConnected       bool   `json:"db_connected"`
	UploadDirWritable bool   `json:"upload_dir_writable"`
	MemoryLimit       string `json:"memory_limit"`
	MaxExecutionTime  string `json:"max_execution_time"`
}

// NewClient creates a new WordPress client
func NewClient(url, key string) *Client {
	// Remove trailing slash from URL
	url = strings.TrimRight(url, "/")
	return &Client{
		BaseURL: url,
		Key:     key,
		Client: &http.Client{
			Timeout: 30 * time.Minute, // Long timeout for large DB exports
		},
	}
}

// Ping tests connectivity and returns server info
func (c *Client) Ping() (*PingResponse, error) {
	apiURL := fmt.Sprintf("%s/wp-json/dback/v1/ping", c.BaseURL)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-DBACK-KEY", c.Key)

	// Use shorter timeout for ping
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for error response
	if resp.StatusCode != http.StatusOK {
		var wpErr WPError
		if json.Unmarshal(body, &wpErr) == nil && wpErr.Code != "" {
			return nil, fmt.Errorf("API error [%s]: %s", wpErr.Code, wpErr.Message)
		}
		return nil, fmt.Errorf("ping failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Parse success response
	var pingResp PingResponse
	if err := json.Unmarshal(body, &pingResp); err != nil {
		return nil, fmt.Errorf("invalid response format: %s", string(body))
	}

	return &pingResp, nil
}

// Export triggers a DB dump on WP and streams it to destPath
func (c *Client) Export(destPath string, progressCallback func(int64)) error {
	// URL: /wp-json/dback/v1/export
	apiURL := fmt.Sprintf("%s/wp-json/dback/v1/export", c.BaseURL)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-DBACK-KEY", c.Key)

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check Content-Type first - WordPress plugin sends application/gzip for success
	contentType := resp.Header.Get("Content-Type")

	// If response is JSON, it's likely an error (even with 200 status)
	if strings.Contains(contentType, "application/json") {
		body, _ := io.ReadAll(resp.Body)
		var wpErr WPError
		if json.Unmarshal(body, &wpErr) == nil && wpErr.Code != "" {
			return fmt.Errorf("WordPress API error [%s]: %s", wpErr.Code, wpErr.Message)
		}
		// Fallback if not a structured WP error
		return fmt.Errorf("unexpected JSON response (status %d): %s", resp.StatusCode, string(body))
	}

	// Handle non-200 status codes
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		if bodyStr == "" {
			bodyStr = "(empty response)"
		}
		return fmt.Errorf("export failed (HTTP %d): %s", resp.StatusCode, bodyStr)
	}

	// Verify we're getting gzip content (expected for successful export)
	if !strings.Contains(contentType, "application/gzip") && !strings.Contains(contentType, "application/octet-stream") {
		body, _ := io.ReadAll(resp.Body)
		// Limit body preview to avoid huge error messages
		bodyPreview := string(body)
		if len(bodyPreview) > 500 {
			bodyPreview = bodyPreview[:500] + "..."
		}
		return fmt.Errorf("unexpected Content-Type '%s'. Expected 'application/gzip'. Response: %s", contentType, bodyPreview)
	}

	// Create local file
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer out.Close()

	// Copy with progress
	pr := &ProgressReader{
		Reader: resp.Body,
		Callback: func(curr int64) {
			if progressCallback != nil {
				progressCallback(curr)
			}
		},
	}

	written, err := io.Copy(out, pr)
	if err != nil {
		// Clean up partial file
		out.Close()
		os.Remove(destPath)
		return fmt.Errorf("failed to download: %w", err)
	}

	// Validate file size - a valid gzip SQL dump should be at least a few hundred bytes
	// 100 bytes is extremely small and likely indicates an error
	if written < 100 {
		// Read what we got to provide better error message
		out.Close()
		content, _ := os.ReadFile(destPath)
		os.Remove(destPath)
		return fmt.Errorf("export file too small (%d bytes). This usually indicates an error on the server. Content: %s", written, string(content))
	}

	return nil
}

// Import uploads a SQL dump to WP
func (c *Client) Import(sourcePath string, progressCallback func(int64)) error {
	apiURL := fmt.Sprintf("%s/wp-json/dback/v1/import", c.BaseURL)

	file, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}
	totalSize := fileInfo.Size()

	pr := &ProgressReader{
		Reader: file,
		Callback: func(curr int64) {
			if progressCallback != nil {
				progressCallback(curr)
			}
		},
	}

	// We send raw body (application/gzip)
	req, err := http.NewRequest("POST", apiURL, pr)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-DBACK-KEY", c.Key)
	req.Header.Set("Content-Type", "application/gzip")
	req.ContentLength = totalSize // Important for streaming upload

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, _ := io.ReadAll(resp.Body)

	// Check for JSON error response (even on 200)
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var wpErr WPError
		if json.Unmarshal(body, &wpErr) == nil && wpErr.Code != "" {
			return fmt.Errorf("WordPress API error [%s]: %s", wpErr.Code, wpErr.Message)
		}
	}

	if resp.StatusCode != http.StatusOK {
		bodyStr := string(body)
		if bodyStr == "" {
			bodyStr = "(empty response)"
		}
		return fmt.Errorf("import failed (HTTP %d): %s", resp.StatusCode, bodyStr)
	}

	return nil
}

// ProgressReader helper
type ProgressReader struct {
	Reader   io.Reader
	Current  int64
	Callback func(int64)
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.Current += int64(n)
	if pr.Callback != nil {
		pr.Callback(pr.Current)
	}
	return n, err
}
