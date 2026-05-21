package transfer

import (
	"fmt"
	"os/exec"
	"strings"
)

// validateBackupIntegrity ensures the backup file is a complete, valid gzip archive.
func validateBackupIntegrity(path string) error {
	out, err := exec.Command("gzip", "-t", path).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("backup file is corrupt or incomplete (gzip check failed): %s", msg)
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
