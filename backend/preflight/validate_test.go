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
