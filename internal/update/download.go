package update

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func Download(ctx context.Context, client *http.Client, userAgent string, asset Asset, destDir string, progress func(stage string)) (string, error) {
	if client == nil {
		client = http.DefaultClient
	}
	if asset.URL == "" {
		return "", fmt.Errorf("update asset URL is empty")
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", err
	}

	destPath := filepath.Join(destDir, asset.Name)
	if progress != nil {
		progress("Downloading " + asset.Name + "…")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.URL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("download HTTP %d: %s", resp.StatusCode, truncate(string(body), 300))
	}

	tmpPath := destPath + ".part"
	out, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}

	written, err := io.Copy(out, resp.Body)
	closeErr := out.Close()
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return "", closeErr
	}
	if written == 0 {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("downloaded file is empty")
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	return destPath, nil
}
