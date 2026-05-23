package preflight

import (
	"strings"
	"testing"

	"dback/models"
)

func TestValidateParsedOutputRequiresLinuxAndTools(t *testing.T) {
	out := `===OS===
Linux staging 6.8.0 x86_64 GNU/Linux
===DB===
mysql Ver 8.0
mysqldump Ver 10.6
===TOOLS===
gzip 1.10
===DISK===
/dev/sda1|1048576|/tmp
===WRITE===
ok|/tmp
===REQUIRED_KB===
524288
===RESULT===
fail=0
msg=`
	p := models.Profile{DBType: models.DBTypeMySQL}
	if err := validateParsedOutput(out, p, 524288); err != nil {
		t.Fatalf("expected pass, got %v", err)
	}
}

func TestValidateParsedOutputFailsWithoutDumpTool(t *testing.T) {
	out := `===OS===
Linux host
===DB===
mysql Ver 8.0
===TOOLS===
gzip 1.10
===DISK===
/dev/sda1|1048576|/tmp
===WRITE===
ok|/tmp
===REQUIRED_KB===
524288
===RESULT===
fail=0
msg=`
	p := models.Profile{DBType: models.DBTypeMySQL}
	err := validateParsedOutput(out, p, 524288)
	if err == nil || !strings.Contains(err.Error(), "dump") {
		t.Fatalf("expected dump tool failure, got %v", err)
	}
}

func TestValidateParsedOutputDockerRequiresRunningContainer(t *testing.T) {
	out := `===OS===
Linux host
===DB===
mariadb-dump Ver 10.6
===TOOLS===
gzip 1.10
===DOCKER===
docker missing
exited
===DISK===
/dev/sda1|1048576|/tmp
===WRITE===
ok|/tmp
===REQUIRED_KB===
524288
===RESULT===
fail=0
msg=`
	p := models.Profile{DBType: models.DBTypeMariaDB, IsDocker: true, ContainerID: "db1"}
	err := validateParsedOutput(out, p, 524288)
	if err == nil {
		t.Fatal("expected docker validation failure")
	}
}

func TestValidateParsedOutputDockerUsesContainerTools(t *testing.T) {
	out := `===OS===
Linux host
===DB===
mariadb-dump Ver 10.6
===TOOLS===
gzip 1.10
===DOCKER===
Docker version 28.1.1
running
mariadb-dump Ver 10.6
mysql Ver 15.1 Distrib 10.6.21-MariaDB
===DISK===
/dev/sda1|1048576|/tmp
===WRITE===
ok|/tmp
===REQUIRED_KB===
524288
===RESULT===
fail=0
msg=`
	p := models.Profile{DBType: models.DBTypeMariaDB, IsDocker: true, ContainerID: "db1"}
	if err := validateParsedOutput(out, p, 524288); err != nil {
		t.Fatalf("expected docker preflight to pass, got %v", err)
	}
}

func TestValidateParsedOutputDockerFailsWhenClientMissingButDumpPresent(t *testing.T) {
	out := `===OS===
Linux host
===DB===
mysqldump Ver 8.0
===TOOLS===
gzip 1.10
===DOCKER===
Docker version 28.1.1
running
mysqldump Ver 8.0
no-mysql-client
===DISK===
/dev/sda1|1048576|/tmp
===WRITE===
ok|/tmp
===REQUIRED_KB===
524288
===RESULT===
fail=0
msg=`
	p := models.Profile{DBType: models.DBTypeMySQL, IsDocker: true, ContainerID: "db"}
	err := validateParsedOutput(out, p, 524288)
	if err == nil {
		t.Fatal("expected client missing failure")
	}
	if strings.Contains(err.Error(), "required database tool missing inside container") {
		t.Fatalf("expected specific client error, got %v", err)
	}
	if !strings.Contains(err.Error(), "mysql or mariadb client missing inside container") {
		t.Fatalf("expected client missing error, got %v", err)
	}
	if strings.Contains(err.Error(), "mysqldump or mariadb-dump missing") {
		t.Fatalf("dump tool should not be reported missing, got %v", err)
	}
}

func TestValidateParsedOutputDockerFailsWhenDumpMissingButClientPresent(t *testing.T) {
	out := `===OS===
Linux host
===DB===
mysql Ver 8.0
===TOOLS===
gzip 1.10
===DOCKER===
Docker version 28.1.1
running
no-dump-tool
mysql Ver 8.0
===DISK===
/dev/sda1|1048576|/tmp
===WRITE===
ok|/tmp
===REQUIRED_KB===
524288
===RESULT===
fail=0
msg=`
	p := models.Profile{DBType: models.DBTypeMySQL, IsDocker: true, ContainerID: "db"}
	err := validateParsedOutput(out, p, 524288)
	if err == nil {
		t.Fatal("expected dump tool missing failure")
	}
	if !strings.Contains(err.Error(), "mysqldump or mariadb-dump missing inside container") {
		t.Fatalf("expected dump missing error, got %v", err)
	}
	if strings.Contains(err.Error(), "mysql or mariadb client missing") {
		t.Fatalf("client should not be reported missing, got %v", err)
	}
}

func TestLineHasClientIgnoresDumpToolLines(t *testing.T) {
	if lineHasClient("mysqldump Ver 8.0") {
		t.Fatal("mysqldump version line must not count as mysql client")
	}
	if lineHasClient("no-mysql-client") {
		t.Fatal("failure marker must not count as mysql client")
	}
	if !lineHasClient("mysql Ver 8.0") {
		t.Fatal("mysql version line must count as mysql client")
	}
}

func TestLineHasDumpToolIgnoresFailureMarkers(t *testing.T) {
	if lineHasDumpTool("no-dump-tool") {
		t.Fatal("failure marker must not count as dump tool")
	}
}

func TestValidateParsedOutputDockerPassesDespiteStaleFailureMarkers(t *testing.T) {
	out := `===OS===
Linux parned 6.8.0-100-generic #100-Ubuntu SMP PREEMPT_DYNAMIC Tue Jan 13 16:40:06 UTC 2026 x86_64 x86_64 x86_64 GNU/Linux
===DB===
mysqldump  Ver 10.13 Distrib 5.7.44, for Linux (x86_64)
===TOOLS===
gzip 1.10
===DOCKER===
Docker version 29.2.1, build a5c7197
running
mysqldump  Ver 10.13 Distrib 5.7.44, for Linux (x86_64)
no-dump-tool
mysql  Ver 14.14 Distrib 5.7.44, for Linux (x86_64) using  EditLine wrapper
no-mysql-client
===DISK===
/dev/sda1|1048576|/tmp
===WRITE===
ok|/tmp
===REQUIRED_KB===
524288
===RESULT===
fail=0
msg=`
	p := models.Profile{DBType: models.DBTypeMySQL, IsDocker: true, ContainerID: "db"}
	if err := validateParsedOutput(out, p, 524288); err != nil {
		t.Fatalf("expected pass when tool versions are present, got %v", err)
	}
}

func TestFailureDetailsIncludesProbeChecks(t *testing.T) {
	out := `===OS===
Linux host
===DB===
mysql Ver 8.0
mysqldump Ver 8.0
===TOOLS===
gzip 1.10
===CHECKS===
check|container_dump_version|cmd|docker exec 'db' sh -c dump
check|container_dump_version|exit|0
check|container_dump_version|out|mysqldump Ver 8.0
===DISK===
/dev/sda1|1048576|/tmp
===WRITE===
ok|/tmp
===REQUIRED_KB===
524288
===RESULT===
fail=0
msg=`
	result := Result{DiskPaths: make(map[string]int64)}
	parsePreflightOutput(out, &result)
	details := FailureDetails(result, nil)
	if !strings.Contains(details, "container_dump_version") {
		t.Fatalf("expected probe name in details, got %q", details)
	}
	if !strings.Contains(details, "cmd=\"docker exec 'db' sh -c dump\"") {
		t.Fatalf("expected probe cmd in details, got %q", details)
	}
	if !strings.Contains(details, "exit=0") || !strings.Contains(details, "out=\"mysqldump Ver 8.0\"") {
		t.Fatalf("expected probe exit/out in details, got %q", details)
	}
}
