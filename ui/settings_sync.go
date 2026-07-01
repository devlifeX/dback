package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"dback/internal/store"
	"dback/models"

	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type RemoteDestinationForm struct {
	Name        widget.Editor
	Endpoint    widget.Editor
	Region      widget.Editor
	Bucket      widget.Editor
	AccessKeyID widget.Editor
	SecretKey   widget.Editor
	UseSSL      widget.Bool

	secretVisible bool
	secretToggle  widget.Clickable
}

func newRemoteDestinationForm() *RemoteDestinationForm {
	f := &RemoteDestinationForm{}
	f.Name.SingleLine = true
	f.Endpoint.SingleLine = true
	f.Region.SingleLine = true
	f.Bucket.SingleLine = true
	f.AccessKeyID.SingleLine = true
	f.SecretKey.SingleLine = true
	f.UseSSL.Value = true
	return f
}

func (f *RemoteDestinationForm) load(dest *models.RemoteDestination) {
	if dest == nil {
		f.Name.SetText("")
		f.Endpoint.SetText("")
		f.Region.SetText("")
		f.Bucket.SetText("")
		f.AccessKeyID.SetText("")
		f.SecretKey.SetText("")
		f.UseSSL.Value = true
		return
	}
	f.Name.SetText(dest.Name)
	if dest.S3 != nil {
		f.Endpoint.SetText(dest.S3.Endpoint)
		f.Region.SetText(dest.S3.Region)
		f.Bucket.SetText(dest.S3.Bucket)
		f.AccessKeyID.SetText(dest.S3.AccessKeyID)
		f.SecretKey.SetText(dest.S3.SecretKey)
		f.UseSSL.Value = dest.S3.UseSSL
	}
}

func (f *RemoteDestinationForm) destination(id string) models.RemoteDestination {
	return models.RemoteDestination{
		ID:   id,
		Name: strings.TrimSpace(editorText(&f.Name)),
		Type: models.RemoteProviderS3,
		S3: &models.S3DestinationConfig{
			Endpoint:    editorText(&f.Endpoint),
			Region:      editorText(&f.Region),
			Bucket:      editorText(&f.Bucket),
			AccessKeyID: editorText(&f.AccessKeyID),
			SecretKey:   editorText(&f.SecretKey),
			UseSSL:      f.UseSSL.Value,
		},
	}
}

func (u *UI) refreshSyncDestinations() {
	if u.core == nil {
		return
	}
	dests, err := u.core.ListRemoteDestinations()
	if err != nil {
		return
	}
	u.syncDestinations = dests
	appID, err := u.core.AppSettingsDestinationID()
	if err == nil {
		u.syncAppSettingsDestID = appID
	}
}

func (u *UI) loadSyncFormFromCore() {
	if u.syncDestForm == nil {
		u.syncDestForm = newRemoteDestinationForm()
	}
	u.refreshSyncDestinations()
	u.syncConnectionOK = false
	u.syncShowDestEditor = false
	u.syncEditingDestID = ""
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
	if u.syncDestForm == nil {
		u.loadSyncFormFromCore()
	}

	return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Subtitle1(th, "Remote Sync")
				lbl.Color = theme.Text
				return lbl.Layout(gtx)
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return mutedLabel(gtx, th, theme, "Manage S3-compatible destinations. App settings sync uses one destination; backup files are configured per host.")
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return u.layoutRemoteDestinationsList(gtx, th, theme)
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if !u.syncShowDestEditor {
					return layout.Dimensions{}
				}
				return u.layoutRemoteDestinationEditor(gtx, th, theme)
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return u.layoutAppSettingsSync(gtx, th, theme)
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return u.layoutSyncActivityLog(gtx, th, theme)
			}),
		)
	})
}

func (u *UI) layoutRemoteDestinationsList(gtx layout.Context, th *material.Theme, theme *AppTheme) layout.Dimensions {
	if u.syncDestEditBtns == nil {
		u.syncDestEditBtns = map[string]*widget.Clickable{}
	}
	if u.syncDestDeleteBtns == nil {
		u.syncDestDeleteBtns = map[string]*widget.Clickable{}
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Subtitle2(th, "Remote Destinations")
					lbl.Color = theme.Text
					return lbl.Layout(gtx)
				}),
				layout.Flexed(1, layout.Spacer{}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return primaryButton(gtx, th, theme, &u.syncAddDestBtn, "+ S3", func() {
						u.syncShowDestEditor = true
						u.syncEditingDestID = ""
						u.syncDestForm.load(nil)
						u.syncConnectionOK = false
						u.invalidate()
					})
				}),
			)
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if len(u.syncDestinations) == 0 {
				return mutedLabel(gtx, th, theme, "No destinations yet. Add an S3 destination to get started.")
			}
			var rows []layout.FlexChild
			for _, dest := range u.syncDestinations {
				dest := dest
				editBtn, ok := u.syncDestEditBtns[dest.ID]
				if !ok {
					editBtn = new(widget.Clickable)
					u.syncDestEditBtns[dest.ID] = editBtn
				}
				delBtn, ok := u.syncDestDeleteBtns[dest.ID]
				if !ok {
					delBtn = new(widget.Clickable)
					u.syncDestDeleteBtns[dest.ID] = delBtn
				}
				summary := destSummary(dest)
				rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									lbl := material.Body1(th, dest.Name)
									lbl.Color = theme.Text
									return lbl.Layout(gtx)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return mutedLabel(gtx, th, theme, summary)
								}),
							)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return secondaryButton(gtx, th, theme, editBtn, "Edit", func() {
								u.syncShowDestEditor = true
								u.syncEditingDestID = dest.ID
								d := dest
								u.syncDestForm.load(&d)
								u.syncConnectionOK = false
								u.invalidate()
							})
						}),
						layout.Rigid(hgap(theme)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return dangerButton(gtx, th, theme, delBtn, "Delete", func() {
								u.confirmDeleteDestination(dest)
							})
						}),
					)
				}))
				rows = append(rows, layout.Rigid(vgap(theme)))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
		}),
	)
}

func destSummary(dest models.RemoteDestination) string {
	if dest.S3 == nil {
		return string(dest.Type)
	}
	return fmt.Sprintf("S3 · %s · %s", dest.S3.Bucket, dest.S3.Endpoint)
}

func (u *UI) layoutRemoteDestinationEditor(gtx layout.Context, th *material.Theme, theme *AppTheme) layout.Dimensions {
	f := u.syncDestForm
	f.UseSSL.Update(gtx)
	title := "Add S3 Destination"
	if u.syncEditingDestID != "" {
		title = "Edit S3 Destination"
	}
	return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Subtitle2(th, title)
				lbl.Color = theme.Text
				return lbl.Layout(gtx)
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return labeledField(gtx, th, theme, "Name", func(gtx layout.Context) layout.Dimensions {
					return editorField(gtx, th, theme, &f.Name, "Production S3")
				})
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
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return successButton(gtx, th, theme, &u.syncSaveDestBtn, "Save", u.saveRemoteDestination)
					}),
					layout.Rigid(hgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return secondaryButton(gtx, th, theme, &u.syncTestDestBtn, "Test Connection", u.testRemoteDestination)
					}),
					layout.Rigid(hgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return secondaryButton(gtx, th, theme, &u.syncCancelDestBtn, "Cancel", func() {
							u.syncShowDestEditor = false
							u.syncEditingDestID = ""
							u.syncConnectionOK = false
							u.invalidate()
						})
					}),
				)
			}),
		)
	})
}

func (u *UI) layoutAppSettingsSync(gtx layout.Context, th *material.Theme, theme *AppTheme) layout.Dimensions {
	values := make([]string, 0, len(u.syncDestinations))
	labels := make([]string, 0, len(u.syncDestinations))
	for _, d := range u.syncDestinations {
		values = append(values, d.ID)
		labels = append(labels, d.Name)
	}
	showPushPull := u.syncConnectionOK && u.syncAppSettingsDestID != ""

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return divider(gtx, theme)
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Subtitle2(th, "App Settings Sync")
			lbl.Color = theme.Text
			return lbl.Layout(gtx)
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return mutedLabel(gtx, th, theme, "Push or pull encrypted app settings (dback/app-data.json) to the selected destination.")
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if len(values) == 0 {
				return mutedLabel(gtx, th, theme, "Add a destination first.")
			}
			if u.syncAppSettingsSelect == nil {
				u.syncAppSettingsSelect = new(widget.Enum)
			}
			if u.syncAppSettingsDestID != "" {
				u.syncAppSettingsSelect.Value = u.syncAppSettingsDestID
			}
			if u.syncAppSettingsDropdown.ItemBtns == nil {
				u.syncAppSettingsDropdown.ItemBtns = map[string]*widget.Clickable{}
			}
			return dropdownField(gtx, th, theme, "Destination for app settings", DropdownOptions{
				Values: values,
				Labels: labels,
			}, &u.syncAppSettingsSelect.Value, &u.syncAppSettingsDropdown.Open, &u.syncAppSettingsDropdown.Toggle, &u.syncAppSettingsDropdown.List, u.syncAppSettingsDropdown.ItemBtns, u.invalidate, func(id string) {
				if err := u.core.SetAppSettingsDestinationID(id); err != nil {
					u.showError(err)
					return
				}
				u.syncAppSettingsDestID = id
				u.syncConnectionOK = false
				u.invalidate()
			})
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if u.syncAppSettingsDestID == "" {
				return layout.Dimensions{}
			}
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return secondaryButton(gtx, th, theme, &u.testSyncBtn, "Test Connection", u.testAppSettingsDestination)
				}),
				layout.Rigid(hgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if !showPushPull {
						return layout.Dimensions{}
					}
					return dangerButton(gtx, th, theme, &u.syncPushBtn, "Push", u.syncPush)
				}),
				layout.Rigid(hgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if !showPushPull {
						return layout.Dimensions{}
					}
					return primaryButton(gtx, th, theme, &u.syncPullBtn, "Pull", u.syncPull)
				}),
			)
		}),
	)
}

func (u *UI) saveRemoteDestination() {
	dest := u.syncDestForm.destination(u.syncEditingDestID)
	if dest.ID == "" {
		dest.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	if err := u.core.SaveRemoteDestination(dest); err != nil {
		u.showError(err)
		return
	}
	u.syncEditingDestID = dest.ID
	u.refreshSyncDestinations()
	u.syncShowDestEditor = false
	u.syncConnectionOK = false
	u.invalidate()
}

func (u *UI) testRemoteDestination() {
	dest := u.syncDestForm.destination(u.syncEditingDestID)
	if dest.ID == "" {
		dest.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	if err := u.core.SaveRemoteDestination(dest); err != nil {
		u.showError(err)
		return
	}
	u.syncEditingDestID = dest.ID
	u.refreshSyncDestinations()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	u.showLoadingWithCancel("Testing connection", "Connecting to S3...", cancel)
	go func() {
		defer cancel()
		saved, err := u.core.RemoteDestinationByID(dest.ID)
		if err != nil {
			u.closeDialog()
			u.showError(err)
			return
		}
		err = u.core.TestRemoteDestination(ctx, saved)
		if errors.Is(err, context.Canceled) {
			return
		}
		u.closeDialog()
		if err != nil {
			u.syncConnectionOK = false
			u.invalidate()
			if errors.Is(err, context.DeadlineExceeded) {
				u.showError(fmt.Errorf("connection timed out after 10 seconds"))
				return
			}
			u.showError(err)
			return
		}
		u.syncConnectionOK = true
		u.invalidate()
		u.showInfo("Connection OK", "S3 connection test succeeded.")
	}()
}

func (u *UI) testAppSettingsDestination() {
	if u.syncAppSettingsDestID == "" {
		u.showError(store.ErrSyncNotConfigured)
		return
	}
	dest, err := u.core.RemoteDestinationByID(u.syncAppSettingsDestID)
	if err != nil {
		u.showError(err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	u.showLoadingWithCancel("Testing connection", "Connecting to S3...", cancel)
	go func() {
		defer cancel()
		err := u.core.TestRemoteDestination(ctx, dest)
		if errors.Is(err, context.Canceled) {
			return
		}
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

func (u *UI) confirmDeleteDestination(dest models.RemoteDestination) {
	usage, err := u.core.DestinationUsage(dest.ID)
	if err != nil {
		u.showError(err)
		return
	}
	if !usage.UsedForAppSettings && len(usage.ProfileIDs) == 0 {
		u.showDialog(DialogState{
			Kind:    DialogConfirm,
			Title:   "Delete destination?",
			Message: fmt.Sprintf("Delete %q?", dest.Name),
			OnOK: func() {
				if err := u.core.DeleteRemoteDestination(dest.ID, false); err != nil {
					u.showError(err)
					return
				}
				u.refreshSyncDestinations()
				u.invalidate()
			},
		})
		return
	}
	msg := fmt.Sprintf("%q is in use.", dest.Name)
	if usage.UsedForAppSettings {
		msg += "\n· App settings sync"
	}
	for _, name := range usage.ProfileNames {
		msg += "\n· Host: " + name
	}
	msg += "\n\nForce delete and remove all references?"
	u.showDialog(DialogState{
		Kind:    DialogConfirm,
		Title:   "Destination in use",
		Message: msg,
		OnOK: func() {
			if err := u.core.DeleteRemoteDestination(dest.ID, true); err != nil {
				u.showError(err)
				return
			}
			u.refreshSyncDestinations()
			if u.syncAppSettingsDestID == dest.ID {
				u.syncAppSettingsDestID = ""
			}
			u.syncConnectionOK = false
			u.invalidate()
		},
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

func (u *UI) syncPush() {
	if u.syncAppSettingsDestID == "" {
		u.showError(store.ErrSyncNotConfigured)
		return
	}
	u.showSyncPushWarning()
}

func (u *UI) syncPullThenPush() {
	u.syncPushPending = true
	u.syncPull()
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
	if u.syncAppSettingsDestID == "" {
		u.syncPushPending = false
		u.showError(store.ErrSyncNotConfigured)
		return
	}
	u.showLoading("Sync pull", "Downloading from S3...")
	go func() {
		raw, err := u.core.SyncDownload(context.Background())
		u.closeDialog()
		u.invalidate()
		if err != nil {
			u.syncPushPending = false
			u.syncConnectionOK = false
			u.showError(err)
			return
		}
		u.showDialog(DialogState{
			Kind:    DialogConfirm,
			Title:   "Load remote settings?",
			Message: "Remote app settings were downloaded from S3. Load them into this device?",
			OnOK: func() {
				u.runSyncImport(raw)
			},
			OnCancel: func() {
				u.syncPushPending = false
			},
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
			u.syncPushPending = false
			u.showError(err)
			return
		}
		u.applySyncImport(imported, profileConflicts, templateConflicts)
	}()
}

func (u *UI) applySyncImport(imported store.AppImportData, profileConflicts []store.ProfileConflict, templateConflicts []store.TemplateConflict) {
	apply := func() {
		if err := u.core.ImportAppDataFromBundle(imported); err != nil {
			u.syncPushPending = false
			u.showError(err)
			return
		}
		if err := u.core.RecordSyncPull(); err != nil {
			u.syncPushPending = false
			u.showError(err)
			return
		}
		u.loadSyncFormFromCore()
		u.refreshSyncActivity()
		u.invalidate()
		if u.syncPushPending {
			u.syncPushPending = false
			u.runSyncPush()
			return
		}
		u.showInfo("Pull complete", summarizeAppImport(imported))
	}

	if len(profileConflicts) == 0 && len(templateConflicts) == 0 {
		apply()
		return
	}

	msg := formatAppImportConflicts(profileConflicts, templateConflicts)
	u.showDialog(DialogState{
		Kind:    DialogConfirm,
		Title:   "Sync conflicts",
		Message: msg,
		OnOK:    apply,
		OnCancel: func() {
			u.syncPushPending = false
		},
	})
}
