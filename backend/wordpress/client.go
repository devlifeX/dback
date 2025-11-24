package wordpress

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

// Client handles communication with the WordPress plugin
type Client struct {
	BaseURL string
	Key     string
	Client  *http.Client
}

// NewClient creates a new WordPress client
func NewClient(url, key string) *Client {
	return &Client{
		BaseURL: url,
		Key:     key,
		Client:  &http.Client{},
	}
}

// Export triggers a DB dump on WP and streams it to destPath
func (c *Client) Export(destPath string, progressCallback func(int64)) error {
	// URL: /wp-json/dback/v1/export
	// We need to ensure BaseURL doesn't have trailing slash, and has wp-json if not provided?
	// Usually user provides site root. We append /wp-json/...
	// Let's assume BaseURL is site root.
	apiURL := fmt.Sprintf("%s/wp-json/dback/v1/export", c.BaseURL)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-DBACK-KEY", c.Key)

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("export failed (status %d): %s", resp.StatusCode, string(body))
	}

	// Create local file
	out, err := os.Create(destPath)
	if err != nil {
		return err
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

	_, err = io.Copy(out, pr)
	return err
}

// Import uploads a SQL dump to WP
func (c *Client) Import(sourcePath string, progressCallback func(int64)) error {
	apiURL := fmt.Sprintf("%s/wp-json/dback/v1/import", c.BaseURL)

	file, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer file.Close()

	fileInfo, _ := file.Stat()
	totalSize := fileInfo.Size()

	pr := &ProgressReader{
		Reader: file,
		Callback: func(curr int64) {
			if progressCallback != nil {
				progressCallback(curr) // % can be calc with totalSize
			}
		},
	}

	// We send raw body (application/gzip)
	req, err := http.NewRequest("POST", apiURL, pr)
	if err != nil {
		return err
	}
	req.Header.Set("X-DBACK-KEY", c.Key)
	req.Header.Set("Content-Type", "application/gzip")
	req.ContentLength = totalSize // Important for streaming upload

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("import failed (status %d): %s", resp.StatusCode, string(body))
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
