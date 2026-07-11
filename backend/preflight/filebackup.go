package preflight

import (
	"fmt"
	"os/exec"
	"strings"

	"dback/backend/ssh"
	"dback/models"
)

// BuildFileBackupPreflightScript returns a remote shell script that checks tar and compression tools.
func BuildFileBackupPreflightScript(compression models.ArchiveCompression) string {
	comp := models.NormalizeArchiveCompression(compression)
	compProbe := "command -v gzip >/dev/null 2>&1 && gzip --version 2>/dev/null | head -1 || echo no-gzip"
	if comp == models.ArchiveCompressionZstd {
		compProbe = "command -v zstd >/dev/null 2>&1 && zstd --version 2>/dev/null | head -1 || echo no-zstd"
	}
	return strings.Join([]string{
		"set -eu",
		"echo '===OS==='",
		"uname -s",
		"echo '===TOOLS==='",
		"command -v tar >/dev/null 2>&1 && tar --version 2>/dev/null | head -1 || echo no-tar",
		compProbe,
		"echo '===RESULT==='",
		"echo fail=0",
	}, "\n")
}

// ValidateFileBackupOutput checks remote preflight output for file backup requirements.
func ValidateFileBackupOutput(out string, compression models.ArchiveCompression) error {
	comp := models.NormalizeArchiveCompression(compression)
	section := ""
	isLinux := false
	hasTar := false
	hasCompress := false

	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		switch line {
		case "===OS===":
			section = "os"
		case "===TOOLS===":
			section = "tools"
		case "===RESULT===":
			section = "result"
		default:
			if line == "" {
				continue
			}
			switch section {
			case "os":
				if strings.Contains(strings.ToLower(line), "linux") {
					isLinux = true
				}
			case "tools":
				low := strings.ToLower(line)
				if strings.Contains(low, "tar") && !strings.Contains(low, "no-tar") {
					hasTar = true
				}
				if comp == models.ArchiveCompressionGzip {
					if strings.Contains(low, "gzip") && !strings.Contains(low, "no-gzip") {
						hasCompress = true
					}
				} else if strings.Contains(low, "zstd") && !strings.Contains(low, "no-zstd") {
					hasCompress = true
				}
			}
		}
	}

	var fails []string
	if !isLinux {
		fails = append(fails, "remote host is not Linux")
	}
	if !hasTar {
		fails = append(fails, "tar not found")
	}
	if !hasCompress {
		if comp == models.ArchiveCompressionGzip {
			fails = append(fails, "gzip not found")
		} else {
			fails = append(fails, "zstd not found")
		}
	}
	if len(fails) > 0 {
		return fmt.Errorf("file backup preflight failed: %s", strings.Join(fails, "; "))
	}
	return nil
}

// RunFileBackup executes file-backup preflight checks on a remote SSH host.
func RunFileBackup(client ssh.Executor, compression models.ArchiveCompression) error {
	out, err := client.RunCommand(BuildFileBackupPreflightScript(compression))
	if err != nil {
		return fmt.Errorf("file backup preflight failed: %w: %s", err, truncate(out, 500))
	}
	return ValidateFileBackupOutput(out, compression)
}

// CheckLocalFileBackupTools verifies tar and compression binaries exist locally.
func CheckLocalFileBackupTools(compression models.ArchiveCompression) error {
	if _, err := exec.LookPath("tar"); err != nil {
		return fmt.Errorf("tar not found in PATH")
	}
	comp := models.NormalizeArchiveCompression(compression)
	binary := "gzip"
	if comp == models.ArchiveCompressionZstd {
		binary = "zstd"
	}
	if _, err := exec.LookPath(binary); err != nil {
		return fmt.Errorf("%s not found in PATH", binary)
	}
	return nil
}
