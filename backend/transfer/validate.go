package transfer

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"
)

// validateBackupIntegrity ensures the backup file is a complete, valid gzip archive.
// Uses pure Go to avoid requiring an external gzip binary on any platform.
func validateBackupIntegrity(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("backup file is corrupt or incomplete (gzip check failed): %s", err)
	}
	defer f.Close()
	r, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("backup file is corrupt or incomplete (gzip check failed): %s", err)
	}
	defer r.Close()
	if _, err := io.Copy(io.Discard, r); err != nil {
		return fmt.Errorf("backup file is corrupt or incomplete (gzip check failed): %s", err)
	}
	return nil
}

// validateRemoteBackupIntegrity runs gzip -t on the remote host before download.
func validateRemoteBackupIntegrity(client interface {
	RunCommand(string) (string, error)
}, remotePath string) error {
	cmd := fmt.Sprintf("gzip -t %s 2>&1", shellQuote(remotePath))
	out, err := client.RunCommand(cmd)
	if err != nil {
		msg := strings.TrimSpace(out)
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("remote dump failed integrity check: %s", msg)
	}
	return nil
}
