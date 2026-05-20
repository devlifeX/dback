package ui

import (
	"errors"
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
	records      []models.ExportRecord
	hostValues   []string
	hostLabels   []string
}

func (c *backupViewCache) rebuild(u *UI) {
	if u.core == nil || !u.core.IsUnlocked() {
		return
	}
	rev := u.core.DataRevision()
	filter := u.backupHostFilter
	if c.dataRevision == rev && c.hostFilter == filter && c.records != nil {
		return
	}
	history := u.core.History()
	c.records = filterBackupsByHost(sortBackupsNewestFirst(history), filter)
	c.hostValues, c.hostLabels = sortedBackupHostOptions(u.core.Profiles())
	c.dataRevision = rev
	c.hostFilter = filter
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
		return err.Error()
	}
}

func (u *UI) tryUnlock(passphrase string) {
	if u.core == nil {
		return
	}
	passphrase = strings.TrimSpace(passphrase)
	var err error
	if u.core.HasVault() || u.core.HasLegacyPlaintext() {
		err = u.core.Unlock(passphrase)
	} else {
		err = u.core.CreateVault(passphrase)
	}
	if err != nil {
		u.loginError = unlockErrorMessage(err)
		u.invalidate()
		return
	}
	u.unlocked = true
	u.loginError = ""
	u.loginPassword.SetText("")
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
