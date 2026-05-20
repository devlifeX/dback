package debug

import (
	"fmt"
	"os"
	rdebug "runtime/debug"
	"strings"
	"sync"
)

var (
	Enabled bool
	mu      sync.Mutex
)

func Enable() {
	Enabled = true
}

func EnabledFromEnv() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("DBACK_DEBUG")))
	return v == "1" || v == "true" || v == "yes"
}

func Log(level, action, status, details, profileName, operationID, errText string) {
	if !Enabled {
		return
	}
	mu.Lock()
	defer mu.Unlock()

	line := fmt.Sprintf("[%s] %s", level, action)
	if status != "" {
		line += " " + status
	}
	if profileName != "" {
		line += " profile=" + profileName
	}
	if operationID != "" {
		line += " op=" + operationID
	}
	if details != "" {
		line += " details=" + details
	}
	if errText != "" {
		line += " error=" + quote(errText)
	}
	fmt.Fprintln(os.Stderr, line)
}

func Errorf(format string, args ...any) {
	if !Enabled {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	fmt.Fprintf(os.Stderr, "[ERROR] "+format+"\n", args...)
}

func Stack() string {
	return string(rdebug.Stack())
}

func quote(s string) string {
	if strings.ContainsAny(s, " \t\n\"") {
		return fmt.Sprintf("%q", s)
	}
	return s
}
