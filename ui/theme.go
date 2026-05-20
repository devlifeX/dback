package ui

import (
	"image/color"

	"gioui.org/unit"
	"gioui.org/widget/material"
)

type AppTheme struct {
	*material.Theme
	Bg        color.NRGBA
	Surface   color.NRGBA
	SurfaceAlt color.NRGBA
	Accent    color.NRGBA
	AccentDim color.NRGBA
	Border    color.NRGBA
	InputBg   color.NRGBA
	Text      color.NRGBA
	TextMuted color.NRGBA
	Danger    color.NRGBA
	Success   color.NRGBA
	Radius    unit.Dp
	Padding   unit.Dp
	Gap       unit.Dp
}

func NewAppTheme() *AppTheme {
	th := material.NewTheme()
	return &AppTheme{
		Theme: th,
		Bg:        color.NRGBA{R: 0x0f, G: 0x12, B: 0x18, A: 0xff},
		Surface:   color.NRGBA{R: 0x17, G: 0x1c, B: 0x26, A: 0xff},
		SurfaceAlt: color.NRGBA{R: 0x1e, G: 0x25, B: 0x32, A: 0xff},
		Accent:    color.NRGBA{R: 0x3b, G: 0x82, B: 0xf6, A: 0xff},
		AccentDim: color.NRGBA{R: 0x25, G: 0x63, B: 0xeb, A: 0xff},
		Border:    color.NRGBA{R: 0x3d, G: 0x4a, B: 0x5f, A: 0xff},
		InputBg:   color.NRGBA{R: 0x12, G: 0x17, B: 0x22, A: 0xff},
		Text:      color.NRGBA{R: 0xf1, G: 0xf5, B: 0xf9, A: 0xff},
		TextMuted: color.NRGBA{R: 0x94, G: 0xa3, B: 0xb8, A: 0xff},
		Danger:    color.NRGBA{R: 0xef, G: 0x44, B: 0x44, A: 0xff},
		Success:   color.NRGBA{R: 0x22, G: 0xc5, B: 0x5e, A: 0xff},
		Radius:    unit.Dp(10),
		Padding:   unit.Dp(16),
		Gap:       unit.Dp(12),
	}
}

func (t *AppTheme) WithPalette() *material.Theme {
	t.Theme.Palette.Fg = t.Text
	t.Theme.Palette.Bg = t.Bg
	t.Theme.Palette.ContrastBg = t.Accent
	t.Theme.Palette.ContrastFg = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	return t.Theme
}
