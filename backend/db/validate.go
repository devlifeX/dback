package db

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"dback/models"
)

var (
	containerIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)
	dbUserPattern      = regexp.MustCompile(`^[a-zA-Z0-9_@.-]{1,64}$`)
	dbHostPattern      = regexp.MustCompile(`^[a-zA-Z0-9_.:-]{1,253}$`)
	dbPortPattern      = regexp.MustCompile(`^[0-9]{1,5}$`)
)

func ValidateContainerID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("container ID is required")
	}
	if !containerIDPattern.MatchString(id) {
		return errors.New("invalid container ID")
	}
	return nil
}

func ValidateDBUser(user string) error {
	user = strings.TrimSpace(user)
	if user == "" {
		return errors.New("db user is required")
	}
	if !dbUserPattern.MatchString(user) {
		return errors.New("invalid db user")
	}
	return nil
}

func ValidateDBHost(host string) error {
	host = strings.TrimSpace(host)
	if host == "" {
		return errors.New("db host is required")
	}
	if !dbHostPattern.MatchString(host) {
		return errors.New("invalid db host")
	}
	return nil
}

func ValidateDBPort(port string) error {
	port = strings.TrimSpace(port)
	if port == "" {
		return errors.New("db port is required")
	}
	if !dbPortPattern.MatchString(port) {
		return errors.New("invalid db port")
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return errors.New("invalid db port")
	}
	return nil
}

func ValidateProfileForRemoteOps(p models.Profile) error {
	if err := ValidateDBUser(p.DBUser); err != nil {
		return fmt.Errorf("db user: %w", err)
	}
	if p.IsDocker {
		if err := ValidateContainerID(p.ContainerID); err != nil {
			return fmt.Errorf("container id: %w", err)
		}
		return nil
	}
	if err := ValidateDBHost(p.DBHost); err != nil {
		return fmt.Errorf("db host: %w", err)
	}
	if err := ValidateDBPort(p.DBPort); err != nil {
		return fmt.Errorf("db port: %w", err)
	}
	return nil
}

func dockerExecCommand(containerID, inner string) (string, error) {
	if err := ValidateContainerID(containerID); err != nil {
		return "", err
	}
	return fmt.Sprintf("docker exec -i %s sh -c %s", shellEscape(containerID), shellEscape(shellWithPipefail(inner))), nil
}
