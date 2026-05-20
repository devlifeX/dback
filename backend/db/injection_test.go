package db

import (
	"os/exec"
	"strings"
	"testing"

	"dback/models"
)

func TestValidateProfileForRemoteOps_RejectsInjection(t *testing.T) {
	cases := []struct {
		name    string
		profile models.Profile
	}{
		{
			name: "db user injection",
			profile: models.Profile{
				DBUser:       "root; rm -rf /",
				DBHost:       "127.0.0.1",
				DBPort:       "3306",
				TargetDBName: "app",
			},
		},
		{
			name: "db host injection",
			profile: models.Profile{
				DBUser:       "root",
				DBHost:       "127.0.0.1; evil",
				DBPort:       "3306",
				TargetDBName: "app",
			},
		},
		{
			name: "db port injection",
			profile: models.Profile{
				DBUser:       "root",
				DBHost:       "127.0.0.1",
				DBPort:       "3306; rm -rf /",
				TargetDBName: "app",
			},
		},
		{
			name: "container id injection",
			profile: models.Profile{
				DBUser:       "root",
				IsDocker:     true,
				ContainerID:  "mysql; rm -rf /",
				TargetDBName: "app",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateProfileForRemoteOps(tc.profile); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestBuildQueryCommand_EscapesShellFields(t *testing.T) {
	p := models.Profile{
		DBType:       models.DBTypeMySQL,
		DBUser:       "root",
		DBPassword:   "pass'word",
		DBHost:       "127.0.0.1",
		DBPort:       "3306",
		TargetDBName: "mydb",
	}
	cmd, err := BuildQueryCommand(p, "SELECT 1;", true)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(cmd, "-proot") || strings.Contains(cmd, "-ppass'word") {
		t.Fatalf("password or user must be shell-quoted, got: %s", cmd)
	}
	c := exec.Command("bash", "-n", "-c", cmd)
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("command syntax error: %v\n%s", err, out)
	}
}

func TestBuildExportCommand_DockerContainerEscaped(t *testing.T) {
	p := models.Profile{
		DBType:       models.DBTypeMySQL,
		DBUser:       "root",
		DBPassword:   "secret",
		TargetDBName: "app",
		IsDocker:     true,
		ContainerID:  "mysql_container",
	}
	cmd := BuildExportCommand(p)
	if !strings.Contains(cmd, "'mysql_container'") {
		t.Fatalf("expected escaped container id, got: %s", cmd)
	}
	if _, err := dockerExecCommand("bad;id", "echo hi"); err == nil {
		t.Fatal("expected invalid container id error")
	}
}
