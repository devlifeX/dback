package update

import (
	"strconv"
	"strings"
)

// NormalizeVersion strips a leading "v" and returns major.minor.patch segments.
func NormalizeVersion(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "v")
	if raw == "" {
		return "0.0.0"
	}

	parts := strings.Split(raw, ".")
	for len(parts) < 3 {
		parts = append(parts, "0")
	}
	return strings.Join(parts[:3], ".")
}

// CompareVersions returns -1 if a<b, 0 if equal, 1 if a>b.
func CompareVersions(a, b string) int {
	aParts := strings.Split(NormalizeVersion(a), ".")
	bParts := strings.Split(NormalizeVersion(b), ".")
	for i := 0; i < 3; i++ {
		av, _ := strconv.Atoi(aParts[i])
		bv, _ := strconv.Atoi(bParts[i])
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

// IsNewer reports whether candidate is newer than current.
func IsNewer(current, candidate string) bool {
	return CompareVersions(candidate, current) > 0
}
