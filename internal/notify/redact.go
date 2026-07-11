package notify

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

const maxMessageLen = 4000

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(password|passwd|secret|token|api[_-]?key|authorization)\s*[:=]\s*\S+`),
	regexp.MustCompile(`(?i)bearer\s+[a-z0-9\-._~+/]+=*`),
}

func SanitizeText(s string) string {
	s = strings.TrimSpace(s)
	for _, re := range secretPatterns {
		s = re.ReplaceAllString(s, "$1=[redacted]")
	}
	if utf8.RuneCountInString(s) > maxMessageLen {
		runes := []rune(s)
		s = string(runes[:maxMessageLen]) + "…"
	}
	return s
}

func SanitizeError(err string) string {
	if err == "" {
		return ""
	}
	return SanitizeText(err)
}
