package preflight

import (
	"fmt"
	"strconv"
	"strings"

	"dback/backend/db"
	"dback/backend/ssh"
	"dback/models"
)

// ProbeCheck records one preflight probe command and its result.
type ProbeCheck struct {
	Name string
	Cmd  string
	Exit string
	Out  string
}

// Result holds preflight check output from remote host.
type Result struct {
	OSInfo         string
	DBVersion      string
	DumpTool       string
	DockerStatus   string
	DiskPaths      map[string]int64 // path -> free KB
	RequiredKB     int64
	SelectedTmpDir string
	Checks         []ProbeCheck
	RawOutput      string
}

const minFreeKB = 512 * 1024 // 512 MB default threshold

// Run executes preflight checks on SSH host before backup/restore.
func Run(client ssh.Executor, p models.Profile, requiredBytes int64, operationID string) (Result, error) {
	candidates := []string{"/tmp/dback", "/tmp", "$HOME/dback-tmp"}
	script := db.BuildPreflightScript(p, requiredBytes, candidates)
	out, err := client.RunCommand(script)
	result := Result{
		DiskPaths: make(map[string]int64),
		RawOutput: out,
	}
	parsePreflightOutput(out, &result)

	if requiredBytes > 0 {
		result.RequiredKB = requiredBytes / 1024
	} else {
		result.RequiredKB = minFreeKB
	}

	if err != nil {
		return result, fmt.Errorf("preflight failed: %w: %s", err, truncate(out, 500))
	}
	if validateErr := validateParsedOutput(out, p, result.RequiredKB); validateErr != nil {
		return result, validateErr
	}

	selected, freeKB, ok := selectTmpDir(result.DiskPaths, result.RequiredKB)
	if !ok {
		return result, fmt.Errorf("insufficient disk space on remote host (need %d KB free); checked: %v",
			result.RequiredKB, result.DiskPaths)
	}
	_ = freeKB
	if selected == "/tmp/dback" || selected == "/tmp" {
		result.SelectedTmpDir = db.BuildRemoteTmpDir(operationID)
	} else if strings.HasPrefix(selected, "$HOME") {
		result.SelectedTmpDir = fmt.Sprintf("$HOME/dback/%s", operationID)
	} else {
		result.SelectedTmpDir = selected + "/dback/" + operationID
	}
	return result, nil
}

func parsePreflightOutput(out string, r *Result) {
	section := ""
	checkIndex := map[string]int{}
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
		case "===CHECKS===":
			section = "checks"
		case "===DISK===":
			section = "disk"
		case "===REQUIRED_KB===":
			section = "required"
		default:
			if line == "" {
				continue
			}
			switch section {
			case "os":
				r.OSInfo += line + "\n"
			case "db":
				r.DBVersion += line + "\n"
			case "tools":
				if strings.Contains(line, "dump") {
					r.DumpTool += line + "\n"
				}
			case "docker":
				r.DockerStatus += line + "\n"
			case "checks":
				parts := strings.SplitN(line, "|", 4)
				if len(parts) != 4 || parts[0] != "check" {
					continue
				}
				name := parts[1]
				field := parts[2]
				value := parts[3]
				idx, ok := checkIndex[name]
				if !ok {
					r.Checks = append(r.Checks, ProbeCheck{Name: name})
					idx = len(r.Checks) - 1
					checkIndex[name] = idx
				}
				switch field {
				case "cmd":
					r.Checks[idx].Cmd = value
				case "exit":
					r.Checks[idx].Exit = value
				case "out":
					r.Checks[idx].Out = value
				}
			case "disk":
				parts := strings.Split(line, "|")
				if len(parts) >= 3 {
					freeKB, _ := strconv.ParseInt(parts[1], 10, 64)
					r.DiskPaths[parts[2]] = freeKB
				}
			case "required":
				if kb, err := strconv.ParseInt(line, 10, 64); err == nil {
					r.RequiredKB = kb
				}
			}
		}
	}
}

func selectTmpDir(paths map[string]int64, requiredKB int64) (string, int64, bool) {
	order := []string{"/tmp/dback", "/tmp", "$HOME/dback-tmp"}
	for _, p := range order {
		if free, ok := paths[p]; ok && free >= requiredKB {
			return p, free, true
		}
	}
	for p, free := range paths {
		if free >= requiredKB {
			return p, free, true
		}
	}
	return "", 0, false
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func Summary(r Result) string {
	var b strings.Builder
	b.WriteString("OS: ")
	b.WriteString(strings.TrimSpace(r.OSInfo))
	b.WriteString(" | DB: ")
	b.WriteString(strings.TrimSpace(r.DBVersion))
	if r.DockerStatus != "" {
		b.WriteString(" | Docker: ")
		b.WriteString(strings.TrimSpace(r.DockerStatus))
	}
	b.WriteString(" | tmp: ")
	b.WriteString(r.SelectedTmpDir)
	return b.String()
}

// FailureDetails returns a verbose preflight summary including probe commands and outputs.
func FailureDetails(r Result, err error) string {
	var b strings.Builder
	b.WriteString(Summary(r))
	if err != nil {
		b.WriteString(" | error: ")
		b.WriteString(err.Error())
	}
	if len(r.Checks) > 0 {
		b.WriteString(" | probes:")
		for _, check := range r.Checks {
			b.WriteString(" [")
			b.WriteString(check.Name)
			b.WriteString(" cmd=\"")
			b.WriteString(check.Cmd)
			b.WriteString("\" exit=")
			b.WriteString(check.Exit)
			b.WriteString(" out=\"")
			b.WriteString(check.Out)
			b.WriteString("\"]")
		}
	}
	return b.String()
}
