package archiver

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"dback/models"

	"github.com/klauspost/compress/zstd"
)

// ValidateIntegrity ensures a compressed archive file is readable end-to-end.
func ValidateIntegrity(path string, compression models.ArchiveCompression) error {
	switch models.NormalizeArchiveCompression(compression) {
	case models.ArchiveCompressionGzip:
		return validateGzipArchive(path)
	case models.ArchiveCompressionZstd:
		return validateZstdArchive(path)
	default:
		return fmt.Errorf("unsupported compression %q", compression)
	}
}

func validateGzipArchive(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("archive integrity check failed: %w", err)
	}
	defer f.Close()
	r, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("archive integrity check failed: %w", err)
	}
	defer r.Close()
	if _, err := io.Copy(io.Discard, r); err != nil {
		return fmt.Errorf("archive integrity check failed: %w", err)
	}
	return nil
}

func validateZstdArchive(path string) error {
	if _, err := exec.LookPath("zstd"); err == nil {
		out, err := exec.Command("zstd", "-t", path).CombinedOutput()
		if err != nil {
			msg := strings.TrimSpace(string(out))
			if msg == "" {
				msg = err.Error()
			}
			return fmt.Errorf("archive integrity check failed: %s", msg)
		}
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("archive integrity check failed: %w", err)
	}
	defer f.Close()
	dec, err := zstd.NewReader(f)
	if err != nil {
		return fmt.Errorf("archive integrity check failed: %w", err)
	}
	defer dec.Close()
	if _, err := io.Copy(io.Discard, dec); err != nil {
		return fmt.Errorf("archive integrity check failed: %w", err)
	}
	return nil
}
