package ui

import (
	"fmt"
	"strings"

	"dback/backend/verify"
	"dback/models"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
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
				if d.Kind == DialogVerifyReport {
					maxW = unit.Dp(520)
				}
				if d.Kind == DialogDeepVerifyConfirm {
					maxW = unit.Dp(560)
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
					if d.Kind != DialogDeepVerifyConfirm {
						return layout.Dimensions{}
					}
					hosts := importableProfiles(u.core.Profiles())
					if len(hosts) == 0 {
						return mutedLabel(gtx, th, theme, "No import destinations available.")
					}
					values, labels := importableHostDropdownOptions(hosts)
					if u.deepVerifySelect.Value == "" {
						u.deepVerifySelect.Value = defaultDeepVerifyHostID(hosts)
					}
					return labeledEnumDropdownField(gtx, th, theme, &u.deepVerifySelect, "Verify on host", values, labels, &u.deepVerifyDropdown, u.invalidate, nil)
				}),
				layout.Rigid(vgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if d.Kind != DialogVerifyReport || len(d.VerifyReport) == 0 {
						return layout.Dimensions{}
					}
					return layoutVerifyReportDetails(gtx, th, theme, &u.dialogHostList, d.VerifyReport, d.VerifyPassed)
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

func layoutVerifyReportDetails(gtx layout.Context, th *material.Theme, theme *AppTheme, list *widget.List, report []models.TableVerifyResult, passed bool) layout.Dimensions {
	_, mismatched, matched := verify.PartitionReport(report)
	maxH := gtx.Dp(unit.Dp(260))
	gtx.Constraints.Max.Y = maxH
	return scrollArea(gtx, th, list, func(gtx layout.Context) layout.Dimensions {
		var rows []layout.FlexChild
		if len(mismatched) > 0 {
			rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return sectionLabel(gtx, th, theme, fmt.Sprintf("Tables with row differences (%d)", len(mismatched)))
			}))
			rows = append(rows, layout.Rigid(vgap(theme)))
			for _, row := range mismatched {
				row := row
				rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layoutVerifyMismatchRow(gtx, th, theme, row)
				}))
				rows = append(rows, layout.Rigid(vgap(theme)))
			}
		}
		if len(matched) > 0 {
			rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return sectionLabel(gtx, th, theme, fmt.Sprintf("Matched tables (%d)", len(matched)))
			}))
			rows = append(rows, layout.Rigid(vgap(theme)))
			rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				names := make([]string, len(matched))
				for i, row := range matched {
					names[i] = row.Table
				}
				return mutedLabel(gtx, th, theme, strings.Join(names, ", "))
			}))
		}
		if passed && len(mismatched) == 0 && len(matched) == 0 {
			rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return mutedLabel(gtx, th, theme, "No table details available.")
			}))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
	})
}

func layoutVerifyMismatchRow(gtx layout.Context, th *material.Theme, theme *AppTheme, row models.TableVerifyResult) layout.Dimensions {
	diff := row.Actual - row.Expected
	diffText := fmt.Sprintf("%+d", diff)
	line1 := row.Table
	line2 := fmt.Sprintf("In restored backup: %s", formatVerifyCount(row.Actual))
	line3 := fmt.Sprintf("Recorded at backup: %s", formatVerifyCount(row.Expected))
	line4 := fmt.Sprintf("Difference: %s rows", diffText)
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Body2(th, line1)
			lbl.Color = theme.Text
			return lbl.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return mutedLabel(gtx, th, theme, line2)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return mutedLabel(gtx, th, theme, line3)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Body2(th, line4)
			lbl.Color = theme.Danger
			return lbl.Layout(gtx)
		}),
	)
}

func formatVerifyCount(n int64) string {
	s := fmt.Sprintf("%d", n)
	if n < 1000 {
		return s
	}
	var parts []string
	for s != "" {
		if len(s) <= 3 {
			parts = append([]string{s}, parts...)
			break
		}
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	return strings.Join(parts, ",")
}

func dialogHasActions(kind DialogKind) bool {
	switch kind {
	case DialogConfirm, DialogPassword, DialogTemplateReplace, DialogInfo, DialogError, DialogUpdateAvailable, DialogVerifyReport, DialogDeepVerifyConfirm:
		return true
	default:
		return false
	}
}

func dialogHasCancel(kind DialogKind) bool {
	switch kind {
	case DialogConfirm, DialogPassword, DialogTemplateReplace, DialogUpdateAvailable, DialogDeepVerifyConfirm:
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
	case DialogConfirm, DialogDeepVerifyConfirm:
		return "Start deep verify"
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
