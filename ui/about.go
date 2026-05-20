package ui

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

func (u *UI) layoutAbout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	theme := u.theme
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = gtx.Dp(unit.Dp(480))
		return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return u.layoutLogo(gtx, th, 96)
				}),
				layout.Rigid(vgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.H5(th, "About DBack")
					lbl.Color = theme.Text
					return layout.Center.Layout(gtx, lbl.Layout)
				}),
				layout.Rigid(vgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return mutedLabel(gtx, th, theme, "DB Sync Manager")
					})
				}),
				layout.Rigid(vgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Body1(th, "dariush vesal")
					lbl.Color = theme.Text
					return layout.Center.Layout(gtx, lbl.Layout)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return mutedLabel(gtx, th, theme, "dariush.vesal@gmail.com")
					})
				}),
				layout.Rigid(vgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						lbl := material.Body2(th, "github.com/devlifeX/dback")
						lbl.Color = theme.Accent
						return lbl.Layout(gtx)
					})
				}),
			)
		})
	})
}
