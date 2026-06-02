package ui

import (
	"fmt"

	"dback/models"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

func (u *UI) layoutDialog(gtx layout.Context, th *material.Theme) layout.Dimensions {
	gtx.Constraints.Min = gtx.Constraints.Max
	theme := u.theme
	d := u.dialog

	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			fillRect(gtx, gtx.Constraints.Max, theme.Overlay)
			return layout.Dimensions{Size: gtx.Constraints.Max}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min = gtx.Constraints.Max
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				maxW := unit.Dp(440)
				if d.Kind == DialogTemplateReplace {
					maxW = unit.Dp(520)
				}
				if d.Kind == DialogSyncPushWarning {
					maxW = unit.Dp(480)
				}
				if d.Kind == DialogConnectionTest {
					maxW = unit.Dp(420)
				}
				gtx.Constraints.Max.X = gtx.Dp(maxW)
				if d.Kind == DialogConnectionTest {
					return u.layoutConnectionTestCard(gtx, th, theme)
				}
				return u.layoutDialogCard(gtx, th, theme, d)
			})
		}),
	)
}

func (u *UI) layoutDialogCard(gtx layout.Context, th *material.Theme, theme *AppTheme, d DialogState) layout.Dimensions {
	return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.H6(th, d.Title)
					lbl.Color = theme.Text
					return lbl.Layout(gtx)
				}),
				layout.Rigid(vgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Body1(th, d.Message)
					lbl.Color = theme.TextMuted
					return lbl.Layout(gtx)
				}),
				layout.Rigid(vgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if d.Kind != DialogTemplateReplace || len(d.HostUsages) == 0 {
						return layout.Dimensions{}
					}
					maxH := gtx.Dp(unit.Dp(180))
					gtx.Constraints.Max.Y = maxH
					return scrollArea(gtx, th, &u.dialogHostList, func(gtx layout.Context) layout.Dimensions {
						var rows []layout.FlexChild
						for _, usage := range d.HostUsages {
							usage := usage
							rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								line := fmt.Sprintf("• %s (%s)", usage.ProfileName, usage.LocationLabel())
								return mutedLabel(gtx, th, theme, line)
							}))
							rows = append(rows, layout.Rigid(vgap(theme)))
						}
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
					})
				}),
				layout.Rigid(vgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if d.Kind != DialogPassword {
						return layout.Dimensions{}
					}
					return labeledField(gtx, th, theme, "Master password", func(gtx layout.Context) layout.Dimensions {
						consumeEditorSubmit(gtx, &u.passphraseEditor, func() {
							if d.OnOK != nil {
								d.OnOK()
							}
							u.passphraseEditor.SetText("")
							u.closeDialog()
						})
						return passwordField(gtx, th, theme, &u.passphraseEditor, "", &u.passphraseVisible, &u.passphraseToggle)
					})
				}),
				layout.Rigid(vgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if d.Kind == DialogLoading {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return progressBar(gtx, theme, -1)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if d.OnCancel == nil {
									return layout.Dimensions{}
								}
								return vgap(theme)(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if d.OnCancel == nil {
									return layout.Dimensions{}
								}
								return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return secondaryButton(gtx, th, theme, &u.dialogCancelBtn, "Cancel", func() {
										if d.OnCancel != nil {
											d.OnCancel()
										}
										u.closeDialog()
									})
								})
							}),
						)
					}
					if d.Kind == DialogSyncPushWarning {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return wideSuccessButton(gtx, th, theme, &u.dialogSyncPullBtn, "Pull first, then Push", func() {
									u.closeDialog()
									u.syncPullThenPush()
								})
							}),
							layout.Rigid(vgap(theme)),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return wideDangerButton(gtx, th, theme, &u.dialogForcePushBtn, "Force Push", func() {
									u.syncPushPending = false
									u.closeDialog()
									u.runSyncPush()
								})
							}),
							layout.Rigid(vgap(theme)),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return secondaryButton(gtx, th, theme, &u.dialogCancelBtn, "Cancel", func() {
										u.syncPushPending = false
										u.closeDialog()
									})
								})
							}),
						)
					}
					if !dialogHasActions(d.Kind) {
						return layout.Dimensions{}
					}
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if !dialogHasCancel(d.Kind) {
								return layout.Dimensions{}
							}
							return secondaryButton(gtx, th, theme, &u.dialogCancelBtn, "Cancel", func() {
								if d.OnCancel != nil {
									d.OnCancel()
								}
								u.passphraseEditor.SetText("")
								u.closeDialog()
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if !dialogHasCancel(d.Kind) {
								return layout.Dimensions{}
							}
							return hgap(theme)(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							label := dialogOKLabel(d.Kind, d.OKLabel)
						return primaryButton(gtx, th, theme, &u.dialogOKBtn, label, func() {
							if d.OnOK != nil {
								d.OnOK()
							}
							if d.Kind == DialogUpdateAvailable {
								return
							}
							u.passphraseEditor.SetText("")
							u.closeDialog()
						})
						}),
					)
				}),
			)
		})
}

func (u *UI) showSyncPushWarning() {
	u.showDialog(DialogState{
		Kind:    DialogSyncPushWarning,
		Title:   "Overwrite remote data?",
		Message: "You are about to upload local settings to S3 and replace the remote copy at dback/app-data.json. Pull the latest remote data first to avoid losing changes from other devices.",
	})
}

func (u *UI) showConfirm(title, message string, onOK func()) {
	u.showDialog(DialogState{
		Kind:     DialogConfirm,
		Title:    title,
		Message:  message,
		OnOK:     onOK,
		OnCancel: func() {},
	})
}

func (u *UI) showError(err error) {
	if err == nil {
		return
	}
	u.showDialog(DialogState{
		Kind:    DialogError,
		Title:   "Error",
		Message: sanitizeError(err),
	})
}

func (u *UI) showInfo(title, message string) {
	u.showDialog(DialogState{
		Kind:    DialogInfo,
		Title:   title,
		Message: message,
	})
}

func (u *UI) showLoading(title, message string) {
	u.showDialog(DialogState{
		Kind:    DialogLoading,
		Title:   title,
		Message: message,
	})
}

func (u *UI) showLoadingWithCancel(title, message string, onCancel func()) {
	u.showDialog(DialogState{
		Kind:     DialogLoading,
		Title:    title,
		Message:  message,
		OnCancel: onCancel,
	})
}

func (u *UI) showPasswordPrompt(title, message string, onOK func(passphrase string)) {
	u.passphraseEditor.SetText("")
	u.showDialog(DialogState{
		Kind:    DialogPassword,
		Title:   title,
		Message: message,
		OnOK: func() {
			onOK(editorText(&u.passphraseEditor))
		},
		OnCancel: func() {},
	})
}

func (u *UI) showTemplateReplacePrompt(t models.SQLTemplate, oldBody string, usages []models.TemplateHostUsage, onSave func()) {
	u.showDialog(DialogState{
		Kind:       DialogTemplateReplace,
		Title:      "Update hosts using this template?",
		Message:    fmt.Sprintf("The template SQL changed. %d host(s) still contain the previous version:", len(usages)),
		HostUsages: usages,
		OnCancel:   onSave,
		OnOK: func() {
			ids := make(map[string]struct{}, len(usages))
			for _, usage := range usages {
				ids[usage.ProfileID] = struct{}{}
			}
			if err := u.core.ReplaceTemplateInProfiles(oldBody, t.Body, ids); err != nil {
				u.showError(err)
				return
			}
			onSave()
		},
	})
}

func dialogHasActions(kind DialogKind) bool {
	switch kind {
	case DialogConfirm, DialogPassword, DialogTemplateReplace, DialogInfo, DialogError, DialogUpdateAvailable:
		return true
	default:
		return false
	}
}

func dialogHasCancel(kind DialogKind) bool {
	switch kind {
	case DialogConfirm, DialogPassword, DialogTemplateReplace, DialogUpdateAvailable:
		return true
	default:
		return false
	}
}

func dialogOKLabel(kind DialogKind, custom string) string {
	if custom != "" {
		return custom
	}
	switch kind {
	case DialogConfirm:
		return "Confirm"
	case DialogPassword:
		return "Continue"
	case DialogTemplateReplace:
		return "Replace"
	case DialogUpdateAvailable:
		return "Update now"
	case DialogInfo, DialogError:
		return "OK"
	default:
		return "OK"
	}
}
