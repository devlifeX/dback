package ui

import (
	"fmt"
	"strings"

	"dback/internal/store"
)

func (u *UI) importProfilesFromPath(path string, includeSecrets bool, passphrase string) {
	imported, conflicts, err := u.core.PreviewImportProfiles(path, includeSecrets, passphrase)
	if err != nil {
		u.showError(err)
		return
	}
	if len(conflicts) == 0 {
		if err := u.core.ImportProfiles(path, includeSecrets, passphrase); err != nil {
			u.showError(err)
			return
		}
		u.showInfo("Import complete", fmt.Sprintf("Merged %d host(s).", len(imported)))
		u.openHosts()
		return
	}

	msg := formatImportConflicts(conflicts)
	u.showConfirm("Import conflicts", msg, func() {
		if err := u.core.ImportProfiles(path, includeSecrets, passphrase); err != nil {
			u.showError(err)
			return
		}
		u.showInfo("Import complete", fmt.Sprintf("Merged %d host(s).", len(imported)))
		u.openHosts()
	})
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
