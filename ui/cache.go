package ui

import (
	"errors"
	"log"
	"strings"

	coreapp "dback/internal/app"
	"dback/internal/store"
	"dback/models"
)

type templateOptionCache struct {
	revision   uint64
	names      []string
	labels     []string
	nameToBody map[string]string
}

func (c *templateOptionCache) rebuild(revision uint64, templates []models.SQLTemplate) {
	if c.revision == revision && c.names != nil {
		return
	}
	c.revision = revision
	c.names = make([]string, 0, len(templates))
	c.labels = make([]string, 0, len(templates))
	c.nameToBody = make(map[string]string, len(templates))
	for _, t := range templates {
		c.names = append(c.names, t.Name)
		c.labels = append(c.labels, t.Name)
		c.nameToBody[t.Name] = t.Body
	}
	if len(c.names) == 0 {
		c.names = []string{"(no templates)"}
		c.labels = []string{"(no templates)"}
	}
}

type backupViewCache struct {
	dataRevision uint64
	hostFilter   string
	typeFilter   string
	records      []models.ExportRecord
	hostValues   []string
	hostLabels   []string
	typeValues   []string
	typeLabels   []string
}

func (c *backupViewCache) rebuild(u *UI) {
	if u.core == nil || !u.core.IsUnlocked() {
		return
	}
	rev := u.core.DataRevision()
	hostFilter := u.backupHostFilter
	typeFilter := u.backupTypeFilter
	if c.dataRevision == rev && c.hostFilter == hostFilter && c.typeFilter == typeFilter && c.records != nil {
		return
	}
	history := u.core.History()
	filtered := filterBackupsByHost(sortBackupsNewestFirst(history), hostFilter)
	c.records = filterBackupsByType(filtered, typeFilter)
	c.hostValues, c.hostLabels = sortedBackupHostOptions(u.core.Profiles())
	c.typeValues, c.typeLabels = sortedBackupTypeOptions()
	c.dataRevision = rev
	c.hostFilter = hostFilter
	c.typeFilter = typeFilter
}

func unlockErrorMessage(err error) string {
	switch {
	case errors.Is(err, store.ErrWrongMasterKey):
		return "Wrong master key. Try again."
	case errors.Is(err, store.ErrMasterKeyRequired):
		return "Master key is required."
	case errors.Is(err, store.ErrVaultExists):
		return "Vault already exists. Unlock with your master key."
	case errors.Is(err, store.ErrVaultNotFound):
		return "No vault found. Create a master key first."
	default:
		return sanitizeError(err)
	}
}

func (u *UI) tryUnlockSilent(passphrase string) bool {
	if u.core == nil || u.unlocked {
		return u.unlocked
	}
	// Silent auto-unlock only applies to existing vaults/legacy data.
	// Vault creation always requires an explicit button press or Enter.
	if !u.core.HasVault() && !u.core.HasLegacyPlaintext() {
		return false
	}
	passphrase = strings.TrimSpace(passphrase)
	err := u.core.Unlock(passphrase)
	if err != nil {
		log.Printf("tryUnlockSilent: failed (hasVault=%v hasLegacy=%v): %v", u.core.HasVault(), u.core.HasLegacyPlaintext(), err)
		return false
	}
	u.completeUnlock()
	return true
}

func (u *UI) tryUnlockWithFeedback(passphrase string) {
	if u.core == nil {
		return
	}
	passphrase = strings.TrimSpace(passphrase)
	if !u.core.HasVault() && !u.core.HasLegacyPlaintext() {
		confirm := strings.TrimSpace(editorText(&u.loginConfirmPassword))
		if len(passphrase) < 4 {
			u.loginError = "Master key must be at least 4 characters."
			u.invalidate()
			return
		}
		if passphrase != confirm {
			u.loginError = "Master keys do not match."
			u.invalidate()
			return
		}
	}
	var err error
	if u.core.HasVault() || u.core.HasLegacyPlaintext() {
		err = u.core.Unlock(passphrase)
	} else {
		err = u.core.CreateVault(passphrase)
	}
	if err != nil {
		log.Printf("tryUnlockWithFeedback: failed (hasVault=%v hasLegacy=%v): %v", u.core.HasVault(), u.core.HasLegacyPlaintext(), err)
		u.loginError = unlockErrorMessage(err)
		u.invalidate()
		return
	}
	u.completeUnlock()
}

func (u *UI) completeUnlock() {
	u.unlocked = true
	u.loginError = ""
	u.loginPassword.SetText("")
	u.loginConfirmPassword.SetText("")
	u.invalidateBackupCache()
	u.invalidate()
}

func (u *UI) invalidateBackupCache() {
	u.backupCache = backupViewCache{}
}

func ensureCoreUnlocked(a *coreapp.App, passphrase string) error {
	if a.IsUnlocked() {
		return nil
	}
	if a.HasVault() {
		return a.Unlock(passphrase)
	}
	return a.CreateVault(passphrase)
}
