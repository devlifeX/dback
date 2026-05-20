package ui

import (
	"gioui.org/layout"
	"gioui.org/widget/material"
)

func (u *UI) layoutSettings(gtx layout.Context, th *material.Theme) layout.Dimensions {
	theme := u.theme

	return scrollArea(gtx, th, nil, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return sectionTitle(gtx, th, theme, "Settings")
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Subtitle1(th, "Profile Transfer")
							lbl.Color = theme.Text
							return lbl.Layout(gtx)
						}),
						layout.Rigid(vgap(theme)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return mutedLabel(gtx, th, theme, "Export or import host profiles. Encrypted export requires a master password.")
						}),
						layout.Rigid(vgap(theme)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return checkboxField(gtx, th, theme, &u.includeSecrets, "Include saved passwords/API keys (encrypted)")
						}),
						layout.Rigid(vgap(theme)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return secondaryButton(gtx, th, theme, &u.exportProfilesBtn, "Export Profiles", func() {
										u.pickSaveFile("dback-profiles.json", func(path string) {
											include := u.includeSecrets.Value
											if include {
												u.showPasswordPrompt("Export profiles", "Enter master password for encrypted export", func(pass string) {
													if pass == "" {
														u.showError(errPassphraseRequired)
														return
													}
													if err := u.core.ExportProfiles(path, true, pass); err != nil {
														u.showError(err)
														return
													}
													u.showInfo("Export complete", path)
												})
												return
											}
											if err := u.core.ExportProfiles(path, false, ""); err != nil {
												u.showError(err)
												return
											}
											u.showInfo("Export complete", path)
										})
									})
								}),
								layout.Rigid(hgap(theme)),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return secondaryButton(gtx, th, theme, &u.importProfilesBtn, "Import Profiles", func() {
										u.pickOpenFile(func(path string, _ []byte) {
											include := u.includeSecrets.Value
											if include {
												u.showPasswordPrompt("Import profiles", "Enter master password to decrypt", func(pass string) {
													if pass == "" {
														u.showError(errPassphraseRequired)
														return
													}
													u.importProfilesFromPath(path, true, pass)
												})
												return
											}
											u.importProfilesFromPath(path, false, "")
										})
									})
								}),
							)
						}),
					)
				})
			}),
		)
	})
}

var errPassphraseRequired = errString("master password is required")

type errString string

func (e errString) Error() string { return string(e) }
