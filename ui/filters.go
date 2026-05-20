package ui

import (
	"sort"
	"strings"

	"dback/models"
)

const groupFilterAll = ""

func normalizeGroup(group string) string {
	group = strings.TrimSpace(group)
	if group == "" {
		return "Default"
	}
	return group
}

func filterProfiles(profiles []models.Profile, search, groupFilter string) []models.Profile {
	q := strings.ToLower(strings.TrimSpace(search))
	var out []models.Profile
	for _, p := range profiles {
		if groupFilter != groupFilterAll && normalizeGroup(p.Group) != groupFilter {
			continue
		}
		if q == "" {
			out = append(out, p)
			continue
		}
		if strings.Contains(strings.ToLower(p.Name), q) ||
			strings.Contains(strings.ToLower(p.Host), q) ||
			strings.Contains(strings.ToLower(normalizeGroup(p.Group)), q) ||
			strings.Contains(strings.ToLower(p.TargetDBName), q) {
			out = append(out, p)
		}
	}
	return out
}

func collectGroups(profiles []models.Profile) []string {
	set := map[string]struct{}{}
	for _, p := range profiles {
		set[normalizeGroup(p.Group)] = struct{}{}
	}
	groups := make([]string, 0, len(set))
	for g := range set {
		groups = append(groups, g)
	}
	sort.Strings(groups)
	return groups
}

func sortBackupsNewestFirst(records []models.ExportRecord) []models.ExportRecord {
	out := append([]models.ExportRecord(nil), records...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].ExportDate.After(out[j].ExportDate)
	})
	return out
}

func filterBackupsByHost(records []models.ExportRecord, profileID string) []models.ExportRecord {
	if profileID == "" || profileID == backupFilterAll {
		return records
	}
	var out []models.ExportRecord
	for _, r := range records {
		if r.ProfileID == profileID {
			out = append(out, r)
		}
	}
	return out
}

const backupFilterAll = ""
