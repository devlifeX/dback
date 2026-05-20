package ui

import (
	"image"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func (u *UI) layoutSidebar(gtx layout.Context, th *material.Theme) layout.Dimensions {
	theme := u.theme
	fillRect(gtx, gtx.Constraints.Max, theme.Surface)

	return layout.Inset{
		Top: unit.Dp(20), Bottom: unit.Dp(20),
		Left: unit.Dp(16), Right: unit.Dp(16),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return u.layoutLogo(gtx, th, 32)
			}),
			layout.Rigid(vgap(theme)),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.H6(th, "DBack")
				lbl.Color = theme.Text
				return lbl.Layout(gtx)
			}),
			layout.Rigid(spacer(theme, unit.Dp(24))),
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
	fillRect(gtx, gtx.Constraints.Max, theme.Surface)
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Flexed(1, u.navItem(th, theme, &u.navHosts, SectionHosts, "Hosts")),
		layout.Flexed(1, u.navItem(th, theme, &u.navBackups, SectionBackups, "Backups")),
		layout.Flexed(1, u.navItem(th, theme, &u.navTemplates, SectionTemplates, "Templates")),
		layout.Flexed(1, u.navItem(th, theme, &u.navSettings, SectionSettings, "Settings")),
	)
}

func (u *UI) navItem(th *material.Theme, theme *AppTheme, btn *widget.Clickable, section Section, label string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		active := u.section == section
		if btn.Clicked(gtx) {
			u.navigate(section)
		}

		bg := theme.SurfaceAlt
		fg := theme.TextMuted
		border := theme.Border
		if active {
			bg = theme.AccentDim
			fg = theme.Text
			border = theme.Accent
		}

		return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			if gtx.Constraints.Max.X > 0 {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
			}

			macro := op.Record(gtx.Ops)
			dims := layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body1(th, label)
				lbl.Color = fg
				return lbl.Layout(gtx)
			})
			call := macro.Stop()

			borderPx := gtx.Dp(unit.Dp(1))
			fillRoundedRect(gtx, dims.Size, gtx.Dp(theme.Radius), border)
			inner := image.Rectangle{
				Min: image.Pt(borderPx, borderPx),
				Max: image.Pt(dims.Size.X-borderPx, dims.Size.Y-borderPx),
			}
			if inner.Dx() > 0 && inner.Dy() > 0 {
				stack := clip.UniformRRect(inner, gtx.Dp(theme.Radius)-borderPx).Push(gtx.Ops)
				paint.ColorOp{Color: bg}.Add(gtx.Ops)
				paint.PaintOp{}.Add(gtx.Ops)
				stack.Pop()
			}
			call.Add(gtx.Ops)
			return dims
		})
	}
}

func (u *UI) layoutLogo(gtx layout.Context, th *material.Theme, size int) layout.Dimensions {
	if u.logo == nil {
		return layout.Dimensions{}
	}
	imgOp := paint.NewImageOp(u.logo)
	sz := gtx.Dp(unit.Dp(size))
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
	return layout.Center.Layout(gtx, img.Layout)
}
