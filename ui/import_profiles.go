package ui

import (
	"fmt"
	"strings"

	"dback/internal/store"
)

func (u *UI) importAppDataFromPath(path string, includeSecrets bool, passphrase string) {
	imported, profileConflicts, templateConflicts, err := u.core.PreviewImportAppData(path, includeSecrets, passphrase)
	if err != nil {
		u.showError(err)
		return
	}
	if len(profileConflicts) == 0 && len(templateConflicts) == 0 {
		if err := u.core.ImportAppData(path, includeSecrets, passphrase); err != nil {
			u.showError(err)
			return
		}
		u.showInfo("Import complete", summarizeAppImport(imported))
		u.openHosts()
		return
	}

	msg := formatAppImportConflicts(profileConflicts, templateConflicts)
	u.showConfirm("Import conflicts", msg, func() {
		if err := u.core.ImportAppData(path, includeSecrets, passphrase); err != nil {
			u.showError(err)
			return
		}
		u.showInfo("Import complete", summarizeAppImport(imported))
		u.openHosts()
	})
}

func summarizeAppImport(data store.AppImportData) string {
	return fmt.Sprintf("Merged %d host(s), %d template(s), %d history record(s), %d log entry(ies).",
		len(data.Profiles), len(data.Templates), len(data.History), len(data.Logs))
}

func formatAppImportConflicts(profiles []store.ProfileConflict, templates []store.TemplateConflict) string {
	var b strings.Builder
	b.WriteString("The following items will replace existing entries:\n\n")
	for _, c := range profiles {
		switch c.Reason {
		case "id":
			b.WriteString(fmt.Sprintf("- Host %s (same ID)\n", c.Imported.Name))
		case "name":
			b.WriteString(fmt.Sprintf("- Host %s (same name)\n", c.Imported.Name))
		default:
			b.WriteString(fmt.Sprintf("- Host %s\n", c.Imported.Name))
		}
	}
	for _, c := range templates {
		switch c.Reason {
		case "id":
			b.WriteString(fmt.Sprintf("- Template %s (same ID)\n", c.Imported.Name))
		case "name":
			b.WriteString(fmt.Sprintf("- Template %s (same name)\n", c.Imported.Name))
		default:
			b.WriteString(fmt.Sprintf("- Template %s\n", c.Imported.Name))
		}
	}
	b.WriteString("\nContinue?")
	return b.String()
}

// importProfilesFromPath supports legacy profile-only bundles.
func (u *UI) importProfilesFromPath(path string, includeSecrets bool, passphrase string) {
	u.importAppDataFromPath(path, includeSecrets, passphrase)
}

func formatImportConflicts(conflicts []store.ProfileConflict) string {
	var b strings.Builder
	b.WriteString("The following hosts will replace existing entries:\n\n")
	for _, c := range conflicts {
		switch c.Reason {
		case "id":
			b.WriteString(fmt.Sprintf("- %s (same ID)\n", c.Imported.Name))
		case "name":
			b.WriteString(fmt.Sprintf("- %s (same name)\n", c.Imported.Name))
		default:
			b.WriteString(fmt.Sprintf("- %s\n", c.Imported.Name))
		}
	}
	b.WriteString("\nContinue?")
	return b.String()
}
