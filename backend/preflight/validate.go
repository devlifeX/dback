package preflight

import (
	"fmt"
	"strconv"
	"strings"

	"dback/models"
)

type DockerStatus int

const (
	DockerOK DockerStatus = iota
	DockerMissing
	DockerPermissionDenied
	ContainerNotFound
	ContainerNotRunning
	ContainerToolMissing
)

func (s DockerStatus) Error() string {
	switch s {
	case DockerMissing:
		return "docker is not installed or not in PATH"
	case DockerPermissionDenied:
		return "docker permission denied"
	case ContainerNotFound:
		return "docker container not found"
	case ContainerNotRunning:
		return "docker container is not running"
	case ContainerToolMissing:
		return "required database tool missing inside container"
	default:
		return ""
	}
}

func validateParsedOutput(out string, p models.Profile, requiredKB int64) error {
	section := ""
	var fails []string
	disk := map[string]int64{}
	var dockerStatus string
	hasDumpTool := false
	hasClient := false
	hasCompress := false
	isLinux := false
	writable := map[string]bool{}

	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		switch line {
		case "===OS===":
			section = "os"
		case "===DB===":
			section = "db"
		case "===TOOLS===":
			section = "tools"
		case "===DOCKER===":
			section = "docker"
		case "===DISK===":
			section = "disk"
		case "===WRITE===":
			section = "write"
		case "===REQUIRED_KB===":
			section = "required"
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
			case "db":
				low := strings.ToLower(line)
				if strings.Contains(low, "mysqldump") || strings.Contains(low, "mariadb-dump") {
					hasDumpTool = true
				}
				if strings.Contains(low, "mysql") || strings.Contains(low, "mariadb") {
					hasClient = true
				}
			case "tools":
				low := strings.ToLower(line)
				if strings.Contains(low, "gzip") || strings.Contains(low, "zstd") {
					hasCompress = true
				}
			case "docker":
				low := strings.ToLower(line)
				if strings.Contains(low, "mysqldump") || strings.Contains(low, "mariadb-dump") {
					hasDumpTool = true
				}
				if strings.Contains(low, "mysql") || strings.Contains(low, "mariadb") {
					hasClient = true
				}
				if strings.Contains(low, "docker missing") || strings.Contains(low, "command not found") {
					dockerStatus = "missing"
				}
				if strings.Contains(low, "permission denied") {
					dockerStatus = "permission"
				}
				if strings.Contains(low, "container not found") || strings.Contains(low, "no such object") {
					dockerStatus = "notfound"
				}
				if line == "running" {
					dockerStatus = "running"
				}
				if strings.Contains(low, "no mysql client") || strings.Contains(low, "no dump tool") {
					dockerStatus = "toolmissing"
				}
			case "disk":
				parts := strings.Split(line, "|")
				if len(parts) >= 3 {
					freeKB, _ := strconv.ParseInt(parts[1], 10, 64)
					disk[parts[2]] = freeKB
				}
			case "write":
				parts := strings.SplitN(line, "|", 2)
				if len(parts) == 2 && parts[0] == "ok" {
					writable[parts[1]] = true
				}
			case "result":
				if strings.HasPrefix(line, "fail=") && strings.TrimPrefix(line, "fail=") != "0" {
					fails = append(fails, "remote preflight reported failure")
				}
				if strings.HasPrefix(line, "msg=") {
					msg := strings.TrimSpace(strings.TrimPrefix(line, "msg="))
					if msg != "" {
						fails = append(fails, msg)
					}
				}
			}
		}
	}

	if !isLinux {
		fails = append(fails, "remote host is not Linux")
	}
	if !hasDumpTool {
		if p.IsDocker {
			fails = append(fails, "mysqldump or mariadb-dump not found inside container")
		} else {
			fails = append(fails, "mysqldump or mariadb-dump not found")
		}
	}
	if !hasClient {
		if p.IsDocker {
			fails = append(fails, "mysql or mariadb client not found inside container")
		} else {
			fails = append(fails, "mysql or mariadb client not found")
		}
	}
	if !hasCompress {
		fails = append(fails, "gzip or zstd not found")
	}

	if p.IsDocker {
		switch dockerStatus {
		case "missing":
			fails = append(fails, DockerMissing.Error())
		case "permission":
			fails = append(fails, DockerPermissionDenied.Error())
		case "notfound":
			fails = append(fails, ContainerNotFound.Error())
		case "toolmissing":
			fails = append(fails, ContainerToolMissing.Error())
		case "running":
			// ok
		default:
			if dockerStatus == "" {
				fails = append(fails, ContainerNotRunning.Error())
			} else if dockerStatus != "running" {
				fails = append(fails, ContainerNotRunning.Error())
			}
		}
		if strings.TrimSpace(p.ContainerID) == "" {
			fails = append(fails, "container ID is required for docker hosts")
		}
	}

	selected, _, ok := selectTmpDir(disk, requiredKB)
	if !ok {
		fails = append(fails, fmt.Sprintf("insufficient disk space (need %d KB)", requiredKB))
	} else if len(writable) == 0 {
		fails = append(fails, "no writable tmp path found")
	} else if !writable[selected] {
		// Allow fallback when another candidate path is writable.
		allowed := false
		for path := range writable {
			if free, ok := disk[path]; ok && free >= requiredKB {
				allowed = true
				break
			}
		}
		if !allowed {
			fails = append(fails, "selected tmp path is not writable")
		}
	}

	if len(fails) > 0 {
		return fmt.Errorf("preflight failed: %s", strings.Join(fails, "; "))
	}
	return nil
}
