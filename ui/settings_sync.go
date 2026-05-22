package ui

import (
	"context"

	"dback/internal/store"
	"dback/models"

	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type SyncForm struct {
	Endpoint    widget.Editor
	Region      widget.Editor
	Bucket      widget.Editor
	AccessKeyID widget.Editor
	SecretKey   widget.Editor
	UseSSL      widget.Bool

	secretVisible bool
	secretToggle  widget.Clickable
}

func newSyncForm() *SyncForm {
	f := &SyncForm{}
	f.Endpoint.SingleLine = true
	f.Region.SingleLine = true
	f.Bucket.SingleLine = true
	f.AccessKeyID.SingleLine = true
	f.SecretKey.SingleLine = true
	f.UseSSL.Value = true
	return f
}

func (f *SyncForm) load(settings *models.SyncSettings) {
	if settings == nil {
		f.Endpoint.SetText("")
		f.Region.SetText("")
		f.Bucket.SetText("")
		f.AccessKeyID.SetText("")
		f.SecretKey.SetText("")
		f.UseSSL.Value = true
		return
	}
	f.Endpoint.SetText(settings.Endpoint)
	f.Region.SetText(settings.Region)
	f.Bucket.SetText(settings.Bucket)
	f.AccessKeyID.SetText(settings.AccessKeyID)
	f.SecretKey.SetText(settings.SecretKey)
	f.UseSSL.Value = settings.UseSSL
}

func (f *SyncForm) settings() models.SyncSettings {
	return models.SyncSettings{
		Endpoint:    editorText(&f.Endpoint),
		Region:      editorText(&f.Region),
		Bucket:      editorText(&f.Bucket),
		AccessKeyID: editorText(&f.AccessKeyID),
		SecretKey:   editorText(&f.SecretKey),
		UseSSL:      f.UseSSL.Value,
	}
}

func syncSettingsEqual(a, b models.SyncSettings) bool {
	return a.Endpoint == b.Endpoint &&
		a.Region == b.Region &&
		a.Bucket == b.Bucket &&
		a.AccessKeyID == b.AccessKeyID &&
		a.SecretKey == b.SecretKey &&
		a.UseSSL == b.UseSSL
}

func syncSettingsEmpty(s models.SyncSettings) bool {
	return s.Endpoint == "" && s.Region == "" && s.Bucket == "" &&
		s.AccessKeyID == "" && s.SecretKey == ""
}

func (u *UI) syncFormDirty() bool {
	if u.syncForm == nil {
		return false
	}
	current := u.syncForm.settings()
	if u.syncSavedBaseline == nil {
		return !syncSettingsEmpty(current)
	}
	return !syncSettingsEqual(current, *u.syncSavedBaseline)
}

func (u *UI) setSyncSavedBaseline(settings models.SyncSettings) {
	s := settings
	u.syncSavedBaseline = &s
}

func (u *UI) reloadSyncFormFromSaved() {
	if u.syncForm == nil || u.core == nil {
		return
	}
	settings, err := u.core.SyncSettings()
	if err != nil {
		return
	}
	u.syncForm.load(settings)
	if settings != nil {
		u.setSyncSavedBaseline(*settings)
	} else {
		u.syncSavedBaseline = nil
	}
}

func (u *UI) loadSyncFormFromCore() {
	u.reloadSyncFormFromSaved()
	u.syncConnectionOK = false
	u.refreshSyncActivity()
}

func (u *UI) refreshSyncActivity() {
	if u.core == nil {
		return
	}
	activity, err := u.core.SyncActivity()
	if err != nil {
		return
	}
	u.syncActivity = activity
}

func (u *UI) layoutSettingsSync(gtx layout.Context, th *material.Theme, theme *AppTheme) layout.Dimensions {
	if u.syncForm == nil {
		u.syncForm = newSyncForm()
		u.loadSyncFormFromCore()
	}
	f := u.syncForm
	f.UseSSL.Update(gtx)

	dirty := u.syncFormDirty()
	if u.syncConnectionOK && dirty {
		u.syncConnectionOK = false
	}
	showPushPull := u.syncConnectionOK

	return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Subtitle1(th, "Remote Sync")
				lbl.Color = theme.Text
				return lbl.Layout(gtx)
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return mutedLabel(gtx, th, theme, "Push or pull encrypted app settings to S3-compatible storage (dback/app-data.json). Data is encrypted with your vault master key—the same key used to unlock DBack.")
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return labeledField(gtx, th, theme, "Endpoint", func(gtx layout.Context) layout.Dimensions {
					return editorField(gtx, th, theme, &f.Endpoint, "s3.amazonaws.com or minio.example.com:9000")
				})
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return labeledField(gtx, th, theme, "Region", func(gtx layout.Context) layout.Dimensions {
					return editorField(gtx, th, theme, &f.Region, "us-east-1 (optional)")
				})
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return labeledField(gtx, th, theme, "Bucket", func(gtx layout.Context) layout.Dimensions {
					return editorField(gtx, th, theme, &f.Bucket, "my-bucket")
				})
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return labeledField(gtx, th, theme, "Access Key ID", func(gtx layout.Context) layout.Dimensions {
					return editorField(gtx, th, theme, &f.AccessKeyID, "")
				})
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return labeledField(gtx, th, theme, "Secret Key", func(gtx layout.Context) layout.Dimensions {
					return passwordField(gtx, th, theme, &f.SecretKey, "", &f.secretVisible, &f.secretToggle)
				})
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return checkboxField(gtx, th, theme, &f.UseSSL, "Use SSL (HTTPS)")
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				var actions []layout.FlexChild
				actions = append(actions, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if !dirty {
						return disabledButton(gtx, th, theme, "Save")
					}
					return successButton(gtx, th, theme, &u.saveSyncBtn, "Save", u.saveSyncSettings)
				}))
				actions = append(actions, layout.Rigid(hgap(theme)))
				actions = append(actions, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return secondaryButton(gtx, th, theme, &u.testSyncBtn, "Test Connection", u.testSyncConnection)
				}))
				if showPushPull {
					actions = append(actions, layout.Rigid(hgap(theme)))
					actions = append(actions, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return primaryButton(gtx, th, theme, &u.syncPushBtn, "Push", u.syncPush)
					}))
					actions = append(actions, layout.Rigid(hgap(theme)))
					actions = append(actions, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return primaryButton(gtx, th, theme, &u.syncPullBtn, "Pull", u.syncPull)
					}))
				}
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, actions...)
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return u.layoutSyncActivityLog(gtx, th, theme)
			}),
		)
	})
}

func (u *UI) layoutSyncActivityLog(gtx layout.Context, th *material.Theme, theme *AppTheme) layout.Dimensions {
	activity := u.syncActivity
	pushLine := "Last push: never"
	if !activity.LastPushAt.IsZero() {
		pushLine = "Last push: " + formatRelativeTime(activity.LastPushAt)
	}
	pullLine := "Last pull: never"
	if !activity.LastPullAt.IsZero() {
		pullLine = "Last pull: " + formatRelativeTime(activity.LastPullAt)
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return divider(gtx, theme)
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Subtitle2(th, "Sync log")
			lbl.Color = theme.Text
			return lbl.Layout(gtx)
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return mutedLabel(gtx, th, theme, pushLine)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return mutedLabel(gtx, th, theme, pullLine)
		}),
	)
}

func (u *UI) saveSyncSettings() {
	cfg := u.syncForm.settings()
	if err := u.core.SaveSyncSettings(cfg); err != nil {
		u.showError(err)
		return
	}
	u.reloadSyncFormFromSaved()
	u.syncConnectionOK = false
	u.invalidate()
}

func (u *UI) testSyncConnection() {
	cfg := u.syncForm.settings()
	if err := u.core.SaveSyncSettings(cfg); err != nil {
		u.showError(err)
		return
	}
	u.reloadSyncFormFromSaved()
	u.showLoading("Testing connection", "Connecting to S3...")
	go func() {
		saved, err := u.core.SyncSettings()
		if err != nil {
			u.closeDialog()
			u.showError(err)
			return
		}
		if saved == nil {
			u.closeDialog()
			u.showError(store.ErrSyncNotConfigured)
			return
		}
		err = u.core.TestSyncConnection(context.Background(), *saved)
		u.closeDialog()
		if err != nil {
			u.syncConnectionOK = false
			u.invalidate()
			u.showError(err)
			return
		}
		u.syncConnectionOK = true
		u.invalidate()
		u.showInfo("Connection OK", "S3 connection test succeeded.")
	}()
}

func (u *UI) syncPush() {
	if err := u.core.SaveSyncSettings(u.syncForm.settings()); err != nil {
		u.showError(err)
		return
	}
	u.reloadSyncFormFromSaved()
	u.runSyncPush()
}

func (u *UI) runSyncPush() {
	u.showLoading("Sync push", "Uploading encrypted app data...")
	go func() {
		err := u.core.SyncPush(context.Background())
		u.closeDialog()
		if err != nil {
			u.syncConnectionOK = false
			u.invalidate()
			u.showError(err)
			return
		}
		u.refreshSyncActivity()
		u.invalidate()
		u.showInfo("Push complete", "App settings were uploaded to dback/app-data.json.")
	}()
}

func (u *UI) syncPull() {
	if err := u.core.SaveSyncSettings(u.syncForm.settings()); err != nil {
		u.showError(err)
		return
	}
	u.reloadSyncFormFromSaved()
	u.showLoading("Sync pull", "Downloading from S3...")
	go func() {
		raw, err := u.core.SyncDownload(context.Background())
		u.closeDialog()
		u.invalidate()
		if err != nil {
			u.syncConnectionOK = false
			u.showError(err)
			return
		}
		u.showConfirm("Load remote settings?", "Remote app settings were downloaded from S3. Load them into this device?", func() {
			u.runSyncImport(raw)
		})
	}()
}

func (u *UI) runSyncImport(raw []byte) {
	u.showLoading("Sync pull", "Decrypting app data...")
	go func() {
		imported, profileConflicts, templateConflicts, err := u.core.PreviewSyncImport(raw)
		u.closeDialog()
		u.invalidate()
		if err != nil {
			u.showError(err)
			return
		}
		u.applySyncImport(imported, profileConflicts, templateConflicts)
	}()
}

func (u *UI) applySyncImport(imported store.AppImportData, profileConflicts []store.ProfileConflict, templateConflicts []store.TemplateConflict) {
	apply := func() {
		if err := u.core.ImportAppDataFromBundle(imported); err != nil {
			u.showError(err)
			return
		}
		if err := u.core.RecordSyncPull(); err != nil {
			u.showError(err)
			return
		}
		u.reloadSyncFormFromSaved()
		u.refreshSyncActivity()
		u.invalidate()
		u.showInfo("Pull complete", summarizeAppImport(imported))
	}

	if len(profileConflicts) == 0 && len(templateConflicts) == 0 {
		apply()
		return
	}

	msg := formatAppImportConflicts(profileConflicts, templateConflicts)
	u.showConfirm("Sync conflicts", msg, apply)
}
