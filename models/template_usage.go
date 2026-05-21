package models

import "strings"

// TemplateHostUsage describes a host that embeds a template's SQL in import queries.
type TemplateHostUsage struct {
	ProfileID    string
	ProfileName  string
	InPreImport  bool
	InPostImport bool
}

// TemplateBodyChanged reports whether template SQL body was edited.
func TemplateBodyChanged(oldBody, newBody string) bool {
	return strings.TrimSpace(oldBody) != strings.TrimSpace(newBody)
}

// SubstitutedTemplateSQL applies host placeholders to template SQL.
func SubstitutedTemplateSQL(body string, vars QueryVars) string {
	return strings.TrimSpace(SubstituteQuery(body, vars))
}

// FindProfilesUsingTemplate returns hosts whose pre/post import queries contain
// the previous template body (after placeholder substitution per host).
func FindProfilesUsingTemplate(profiles []Profile, oldTemplateBody string) []TemplateHostUsage {
	oldBody := strings.TrimSpace(oldTemplateBody)
	if oldBody == "" {
		return nil
	}
	var usages []TemplateHostUsage
	for _, p := range profiles {
		sub := SubstitutedTemplateSQL(oldBody, p.QueryVars())
		if sub == "" {
			continue
		}
		inPre := strings.Contains(p.PreImportQuery, sub)
		inPost := strings.Contains(p.PostImportQuery, sub)
		if !inPre && !inPost {
			continue
		}
		usages = append(usages, TemplateHostUsage{
			ProfileID:    p.ID,
			ProfileName:  p.Name,
			InPreImport:  inPre,
			InPostImport: inPost,
		})
	}
	return usages
}

// ReplaceTemplateInProfile swaps embedded old template SQL with new SQL for one host.
func ReplaceTemplateInProfile(p Profile, oldBody, newBody string) Profile {
	oldSub := SubstitutedTemplateSQL(oldBody, p.QueryVars())
	newSub := SubstitutedTemplateSQL(newBody, p.QueryVars())
	if oldSub == "" || oldSub == newSub {
		return p
	}
	if strings.Contains(p.PreImportQuery, oldSub) {
		p.PreImportQuery = strings.ReplaceAll(p.PreImportQuery, oldSub, newSub)
	}
	if strings.Contains(p.PostImportQuery, oldSub) {
		p.PostImportQuery = strings.ReplaceAll(p.PostImportQuery, oldSub, newSub)
	}
	return p
}

// ReplaceTemplateInProfiles updates all given hosts that embed oldBody.
func ReplaceTemplateInProfiles(profiles []Profile, oldBody, newBody string, profileIDs map[string]struct{}) []Profile {
	if len(profileIDs) == 0 {
		return profiles
	}
	out := append([]Profile(nil), profiles...)
	for i := range out {
		if _, ok := profileIDs[out[i].ID]; !ok {
			continue
		}
		out[i] = ReplaceTemplateInProfile(out[i], oldBody, newBody)
	}
	return out
}

func (u TemplateHostUsage) LocationLabel() string {
	switch {
	case u.InPreImport && u.InPostImport:
		return "before & after import"
	case u.InPreImport:
		return "before import"
	case u.InPostImport:
		return "after import"
	default:
		return "import queries"
	}
}
