package storemodel

import "dback/models"

func DetectTemplateConflicts(existing, imported []models.SQLTemplate) []TemplateConflict {
	byID := map[string]models.SQLTemplate{}
	byName := map[string]models.SQLTemplate{}
	for _, t := range existing {
		byID[t.ID] = t
		byName[t.Name] = t
	}
	var conflicts []TemplateConflict
	seen := map[string]bool{}
	for _, t := range imported {
		if ex, ok := byID[t.ID]; ok {
			key := "id:" + t.ID
			if !seen[key] {
				conflicts = append(conflicts, TemplateConflict{Imported: t, Existing: ex, Reason: "id"})
				seen[key] = true
			}
			continue
		}
		if ex, ok := byName[t.Name]; ok {
			key := "name:" + t.Name
			if !seen[key] {
				conflicts = append(conflicts, TemplateConflict{Imported: t, Existing: ex, Reason: "name"})
				seen[key] = true
			}
		}
	}
	if conflicts == nil {
		return []TemplateConflict{}
	}
	return conflicts
}

func MergeTemplates(existing, imported []models.SQLTemplate) []models.SQLTemplate {
	byID := map[string]int{}
	byName := map[string]int{}
	out := append([]models.SQLTemplate(nil), existing...)
	for i, t := range out {
		byID[t.ID] = i
		byName[t.Name] = i
	}
	for _, t := range imported {
		if idx, ok := byID[t.ID]; ok {
			out[idx] = t
			continue
		}
		if idx, ok := byName[t.Name]; ok {
			out[idx] = t
			continue
		}
		out = append(out, t)
		byID[t.ID] = len(out) - 1
		byName[t.Name] = len(out) - 1
	}
	return out
}

func MergeHistory(existing, imported []models.ExportRecord) []models.ExportRecord {
	seen := map[string]bool{}
	out := append([]models.ExportRecord(nil), existing...)
	for _, r := range out {
		if r.ID != "" {
			seen[r.ID] = true
		}
	}
	for _, r := range imported {
		if r.ID != "" && seen[r.ID] {
			continue
		}
		out = append(out, r)
		if r.ID != "" {
			seen[r.ID] = true
		}
	}
	return out
}

func MergeLogs(existing, imported []models.LogEntry) []models.LogEntry {
	seen := map[string]bool{}
	out := append([]models.LogEntry(nil), existing...)
	for _, e := range out {
		if e.ID != "" {
			seen[e.ID] = true
		}
	}
	for _, e := range imported {
		if e.ID != "" && seen[e.ID] {
			continue
		}
		out = append(out, e)
		if e.ID != "" {
			seen[e.ID] = true
		}
	}
	return out
}

func DetectProfileConflicts(existing, imported []models.Profile) []ProfileConflict {
	byID := map[string]models.Profile{}
	byName := map[string]models.Profile{}
	for _, p := range existing {
		byID[p.ID] = p
		byName[p.Name] = p
	}
	var conflicts []ProfileConflict
	seen := map[string]bool{}
	for _, p := range imported {
		if ex, ok := byID[p.ID]; ok {
			key := "id:" + p.ID
			if !seen[key] {
				conflicts = append(conflicts, ProfileConflict{Imported: p, Existing: ex, Reason: "id"})
				seen[key] = true
			}
			continue
		}
		if ex, ok := byName[p.Name]; ok {
			key := "name:" + p.Name
			if !seen[key] {
				conflicts = append(conflicts, ProfileConflict{Imported: p, Existing: ex, Reason: "name"})
				seen[key] = true
			}
		}
	}
	if conflicts == nil {
		return []ProfileConflict{}
	}
	return conflicts
}

func MergeProfiles(existing, imported []models.Profile) []models.Profile {
	byID := map[string]int{}
	byName := map[string]int{}
	out := append([]models.Profile(nil), existing...)
	for i, p := range out {
		byID[p.ID] = i
		byName[p.Name] = i
	}
	for _, p := range imported {
		if idx, ok := byID[p.ID]; ok {
			out[idx] = p
			continue
		}
		if idx, ok := byName[p.Name]; ok {
			out[idx] = p
			continue
		}
		out = append(out, p)
		byID[p.ID] = len(out) - 1
		byName[p.Name] = len(out) - 1
	}
	return out
}
