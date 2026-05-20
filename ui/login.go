package ui

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

func (u *UI) layoutLogin(gtx layout.Context, th *material.Theme) layout.Dimensions {
	theme := u.theme
	title := "Unlock DBack"
	message := "Enter your master key to access hosts, backups, and settings."
	buttonLabel := "Unlock"
	if !u.core.HasVault() {
		title = "Create Master Key"
		buttonLabel = "Create Vault"
		if u.core.HasLegacyPlaintext() {
			message = "Set a master key to encrypt your existing data. Legacy plaintext files will be migrated into the encrypted vault."
			buttonLabel = "Encrypt & Unlock"
		} else {
			message = "Choose a master key to encrypt all application data. You will need it every time you open DBack."
		}
	}

	submitLogin := func() {
		u.tryUnlock(editorText(&u.loginPassword))
	}

	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = gtx.Dp(unit.Dp(440))
		return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return u.layoutLogo(gtx, th, 56)
					})
				}),
				layout.Rigid(spacer(theme, unit.Dp(20))),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						lbl := material.H5(th, title)
						lbl.Color = theme.Text
						return lbl.Layout(gtx)
					})
				}),
				layout.Rigid(vgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return mutedLabel(gtx, th, theme, message)
					})
				}),
				layout.Rigid(spacer(theme, unit.Dp(24))),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					consumeEditorSubmit(gtx, &u.loginPassword, submitLogin)
					return labeledField(gtx, th, theme, "Master key", func(gtx layout.Context) layout.Dimensions {
						return passwordField(gtx, th, theme, &u.loginPassword, "", &u.loginPasswordVisible, &u.loginPasswordToggle)
					})
				}),
				layout.Rigid(vgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if u.core.HasVault() {
						return layout.Dimensions{}
					}
					consumeEditorSubmit(gtx, &u.loginConfirmPassword, submitLogin)
					return labeledField(gtx, th, theme, "Confirm master key", func(gtx layout.Context) layout.Dimensions {
						return passwordField(gtx, th, theme, &u.loginConfirmPassword, "", &u.loginConfirmVisible, &u.loginConfirmToggle)
					})
				}),
				layout.Rigid(vgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if u.loginError == "" {
						return layout.Dimensions{}
					}
					return mutedLabel(gtx, th, theme, u.loginError)
				}),
				layout.Rigid(spacer(theme, unit.Dp(8))),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return primaryButton(gtx, th, theme, &u.loginBtn, buttonLabel, submitLogin)
				}),
			)
		})
	})
}
