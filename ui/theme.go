package ui

import (
	"image/color"

	"gioui.org/unit"
	"gioui.org/widget/material"
)

type AppTheme struct {
	*material.Theme
	Bg          color.NRGBA
	Surface     color.NRGBA
	SurfaceAlt  color.NRGBA
	SidebarBg   color.NRGBA
	Accent      color.NRGBA
	AccentDim   color.NRGBA
	AccentSoft  color.NRGBA
	Link        color.NRGBA
	Border      color.NRGBA
	InputBg     color.NRGBA
	Text        color.NRGBA
	TextMuted   color.NRGBA
	Danger      color.NRGBA
	DangerSoft  color.NRGBA
	Success     color.NRGBA
	Overlay     color.NRGBA
	Radius      unit.Dp
	RadiusSm    unit.Dp
	Padding     unit.Dp
	CardPadding unit.Dp
	Gap         unit.Dp
	SectionGap  unit.Dp
}

func NewAppTheme() *AppTheme {
	th := material.NewTheme()
	return &AppTheme{
		Theme: th,
		// GitHub dark (primer)
		Bg:          color.NRGBA{R: 0x0d, G: 0x11, B: 0x17, A: 0xff}, // canvas-default
		Surface:     color.NRGBA{R: 0x16, G: 0x1b, B: 0x22, A: 0xff}, // canvas-subtle
		SurfaceAlt:  color.NRGBA{R: 0x21, G: 0x26, B: 0x2d, A: 0xff}, // neutral-muted
		SidebarBg:   color.NRGBA{R: 0x01, G: 0x04, B: 0x09, A: 0xff}, // canvas-inset
		Accent:      color.NRGBA{R: 0x23, G: 0x86, B: 0x36, A: 0xff}, // btn-primary
		AccentDim:   color.NRGBA{R: 0x2e, G: 0xa0, B: 0x43, A: 0xff}, // btn-primary hover
		AccentSoft:  color.NRGBA{R: 0x21, G: 0x26, B: 0x2d, A: 0xff}, // selected/highlight
		Link:        color.NRGBA{R: 0x44, G: 0x93, B: 0xf8, A: 0xff}, // accent-fg
		Border:      color.NRGBA{R: 0x30, G: 0x36, B: 0x3d, A: 0xff}, // border-default
		InputBg:     color.NRGBA{R: 0x0d, G: 0x11, B: 0x17, A: 0xff},
		Text:        color.NRGBA{R: 0xe6, G: 0xed, B: 0xf3, A: 0xff}, // fg-default
		TextMuted:   color.NRGBA{R: 0x8b, G: 0x94, B: 0x9e, A: 0xff}, // fg-muted
		Danger:      color.NRGBA{R: 0xda, G: 0x36, B: 0x33, A: 0xff}, // danger-emphasis
		DangerSoft:  color.NRGBA{R: 0x3d, G: 0x12, B: 0x14, A: 0xff},
		Success:     color.NRGBA{R: 0x3f, G: 0xb9, B: 0x50, A: 0xff},
		Overlay:     color.NRGBA{R: 0x01, G: 0x04, B: 0x09, A: 0xb3},
		Radius:      unit.Dp(12),
		RadiusSm:    unit.Dp(8),
		Padding:     unit.Dp(24),
		CardPadding: unit.Dp(20),
		Gap:         unit.Dp(16),
		SectionGap:  unit.Dp(28),
	}
}

func (t *AppTheme) WithPalette() *material.Theme {
	t.Theme.Palette.Fg = t.Text
	t.Theme.Palette.Bg = t.Bg
	t.Theme.Palette.ContrastBg = t.Accent
	t.Theme.Palette.ContrastFg = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	return t.Theme
}
