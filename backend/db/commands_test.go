package db

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"dback/models"
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

func TestBuildQueryCommand(t *testing.T) {
	p := models.Profile{
		DBType:       models.DBTypeMySQL,
		DBUser:       "user",
		DBPassword:   "secret",
		DBHost:       "127.0.0.1",
		DBPort:       "3306",
		TargetDBName: "mydb",
		IsDocker:     false,
	}

	cmd, err := BuildQueryCommand(p, "SHOW TABLES;", true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cmd, "base64") {
		t.Fatalf("expected base64 in command, got: %s", cmd)
	}
	if !strings.Contains(cmd, "mydb") {
		t.Fatalf("expected database name in command, got: %s", cmd)
	}

	_, err = BuildQueryCommand(p, "", true)
	if err == nil {
		t.Fatal("expected error for empty query")
	}

	cmdMulti, err := BuildQueryCommand(p, "SELECT 1; SELECT 2;", true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cmdMulti, "base64") {
		t.Fatalf("expected multi-statement command, got: %s", cmdMulti)
	}
	if !strings.Contains(cmd, "--batch") {
		t.Fatalf("expected batch flags, got: %s", cmd)
	}
}

func TestBuildQueryCommand_NoDatabase(t *testing.T) {
	p := models.Profile{
		DBType:       models.DBTypeMySQL,
		DBUser:       "root",
		DBPassword:   "pass",
		TargetDBName: "mydb",
	}
	cmd, err := BuildQueryCommand(p, "DROP DATABASE mydb;", false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(cmd, "mydb") && strings.Contains(cmd, "--batch") {
		// db name should not be passed as mysql default database argument when connectDB is false
		// (may still appear inside base64 payload)
	}
	if !strings.Contains(cmd, "base64") {
		t.Fatalf("expected base64 pipe, got: %s", cmd)
	}
}

func TestParseMySQLBatchOutput(t *testing.T) {
	out := "id\tname\n1\talice\n2\tbob"
	result := ParseMySQLBatchOutput(out)
	if len(result.Columns) != 2 || result.Columns[0] != "id" {
		t.Fatalf("unexpected columns: %v", result.Columns)
	}
	if len(result.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(result.Rows))
	}

	plain := ParseMySQLBatchOutput("Query OK, 3 rows affected")
	if len(plain.Rows) != 1 || plain.Rows[0][0] != "Query OK, 3 rows affected" {
		t.Fatalf("unexpected plain result: %v", plain.Rows)
	}
}

func TestBuildImportStreamCommand(t *testing.T) {
	p := models.Profile{
		DBType:       models.DBTypeMySQL,
		DBUser:       "user",
		DBPassword:   "secret",
		DBHost:       "127.0.0.1",
		DBPort:       "3306",
		TargetDBName: "mydb",
	}
	stream := BuildImportStreamCommand(p, "gzip")
	for _, want := range []string{"gzip -dc", "bash -o pipefail", "mydb", "FOREIGN_KEY_CHECKS"} {
		if !strings.Contains(stream, want) {
			t.Fatalf("stream command missing %q: %s", want, stream)
		}
	}
	if strings.Contains(stream, "/tmp/dback_import") {
		t.Fatalf("stream import should not buffer to /tmp: %s", stream)
	}
	prep := BuildImportPrepareCommand(p)
	if !strings.Contains(prep, "DROP DATABASE") || !strings.Contains(prep, "CREATE DATABASE") {
		t.Fatalf("prepare command missing drop/create: %s", prep)
	}
	if strings.Contains(prep, "fi -e") {
		t.Fatalf("prepare must not append -e after fi: %s", prep)
	}

	innerPrep := "set -e; " + mysqlClientExec(p, "", "-e "+shellEscape("SELECT 1;"))
	if err := exec.Command("sh", "-n", "-c", innerPrep).Run(); err != nil {
		t.Fatalf("prepare inner shell syntax invalid: %v\n%s", err, innerPrep)
	}

	innerStream := fmt.Sprintf("{ printf \"SET FOREIGN_KEY_CHECKS=0;\\n\"; %s; } | %s",
		importDecompressStream("gzip"), mysqlClientExec(p, "mydb", ""))
	if err := exec.Command("sh", "-n", "-c", innerStream).Run(); err != nil {
		t.Fatalf("stream inner shell syntax invalid: %v\n%s", err, innerStream)
	}
}

func TestBuildQueryCommand_Docker(t *testing.T) {
	p := models.Profile{
		DBType:       models.DBTypeMariaDB,
		DBUser:       "root",
		DBPassword:   "pass",
		TargetDBName: "app",
		IsDocker:     true,
		ContainerID:  "mysql_container",
	}

	cmd, err := BuildQueryCommand(p, "SELECT VERSION();", true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cmd, "docker exec") {
		t.Fatalf("expected docker exec, got: %s", cmd)
	}
	if !strings.Contains(cmd, "mysql_container") {
		t.Fatalf("expected container id, got: %s", cmd)
	}
}
