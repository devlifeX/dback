package ui

import (
	"fmt"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

func (u *UI) layoutAbout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	theme := u.theme
	gtx.Constraints.Min = gtx.Constraints.Max
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = gtx.Dp(unit.Dp(520))
		return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return u.layoutLogo(gtx, th, 72)
					})
				}),
				layout.Rigid(spacer(theme, unit.Dp(20))),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						lbl := material.H4(th, "About DBack")
						lbl.Color = theme.Text
						return lbl.Layout(gtx)
					})
				}),
				layout.Rigid(vgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return mutedLabel(gtx, th, theme, "DB Sync Manager")
					})
				}),
				layout.Rigid(vgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return mutedLabel(gtx, th, theme, fmt.Sprintf("Version %s", u.version))
					})
				}),
				layout.Rigid(spacer(theme, unit.Dp(24))),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return divider(gtx, theme)
				}),
				layout.Rigid(spacer(theme, unit.Dp(24))),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						lbl := material.Body1(th, "dariush vesal")
						lbl.Color = theme.Text
						return lbl.Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return mutedLabel(gtx, th, theme, "dariush.vesal@gmail.com")
					})
				}),
				layout.Rigid(vgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						status := u.updateStatus
						if status == "" {
							status = "Check GitHub Releases for newer versions."
						}
						return mutedLabel(gtx, th, theme, status)
					})
				}),
				layout.Rigid(vgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return primaryButton(gtx, th, theme, &u.aboutCheckUpdateBtn, "Check for updates", u.runAboutUpdateCheck)
							}),
							layout.Rigid(hgap(theme)),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return linkButton(gtx, th, theme, &u.aboutProjectBtn, "Open GitHub", func() {
									if err := u.platform.OpenURL(ProjectURL); err != nil {
										u.showError(fmt.Errorf("open project link: %w", err))
									}
								})
							}),
						)
					})
				}),
			)
		})
	})
}
