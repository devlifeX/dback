package ui

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type buttonStyle int

const (
	btnPrimary buttonStyle = iota
	btnSecondary
	btnDanger
	btnTab
)

func fillRect(gtx layout.Context, size image.Point, col color.NRGBA) {
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: col}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}

func fillRoundedRect(gtx layout.Context, size image.Point, radius int, col color.NRGBA) {
	defer clip.UniformRRect(image.Rectangle{Max: size}, radius).Push(gtx.Ops).Pop()
	paint.ColorOp{Color: col}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}

func sectionTitle(gtx layout.Context, th *material.Theme, theme *AppTheme, title string) layout.Dimensions {
	lbl := material.H5(th, title)
	lbl.Color = theme.Text
	return lbl.Layout(gtx)
}

func mutedLabel(gtx layout.Context, th *material.Theme, theme *AppTheme, text string) layout.Dimensions {
	lbl := material.Body2(th, text)
	lbl.Color = theme.TextMuted
	return lbl.Layout(gtx)
}

func card(gtx layout.Context, theme *AppTheme, w layout.Widget) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{
		Top: theme.Padding, Bottom: theme.Padding,
		Left: theme.Padding, Right: theme.Padding,
	}.Layout(gtx, w)
	call := macro.Stop()

	fillRoundedRect(gtx, dims.Size, gtx.Dp(theme.Radius), theme.Surface)
	call.Add(gtx.Ops)
	return dims
}

func borderedInput(gtx layout.Context, theme *AppTheme, focused bool, w layout.Widget) layout.Dimensions {
	borderCol := theme.Border
	if focused {
		borderCol = theme.Accent
	}
	radius := gtx.Dp(theme.Radius)
	border := gtx.Dp(unit.Dp(1))
	innerRadius := radius - border
	if innerRadius < 0 {
		innerRadius = 0
	}

	inputGtx := gtx
	if inputGtx.Constraints.Max.X > 0 {
		inputGtx.Constraints.Min.X = inputGtx.Constraints.Max.X
	}

	macro := op.Record(gtx.Ops)
	dims := layout.Inset{
		Top: unit.Dp(9), Bottom: unit.Dp(9),
		Left: unit.Dp(13), Right: unit.Dp(13),
	}.Layout(inputGtx, w)
	call := macro.Stop()

	minHeight := gtx.Dp(unit.Dp(42))
	if dims.Size.Y < minHeight {
		dims.Size.Y = minHeight
	}
	fillRoundedRect(gtx, dims.Size, radius, borderCol)
	inner := image.Rectangle{
		Min: image.Pt(border, border),
		Max: image.Pt(dims.Size.X-border, dims.Size.Y-border),
	}
	if inner.Dx() > 0 && inner.Dy() > 0 {
		stack := clip.UniformRRect(inner, innerRadius).Push(gtx.Ops)
		paint.ColorOp{Color: theme.InputBg}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		stack.Pop()
	}
	call.Add(gtx.Ops)
	return dims
}

func vgap(theme *AppTheme) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: image.Pt(0, gtx.Dp(theme.Gap))}
	}
}

func hgap(theme *AppTheme) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: image.Pt(gtx.Dp(theme.Gap), 0)}
	}
}

func labeledField(gtx layout.Context, th *material.Theme, theme *AppTheme, label string, w layout.Widget) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return mutedLabel(gtx, th, theme, label)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(4)}.Layout(gtx, w)
		}),
	)
}

func scrollArea(gtx layout.Context, th *material.Theme, list *widget.List, content layout.Widget) layout.Dimensions {
	if list == nil {
		return content(gtx)
	}
	if list.Axis == 0 {
		list.Axis = layout.Vertical
	}
	return material.List(th, list).Layout(gtx, 1, func(gtx layout.Context, index int) layout.Dimensions {
		if index > 0 {
			return layout.Dimensions{}
		}
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return content(gtx)
	})
}

func actionButton(gtx layout.Context, th *material.Theme, theme *AppTheme, btn *widget.Clickable, label string, style buttonStyle, active bool, onClick func()) layout.Dimensions {
	if btn.Clicked(gtx) && onClick != nil {
		onClick()
	}
	return renderButton(gtx, th, theme, btn, label, style, active)
}

func renderButton(gtx layout.Context, th *material.Theme, theme *AppTheme, btn *widget.Clickable, label string, style buttonStyle, active bool) layout.Dimensions {
	b := material.Button(th, btn, label)
	switch style {
	case btnPrimary:
		b.Background = theme.Accent
		b.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	case btnDanger:
		b.Background = theme.Danger
		b.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	case btnTab:
		if active {
			b.Background = theme.Accent
			b.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
		} else {
			b.Background = theme.SurfaceAlt
			b.Color = theme.TextMuted
		}
	default:
		b.Background = theme.SurfaceAlt
		b.Color = theme.Text
	}
	return b.Layout(gtx)
}

func primaryButton(gtx layout.Context, th *material.Theme, theme *AppTheme, btn *widget.Clickable, label string, onClick func()) layout.Dimensions {
	return actionButton(gtx, th, theme, btn, label, btnPrimary, false, onClick)
}

func secondaryButton(gtx layout.Context, th *material.Theme, theme *AppTheme, btn *widget.Clickable, label string, onClick func()) layout.Dimensions {
	return actionButton(gtx, th, theme, btn, label, btnSecondary, false, onClick)
}

func dangerButton(gtx layout.Context, th *material.Theme, theme *AppTheme, btn *widget.Clickable, label string, onClick func()) layout.Dimensions {
	return actionButton(gtx, th, theme, btn, label, btnDanger, false, onClick)
}

func tabButton(gtx layout.Context, th *material.Theme, theme *AppTheme, btn *widget.Clickable, label string, active bool, onClick func()) layout.Dimensions {
	return actionButton(gtx, th, theme, btn, label, btnTab, active, onClick)
}

func searchField(gtx layout.Context, th *material.Theme, theme *AppTheme, e *widget.Editor, hint string) layout.Dimensions {
	e.SingleLine = true
	e.Submit = true
	return editorField(gtx, th, theme, e, hint)
}

func chipButton(gtx layout.Context, th *material.Theme, theme *AppTheme, btn *widget.Clickable, label, sublabel string, active bool, onClick func()) layout.Dimensions {
	bg := theme.SurfaceAlt
	fg := theme.TextMuted
	border := theme.Border
	if active {
		bg = theme.AccentDim
		fg = theme.Text
		border = theme.Accent
	}
	clicked := btn.Clicked(gtx)
	return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		if clicked && onClick != nil {
			onClick()
		}
		macro := op.Record(gtx.Ops)
		dims := layout.Inset{
			Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(10), Right: unit.Dp(10),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Body2(th, label)
					lbl.Color = fg
					return lbl.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if sublabel == "" {
						return layout.Dimensions{}
					}
					return mutedLabel(gtx, th, theme, sublabel)
				}),
			)
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

// DropdownOptions holds precomputed dropdown labels to avoid per-frame allocations.
type DropdownOptions struct {
	Values []string
	Labels []string
}

func dropdownField(gtx layout.Context, th *material.Theme, theme *AppTheme, label string, opts DropdownOptions, selected *string, open *bool, toggle *widget.Clickable, list *widget.List, itemBtns map[string]*widget.Clickable) layout.Dimensions {
	values := opts.Values
	labels := opts.Labels
	if len(values) == 0 {
		return labeledField(gtx, th, theme, label, func(gtx layout.Context) layout.Dimensions {
			return mutedLabel(gtx, th, theme, "(none)")
		})
	}
	if *selected == "" || !containsString(values, *selected) {
		*selected = values[0]
	}
	display := *selected
	for i, v := range values {
		if v == *selected {
			if i < len(labels) {
				display = labels[i]
			} else {
				display = v
			}
			break
		}
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return mutedLabel(gtx, th, theme, label)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				clicked := toggle.Clicked(gtx)
				if clicked {
					*open = !*open
				}
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return borderedInput(gtx, theme, *open, func(gtx layout.Context) layout.Dimensions {
							return toggle.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
									layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
										lbl := material.Body1(th, display)
										lbl.Color = theme.Text
										return lbl.Layout(gtx)
									}),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										arrow := "▼"
										if *open {
											arrow = "▲"
										}
										return mutedLabel(gtx, th, theme, arrow)
									}),
								)
							})
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if !*open {
							return layout.Dimensions{}
						}
						maxH := gtx.Dp(unit.Dp(180))
						gtx.Constraints.Max.Y = maxH
						gtx.Constraints.Min.Y = maxH
						return borderedInput(gtx, theme, false, func(gtx layout.Context) layout.Dimensions {
							if list.Axis == 0 {
								list.Axis = layout.Vertical
							}
							return material.List(th, list).Layout(gtx, len(values), func(gtx layout.Context, index int) layout.Dimensions {
								value := values[index]
								itemLabel := value
								if index < len(labels) {
									itemLabel = labels[index]
								}
								btn, ok := itemBtns[value]
								if !ok {
									btn = new(widget.Clickable)
									itemBtns[value] = btn
								}
								active := value == *selected
								return chipButton(gtx, th, theme, btn, itemLabel, "", active, func() {
									*selected = value
									*open = false
								})
							})
						})
					}),
				)
			})
		}),
	)
}

func containsString(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

func editorField(gtx layout.Context, th *material.Theme, theme *AppTheme, e *widget.Editor, hint string) layout.Dimensions {
	focused := gtx.Focused(e)
	return borderedInput(gtx, theme, focused, func(gtx layout.Context) layout.Dimensions {
		ed := material.Editor(th, e, hint)
		ed.Color = theme.Text
		ed.HintColor = theme.TextMuted
		return ed.Layout(gtx)
	})
}

func passwordField(gtx layout.Context, th *material.Theme, theme *AppTheme, e *widget.Editor, hint string) layout.Dimensions {
	e.SingleLine = true
	e.Submit = true
	e.Mask = '*'
	return editorField(gtx, th, theme, e, hint)
}

func editorMultiline(gtx layout.Context, th *material.Theme, theme *AppTheme, e *widget.Editor, hint string) layout.Dimensions {
	e.SingleLine = false
	e.Submit = false
	return editorField(gtx, th, theme, e, hint)
}

func checkboxField(gtx layout.Context, th *material.Theme, theme *AppTheme, c *widget.Bool, label string) layout.Dimensions {
	ch := material.CheckBox(th, c, label)
	ch.Color = theme.Text
	return ch.Layout(gtx)
}

func labeledEnumField(gtx layout.Context, th *material.Theme, theme *AppTheme, e *widget.Enum, label string, values, labels []string) layout.Dimensions {
	e.Update(gtx)
	if e.Value == "" && len(values) > 0 {
		e.Value = values[0]
	}
	labelByValue := map[string]string{}
	for i, v := range values {
		text := v
		if i < len(labels) {
			text = labels[i]
		}
		labelByValue[v] = text
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return mutedLabel(gtx, th, theme, label)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return borderedInput(gtx, theme, false, func(gtx layout.Context) layout.Dimensions {
				var children []layout.FlexChild
				for _, v := range values {
					v := v
					display := labelByValue[v]
					children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						rb := material.RadioButton(th, e, v, display)
						rb.Color = theme.Text
						return rb.Layout(gtx)
					}))
					children = append(children, layout.Rigid(hgap(theme)))
				}
				return layout.Flex{Axis: layout.Vertical, Spacing: layout.SpaceStart}.Layout(gtx, children...)
			})
		}),
	)
}

func enumField(gtx layout.Context, th *material.Theme, theme *AppTheme, e *widget.Enum, label string, values []string) layout.Dimensions {
	labels := make([]string, len(values))
	copy(labels, values)
	return labeledEnumField(gtx, th, theme, e, label, values, labels)
}

func spacer(theme *AppTheme, height unit.Dp) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: image.Pt(0, gtx.Dp(height))}
	}
}

func progressBar(gtx layout.Context, theme *AppTheme, progress float64) layout.Dimensions {
	if progress < 0 {
		progress = 0.5
	}
	if progress > 1 {
		progress = 1
	}
	w := gtx.Constraints.Max.X
	h := gtx.Dp(unit.Dp(6))
	fillRect(gtx, image.Pt(w, h), theme.SurfaceAlt)
	fillW := int(float64(w) * progress)
	if fillW > 0 {
		fillRect(gtx, image.Pt(fillW, h), theme.Accent)
	}
	return layout.Dimensions{Size: image.Pt(w, h)}
}
