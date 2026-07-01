package ui

import (
	"strings"

	"gioui.org/layout"
	"gioui.org/widget/material"
)

func (u *UI) layoutSettings(gtx layout.Context, th *material.Theme) layout.Dimensions {
	theme := u.theme

	return scrollArea(gtx, th, &u.settingsList, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return pageHeader(gtx, th, theme, "Settings", nil)
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return tabBar(gtx, th, theme,
					func(gtx layout.Context) layout.Dimensions {
						return tabButton(gtx, th, theme, &u.tabSettingsExport, "Export JSON", u.settingsTab == 0, func() {
							u.settingsTab = 0
							u.invalidate()
						})
					},
					func(gtx layout.Context) layout.Dimensions {
						return tabButton(gtx, th, theme, &u.tabSettingsSync, "Sync", u.settingsTab == 1, func() {
							u.settingsTab = 1
							u.loadSyncFormFromCore()
							u.invalidate()
						})
					},
				)
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if u.settingsTab == 1 {
					return u.layoutSettingsSync(gtx, th, theme)
				}
				return u.layoutSettingsExport(gtx, th, theme)
			}),
		)
	})
}

func (u *UI) layoutSettingsExport(gtx layout.Context, th *material.Theme, theme *AppTheme) layout.Dimensions {
	return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Subtitle1(th, "Export JSON")
				lbl.Color = theme.Text
				return lbl.Layout(gtx)
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return mutedLabel(gtx, th, theme, "Export or import hosts, templates, backup history, and activity logs as JSON. Backup archive files are not included. Exports are encrypted by default; history and log metadata may still contain sensitive host names and paths.")
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return secondaryButton(gtx, th, theme, &u.exportAppDataBtn, "Export App Data", func() {
							u.pickSaveFile("dback-app-data.json", func(path string) {
								u.showPasswordPrompt("Export app data", "Enter an export password to encrypt hosts, templates, history, and logs", func(pass string) {
									if pass == "" {
										u.showError(errPassphraseRequired)
										return
									}
									if err := u.core.ExportAppData(path, true, pass); err != nil {
										u.showError(err)
										return
									}
									u.showInfo("Export complete", path)
								})
							})
						})
					}),
					layout.Rigid(hgap(theme)),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return secondaryButton(gtx, th, theme, &u.importAppDataBtn, "Import App Data", func() {
							u.pickOpenFile(func(path string, _ []byte) {
								u.showPasswordPrompt("Import app data", "Enter export password if the bundle is encrypted (leave blank for legacy plain bundles without secrets)", func(pass string) {
									includeSecrets := strings.TrimSpace(pass) != ""
									u.importAppDataFromPath(path, includeSecrets, pass)
								})
							})
						})
					}),
				)
			}),
		)
	})
}

var errPassphraseRequired = errString("export password is required")

type errString string

func (e errString) Error() string { return string(e) }
