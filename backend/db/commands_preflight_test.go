package db

import (
	"strings"
	"testing"

	"dback/models"
)

func TestBuildPreflightScriptUsesGroupedDockerProbes(t *testing.T) {
	p := models.Profile{DBType: models.DBTypeMySQL, IsDocker: true, ContainerID: "db"}
	script := BuildPreflightScript(p, 0, []string{"/tmp"})
	if !strings.Contains(script, "(command -v mysqldump >/dev/null 2>&1 && mysqldump --version) || (command -v mariadb-dump >/dev/null 2>&1 && mariadb-dump --version)") {
		t.Fatal("expected grouped dump probe to avoid shell precedence bug")
	}
	if !strings.Contains(script, "(command -v mysql >/dev/null 2>&1 && mysql --version) || (command -v mariadb >/dev/null 2>&1 && mariadb --version)") {
		t.Fatal("expected grouped client probe to avoid shell precedence bug")
	}
	if !strings.Contains(script, "===CHECKS===") {
		t.Fatal("expected checks section for debug probes")
	}
	if !strings.Contains(script, "check|container_dump_version|cmd|") {
		t.Fatal("expected recorded dump probe command")
	}
	if !strings.Contains(script, "check|container_client_version|cmd|") {
		t.Fatal("expected recorded client probe command")
	}
}
