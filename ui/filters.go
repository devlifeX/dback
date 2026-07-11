package ui

import (
	"sort"
	"strings"

	"dback/models"
)

const groupFilterAll = ""

const (
	hostSortModifiedDesc = "modified_desc"
	hostSortModifiedAsc  = "modified_asc"
	hostSortNameDesc     = "name_desc"
	hostSortNameAsc      = "name_asc"
)

func hostSortOptions() (values, labels []string) {
	return []string{
			hostSortModifiedDesc,
			hostSortModifiedAsc,
			hostSortNameDesc,
			hostSortNameAsc,
		},
		[]string{
			"Time modified DESC",
			"Time modified ASC",
			"Name DESC",
			"Name ASC",
		}
}

func sortProfiles(profiles []models.Profile, sortKey string) []models.Profile {
	out := append([]models.Profile(nil), profiles...)
	switch sortKey {
	case hostSortModifiedAsc:
		sort.Slice(out, func(i, j int) bool {
			ti, tj := out[i].ModifiedAt(), out[j].ModifiedAt()
			if ti.Equal(tj) {
				return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
			}
			return ti.Before(tj)
		})
	case hostSortNameDesc:
		sort.Slice(out, func(i, j int) bool {
			ni, nj := strings.ToLower(out[i].Name), strings.ToLower(out[j].Name)
			if ni == nj {
				return out[i].ModifiedAt().After(out[j].ModifiedAt())
			}
			return ni > nj
		})
	case hostSortNameAsc:
		sort.Slice(out, func(i, j int) bool {
			ni, nj := strings.ToLower(out[i].Name), strings.ToLower(out[j].Name)
			if ni == nj {
				return out[i].ModifiedAt().After(out[j].ModifiedAt())
			}
			return ni < nj
		})
	default:
		sort.Slice(out, func(i, j int) bool {
			ti, tj := out[i].ModifiedAt(), out[j].ModifiedAt()
			if ti.Equal(tj) {
				return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
			}
			return ti.After(tj)
		})
	}
	return out
}

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

const backupTypeFilterAll = ""

func filterBackupsByType(records []models.ExportRecord, exportType string) []models.ExportRecord {
	if exportType == "" || exportType == backupTypeFilterAll {
		return records
	}
	var out []models.ExportRecord
	for _, r := range records {
		if string(r.EffectiveExportType()) == exportType {
			out = append(out, r)
		}
	}
	return out
}

func exportTypeLabel(t models.ExportType) string {
	switch t {
	case models.ExportTypeFiles:
		return "Files"
	default:
		return "DB"
	}
}

func sortedBackupTypeOptions() (values, labels []string) {
	return []string{backupTypeFilterAll, string(models.ExportTypeDatabase), string(models.ExportTypeFiles)},
		[]string{"All types", "Database", "Files"}
}

const backupFilterAll = ""
