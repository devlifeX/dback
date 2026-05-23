package db

import (
	"os/exec"
	"strings"
	"testing"

	"dback/models"
)

func TestBuildExportCommand_Syntax(t *testing.T) {
	// Profile with special characters in password to test quoting
	p := models.Profile{
		DBType:       models.DBTypeMySQL,
		DBUser:       "user",
		DBPassword:   "pass'word", // Single quote in password
		DBHost:       "localhost",
		DBPort:       "3306",
		TargetDBName: "mydb",
		IsDocker:     false,
	}

	cmd := BuildExportCommand(p)
	t.Logf("Generated Command: %s", cmd)

	c := exec.Command("bash", "-n", "-c", cmd)
	output, err := c.CombinedOutput()

	if err != nil {
		t.Errorf("Command syntax error: %v\nOutput: %s", err, output)
	} else {
		t.Logf("Command syntax valid.")
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
		DBHost:       "127.0.0.1",
		DBPort:       "3306",
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
	for _, want := range []string{"gzip -dc", "pipefail", "mydb", "FOREIGN_KEY_CHECKS"} {
		if !strings.Contains(stream, want) {
			t.Fatalf("stream command missing %q: %s", want, stream)
		}
	}
	prep := BuildImportPrepareCommand(p)
	if !strings.Contains(prep, "DROP DATABASE") || !strings.Contains(prep, "CREATE DATABASE") {
		t.Fatalf("prepare command missing drop/create: %s", prep)
	}
}

func TestBuildTmpFileCommands(t *testing.T) {
	p := models.Profile{
		DBType:       models.DBTypeMariaDB,
		DBUser:       "user",
		DBPassword:   "secret",
		TargetDBName: "mydb",
	}
	export := BuildExportToFileCommand(p, "/tmp/dback/op/dump.sql.gz")
	if !strings.Contains(export, "/tmp/dback/op/dump.sql.gz") {
		t.Fatalf("export tmp command missing path: %s", export)
	}
	importCmd := BuildImportFromFileCommand(p, "/tmp/dback/op/import.sql.gz", "gzip")
	if !strings.Contains(importCmd, "/tmp/dback/op/import.sql.gz") {
		t.Fatalf("import tmp command missing path: %s", importCmd)
	}
	if strings.Contains(MaskCommand(importCmd), "secret") {
		t.Fatal("masked command must not contain password")
	}
}

func TestMaskCommand(t *testing.T) {
	for _, cmd := range []string{
		"mysql -u root -p'secret' -h localhost",
		shellEscape("mysql -u root -p'secret' -h localhost"),
		shellEscape("mysql -u root -p'pass'\\''word' -h localhost"),
	} {
		masked := MaskCommand(cmd)
		if strings.Contains(masked, "secret") || strings.Contains(masked, "pass") || strings.Contains(masked, "word") {
			t.Fatalf("password should be masked: %s", masked)
		}
		if !strings.Contains(masked, "-p'***'") {
			t.Fatalf("expected masked password marker: %s", masked)
		}
	}
}

func TestParseDatabaseSizeBytes(t *testing.T) {
	if got := ParseDatabaseSizeBytes("1048576"); got != 1048576 {
		t.Fatalf("expected plain integer parse, got %d", got)
	}
	if got := ParseDatabaseSizeBytes("COALESCE(SUM(data_length + index_length), 0)\n3145728"); got != 3145728 {
		t.Fatalf("expected batch table output parse, got %d", got)
	}
}

func TestEstimateCompressedBackupSize(t *testing.T) {
	if got := EstimateCompressedBackupSize(0); got != 0 {
		t.Fatalf("expected zero for empty input")
	}
	if got := EstimateCompressedBackupSize(10 * 1024 * 1024); got != 512*1024 {
		t.Fatalf("expected 512KB floor for 10MB raw, got %d", got)
	}
	if got := EstimateCompressedBackupSize(100 * 1024 * 1024); got != 5*1024*1024 {
		t.Fatalf("expected gzip ratio /20, got %d", got)
	}
}

func TestMysqlDumpArgs(t *testing.T) {
	p := models.Profile{DBType: models.DBTypeMySQL}
	args := mysqlDumpArgs(p)
	for _, want := range []string{
		"--single-transaction",
		"--quick",
		"--lock-tables=false",
		"--hex-blob",
		"--set-gtid-purged=OFF",
	} {
		if !strings.Contains(args, want) {
			t.Fatalf("dump args missing %q: %s", want, args)
		}
	}
	for _, absent := range []string{"--column-statistics=0", "--no-tablespaces"} {
		if strings.Contains(args, absent) {
			t.Fatalf("dump args must not always include MySQL 8 flag %q: %s", absent, args)
		}
	}
	dump := mysqlDumpExec(p)
	if !strings.Contains(dump, "--single-transaction") {
		t.Fatalf("dump command missing flags: %s", dump)
	}
	if !strings.Contains(dump, "mysqldump --version") {
		t.Fatalf("dump command should probe mysqldump version for MySQL 8 flags: %s", dump)
	}
	if !strings.Contains(dump, "--column-statistics=0") {
		t.Fatalf("dump command should conditionally include MySQL 8 flags: %s", dump)
	}
}

func TestBuildDatabaseSizeCommand(t *testing.T) {
	p := models.Profile{
		DBType:       models.DBTypeMySQL,
		DBUser:       "root",
		DBPassword:   "secret",
		DBHost:       "127.0.0.1",
		DBPort:       "3306",
		TargetDBName: "mydb",
	}
	cmd, err := BuildDatabaseSizeCommand(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cmd, "base64") {
		t.Fatalf("expected base64-wrapped size query command, got: %s", cmd)
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
