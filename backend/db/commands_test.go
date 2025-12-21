package db

import (
	"dback/models"
	"os/exec"
	"strings"
	"testing"
)

func TestBuildExportCommand_Syntax(t *testing.T) {
	// Profile with special characters in password to test quoting
	p := models.Profile{
		DBType:       models.DBTypePostgreSQL,
		DBUser:       "user",
		DBPassword:   "pass'word", // Single quote in password
		DBHost:       "localhost",
		DBPort:       "5432",
		TargetDBName: "mydb",
		IsDocker:     false,
	}

	cmd := BuildExportCommand(p)
	t.Logf("Generated Command: %s", cmd)

	// Basic syntax check: Does it have nested single quotes that break bash?
	// The command is roughly: bash -c 'set -o pipefail; PGPASSWORD='pass'word' ...'
	// This is definitely broken if not escaped.

	// Try to execute it with safe replacements to verify syntax
	// Replace dump commands with echo to produce output, compression with cat to consume input
	safeCmd := strings.ReplaceAll(cmd, "pg_dump", "echo 'dummy data'")
	safeCmd = strings.ReplaceAll(safeCmd, "mysqldump", "echo 'dummy data'")
	safeCmd = strings.ReplaceAll(safeCmd, "tar cf -", "echo 'dummy data'")
	safeCmd = strings.ReplaceAll(safeCmd, "zstd", "cat")
	safeCmd = strings.ReplaceAll(safeCmd, "gzip", "cat")

	// We wrap it in bash -c because that's how SSH executes it (roughly sh -c)
	// exec.Command("bash", "-c", safeCmd)

	c := exec.Command("bash", "-c", safeCmd)
	output, err := c.CombinedOutput()

	if err != nil {
		t.Errorf("Command syntax error: %v\nOutput: %s", err, output)
	} else {
		t.Logf("Command syntax valid. Output: %s", output)
	}
}
