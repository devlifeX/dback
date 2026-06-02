package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	coreapp "dback/internal/app"
)

func (u *UI) runAboutUpdateCheck() {
	if u.core == nil {
		return
	}
	u.updateStatus = "Checking for updates…"
	u.invalidate()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		info, err := u.core.CheckForUpdate(ctx, u.version)
		u.invalidate()
		if err != nil {
			u.updateStatus = "Update check failed"
			u.showError(fmt.Errorf("check for updates: %w", err))
			u.invalidate()
			return
		}

		if !info.Available {
			u.updateStatus = fmt.Sprintf("You are up to date (v%s).", u.version)
			u.showInfo("Up to date", fmt.Sprintf("DBack v%s is the latest release.", u.version))
			u.invalidate()
			return
		}

		u.updateStatus = fmt.Sprintf("Update available: v%s", info.LatestVersion)
		u.pendingUpdateInfo = info
		u.showUpdateAvailableDialog(info)
		u.invalidate()
	}()
}

func (u *UI) showUpdateAvailableDialog(info coreapp.UpdateInfo) {
	message := fmt.Sprintf("Current version: v%s\nLatest version: v%s", info.CurrentVersion, info.LatestVersion)
	if notes := strings.TrimSpace(info.ReleaseNotes); notes != "" {
		if len(notes) > 300 {
			notes = notes[:300] + "…"
		}
		message += "\n\n" + notes
	}

	u.showDialog(DialogState{
		Kind:     DialogUpdateAvailable,
		Title:    "Update available",
		Message:  message,
		OKLabel:  "Update now",
		OnOK:     u.runAboutApplyUpdate,
		OnCancel: func() {},
	})
}

func (u *UI) runAboutApplyUpdate() {
	info := u.pendingUpdateInfo
	if !info.Available || info.Asset.URL == "" {
		u.showError(fmt.Errorf("update information is unavailable; check again"))
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	u.updateApplyCancel = cancel
	u.showLoadingWithCancel("Updating DBack", "Preparing download…", func() {
		cancel()
		u.updateApplyCancel = nil
	})

	go func() {
		progress := func(stage string) {
			u.dialog.Message = stage
			u.invalidate()
		}

		err := u.core.ApplyUpdate(ctx, info, progress)
		u.updateApplyCancel = nil
		if ctx.Err() != nil {
			u.closeDialog()
			u.invalidate()
			return
		}
		if err != nil {
			u.closeDialog()
			u.showError(fmt.Errorf("apply update: %w", err))
			u.invalidate()
			return
		}

		// Linux install completes in-process; Windows exits before returning.
		u.closeDialog()
		u.updateStatus = fmt.Sprintf("Updated to v%s", info.LatestVersion)
		u.showInfo("Update installed", fmt.Sprintf("DBack was updated to v%s.", info.LatestVersion))
		u.invalidate()
	}()
}