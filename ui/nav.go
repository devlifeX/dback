package ui

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

const sidebarLogoSize = 36

func (u *UI) layoutSidebar(gtx layout.Context, th *material.Theme) layout.Dimensions {
	theme := u.theme
	fillRect(gtx, gtx.Constraints.Max, theme.SidebarBg)

	// Right border separator
	borderW := gtx.Dp(unit.Dp(1))
	fillRect(gtx, image.Pt(borderW, gtx.Constraints.Max.Y), theme.Border)

	return layout.Inset{
		Top: unit.Dp(28), Bottom: unit.Dp(28),
		Left: unit.Dp(20), Right: unit.Dp(20),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return u.layoutLogo(gtx, th, sidebarLogoSize)
					}),
					layout.Rigid(spacer(theme, unit.Dp(10))),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Subtitle1(th, "DBack")
						lbl.Color = theme.Text
						return lbl.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Caption(th, "DB Sync Manager")
						lbl.Color = theme.TextMuted
						return lbl.Layout(gtx)
					}),
				)
			}),
			layout.Rigid(spacer(theme, unit.Dp(32))),
			layout.Rigid(u.navItem(th, theme, &u.navHosts, SectionHosts, "Hosts")),
			layout.Rigid(vgap(theme)),
			layout.Rigid(u.navItem(th, theme, &u.navBackups, SectionBackups, "Backups")),
			layout.Rigid(vgap(theme)),
			layout.Rigid(u.navItem(th, theme, &u.navTemplates, SectionTemplates, "Templates")),
			layout.Rigid(vgap(theme)),
			layout.Rigid(u.navItem(th, theme, &u.navSettings, SectionSettings, "Settings")),
			layout.Rigid(vgap(theme)),
			layout.Rigid(u.navItem(th, theme, &u.navAbout, SectionAbout, "About")),
		)
	})
}

func (u *UI) layoutBottomNav(gtx layout.Context, th *material.Theme) layout.Dimensions {
	theme := u.theme
	fillRect(gtx, gtx.Constraints.Max, theme.SidebarBg)
	borderH := gtx.Dp(unit.Dp(1))
	fillRect(gtx, image.Pt(gtx.Constraints.Max.X, borderH), theme.Border)

	return layout.Inset{
		Top: unit.Dp(8), Bottom: unit.Dp(8),
		Left: unit.Dp(8), Right: unit.Dp(8),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Flexed(1, u.navItem(th, theme, &u.navHosts, SectionHosts, "Hosts")),
			layout.Flexed(1, u.navItem(th, theme, &u.navBackups, SectionBackups, "Backups")),
			layout.Flexed(1, u.navItem(th, theme, &u.navTemplates, SectionTemplates, "Templates")),
			layout.Flexed(1, u.navItem(th, theme, &u.navSettings, SectionSettings, "Settings")),
		)
	})
}

func (u *UI) navItem(th *material.Theme, theme *AppTheme, btn *widget.Clickable, section Section, label string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		active := u.section == section
		if btn.Clicked(gtx) {
			u.navigate(section)
		}

		bg := colorTransparent
		fg := theme.TextMuted
		accentBar := colorTransparent
		if active {
			bg = theme.AccentSoft
			fg = theme.Text
			accentBar = theme.Link
		}

		return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			if gtx.Constraints.Max.X > 0 {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
			}

			macro := op.Record(gtx.Ops)
			dims := layout.Inset{
				Top: unit.Dp(11), Bottom: unit.Dp(11),
				Left: unit.Dp(14), Right: unit.Dp(14),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body1(th, label)
				lbl.Color = fg
				return lbl.Layout(gtx)
			})
			call := macro.Stop()

			radius := gtx.Dp(theme.RadiusSm)
			if bg.A > 0 {
				fillRoundedRect(gtx, dims.Size, radius, bg)
			}
			if accentBar.A > 0 {
				barW := gtx.Dp(unit.Dp(3))
				fillRoundedRect(gtx, image.Pt(barW, dims.Size.Y), barW/2, accentBar)
			}
			call.Add(gtx.Ops)
			return dims
		})
	}
}

var colorTransparent = color.NRGBA{A: 0}

func (u *UI) layoutLogo(gtx layout.Context, th *material.Theme, size int) layout.Dimensions {
	if u.logo == nil {
		return layout.Dimensions{}
	}
	sz := gtx.Dp(unit.Dp(size))
	gtx.Constraints = layout.Exact(image.Pt(sz, sz))
	imgOp := paint.NewImageOp(u.logo)
	bounds := u.logo.Bounds()
	scaleX := float32(sz) / float32(bounds.Dx())
	scaleY := float32(sz) / float32(bounds.Dy())
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}
	img := widget.Image{
		Src:   imgOp,
		Fit:   widget.Contain,
		Scale: scale,
	}
	return img.Layout(gtx)
}
