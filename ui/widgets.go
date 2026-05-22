package ui

import (
	"image"
	"image/color"

	"gioui.org/io/key"
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
	btnSuccess
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

func strokeRoundedRect(gtx layout.Context, size image.Point, radius int, col color.NRGBA, width int) {
	if width <= 0 {
		return
	}
	path := clip.UniformRRect(image.Rectangle{Max: size}, radius).Path(gtx.Ops)
	defer clip.Stroke{Path: path, Width: float32(width)}.Op().Push(gtx.Ops).Pop()
	paint.ColorOp{Color: col}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}

func borderedRoundedRect(gtx layout.Context, size image.Point, radius int, fill, border color.NRGBA, borderWidth int) {
	fillRoundedRect(gtx, size, radius, fill)
	if borderWidth > 0 {
		strokeRoundedRect(gtx, size, radius, border, borderWidth)
	}
}

func sectionTitle(gtx layout.Context, th *material.Theme, theme *AppTheme, title string) layout.Dimensions {
	lbl := material.H4(th, title)
	lbl.Color = theme.Text
	return lbl.Layout(gtx)
}

func sectionLabel(gtx layout.Context, th *material.Theme, theme *AppTheme, text string) layout.Dimensions {
	lbl := material.Subtitle2(th, text)
	lbl.Color = theme.TextMuted
	return lbl.Layout(gtx)
}

func mutedLabel(gtx layout.Context, th *material.Theme, theme *AppTheme, text string) layout.Dimensions {
	lbl := material.Body2(th, text)
	lbl.Color = theme.TextMuted
	return lbl.Layout(gtx)
}

func pageHeader(gtx layout.Context, th *material.Theme, theme *AppTheme, title string, action layout.Widget) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return sectionTitle(gtx, th, theme, title)
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if action == nil {
				return layout.Dimensions{}
			}
			return action(gtx)
		}),
	)
}

func card(gtx layout.Context, theme *AppTheme, w layout.Widget) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{
		Top: theme.CardPadding, Bottom: theme.CardPadding,
		Left: theme.CardPadding, Right: theme.CardPadding,
	}.Layout(gtx, w)
	call := macro.Stop()

	radius := gtx.Dp(theme.Radius)
	borderedRoundedRect(gtx, dims.Size, radius, theme.Surface, theme.Border, gtx.Dp(unit.Dp(1)))
	call.Add(gtx.Ops)
	return dims
}

func borderedInput(gtx layout.Context, theme *AppTheme, focused bool, w layout.Widget) layout.Dimensions {
	borderCol := theme.Border
	if focused {
		borderCol = theme.Link
	}
	radius := gtx.Dp(theme.RadiusSm)
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
		Top: unit.Dp(11), Bottom: unit.Dp(11),
		Left: unit.Dp(14), Right: unit.Dp(14),
	}.Layout(inputGtx, w)
	call := macro.Stop()

	minHeight := gtx.Dp(unit.Dp(44))
	if dims.Size.Y < minHeight {
		dims.Size.Y = minHeight
	}
	borderedRoundedRect(gtx, dims.Size, radius, theme.InputBg, borderCol, border)
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
			lbl := material.Body2(th, label)
			lbl.Color = theme.Text
			return lbl.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(6)}.Layout(gtx, w)
		}),
	)
}

func requestEditorFocus(gtx layout.Context, e *widget.Editor, pending *bool) {
	if pending == nil || !*pending {
		return
	}
	gtx.Execute(key.FocusCmd{Tag: e})
	*pending = false
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
	var bg, fg, border color.NRGBA
	borderWidth := 0
	radius := gtx.Dp(theme.RadiusSm)

	switch style {
	case btnPrimary:
		bg = theme.Accent
		fg = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	case btnDanger:
		bg = theme.Danger
		fg = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	case btnSuccess:
		bg = theme.Success
		fg = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	case btnTab:
		if active {
			bg = theme.SurfaceAlt
			fg = theme.Text
			border = theme.Border
			borderWidth = gtx.Dp(unit.Dp(1))
		} else {
			bg = theme.Bg
			fg = theme.TextMuted
			borderWidth = 0
		}
	default:
		bg = theme.SurfaceAlt
		fg = theme.Text
		border = theme.Border
		borderWidth = gtx.Dp(unit.Dp(1))
	}

	return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		macro := op.Record(gtx.Ops)
		dims := layout.Inset{
			Top: unit.Dp(10), Bottom: unit.Dp(10),
			Left: unit.Dp(16), Right: unit.Dp(16),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Body2(th, label)
			lbl.Color = fg
			return lbl.Layout(gtx)
		})
		call := macro.Stop()

		borderedRoundedRect(gtx, dims.Size, radius, bg, border, borderWidth)
		call.Add(gtx.Ops)
		return dims
	})
}

func primaryButton(gtx layout.Context, th *material.Theme, theme *AppTheme, btn *widget.Clickable, label string, onClick func()) layout.Dimensions {
	return actionButton(gtx, th, theme, btn, label, btnPrimary, false, onClick)
}

func secondaryButton(gtx layout.Context, th *material.Theme, theme *AppTheme, btn *widget.Clickable, label string, onClick func()) layout.Dimensions {
	return actionButton(gtx, th, theme, btn, label, btnSecondary, false, onClick)
}

func successButton(gtx layout.Context, th *material.Theme, theme *AppTheme, btn *widget.Clickable, label string, onClick func()) layout.Dimensions {
	return actionButton(gtx, th, theme, btn, label, btnSuccess, false, onClick)
}

func disabledButton(gtx layout.Context, th *material.Theme, theme *AppTheme, label string) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{
		Top: unit.Dp(10), Bottom: unit.Dp(10),
		Left: unit.Dp(16), Right: unit.Dp(16),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Body2(th, label)
		lbl.Color = theme.TextMuted
		return lbl.Layout(gtx)
	})
	call := macro.Stop()
	borderedRoundedRect(gtx, dims.Size, gtx.Dp(theme.RadiusSm), theme.SurfaceAlt, theme.Border, gtx.Dp(unit.Dp(1)))
	call.Add(gtx.Ops)
	return dims
}

func dangerButton(gtx layout.Context, th *material.Theme, theme *AppTheme, btn *widget.Clickable, label string, onClick func()) layout.Dimensions {
	return actionButton(gtx, th, theme, btn, label, btnDanger, false, onClick)
}

func tabButton(gtx layout.Context, th *material.Theme, theme *AppTheme, btn *widget.Clickable, label string, active bool, onClick func()) layout.Dimensions {
	return actionButton(gtx, th, theme, btn, label, btnTab, active, onClick)
}

func linkButton(gtx layout.Context, th *material.Theme, theme *AppTheme, btn *widget.Clickable, label string, onClick func()) layout.Dimensions {
	if btn.Clicked(gtx) && onClick != nil {
		onClick()
	}
	return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Body2(th, label)
		lbl.Color = theme.Link
		return lbl.Layout(gtx)
	})
}

func tabBar(gtx layout.Context, th *material.Theme, theme *AppTheme, tabs ...layout.Widget) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		func() []layout.FlexChild {
			var children []layout.FlexChild
			for i, tab := range tabs {
				if i > 0 {
					children = append(children, layout.Rigid(hgap(theme)))
				}
				tab := tab
				children = append(children, layout.Rigid(tab))
			}
			return children
		}()...,
	)
}

func badge(gtx layout.Context, th *material.Theme, theme *AppTheme, label string) layout.Dimensions {
	bg := theme.SurfaceAlt
	fg := theme.TextMuted
	border := theme.Border
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{
		Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(10), Right: unit.Dp(10),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Caption(th, label)
		lbl.Color = fg
		return lbl.Layout(gtx)
	})
	call := macro.Stop()
	borderPx := gtx.Dp(unit.Dp(1))
	borderedRoundedRect(gtx, dims.Size, gtx.Dp(theme.RadiusSm), bg, border, borderPx)
	call.Add(gtx.Ops)
	return dims
}

func importProtectedIcon(gtx layout.Context, theme *AppTheme) layout.Dimensions {
	return lockIcon(gtx, theme.Success)
}

func lockIcon(gtx layout.Context, col color.NRGBA) layout.Dimensions {
	sz := gtx.Dp(unit.Dp(18))
	gtx.Constraints = layout.Exact(image.Pt(sz, sz))

	bodyW := sz * 11 / 18
	bodyH := sz * 8 / 18
	bodyX := (sz - bodyW) / 2
	bodyY := sz - bodyH - 1

	stack := op.Offset(image.Pt(bodyX, bodyY)).Push(gtx.Ops)
	fillRoundedRect(gtx, image.Pt(bodyW, bodyH), gtx.Dp(unit.Dp(2)), col)
	stack.Pop()

	shW := bodyW
	shH := sz * 7 / 18
	shX := bodyX
	shY := bodyY - shH/2
	stack = op.Offset(image.Pt(shX, shY)).Push(gtx.Ops)
	strokeRoundedRect(gtx, image.Pt(shW, shH), shW/2, col, gtx.Dp(unit.Dp(2)))
	stack.Pop()

	return layout.Dimensions{Size: image.Pt(sz, sz)}
}

func searchField(gtx layout.Context, th *material.Theme, theme *AppTheme, e *widget.Editor, hint string) layout.Dimensions {
	e.SingleLine = true
	e.Submit = true
	return editorField(gtx, th, theme, e, hint)
}

func chipButton(gtx layout.Context, th *material.Theme, theme *AppTheme, btn *widget.Clickable, label, sublabel string, active bool, onClick func()) layout.Dimensions {
	bg := theme.Surface
	fg := theme.TextMuted
	border := theme.Border
	if active {
		bg = theme.AccentSoft
		fg = theme.Link
		border = theme.Link
	}
	clicked := btn.Clicked(gtx)
	return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		if clicked && onClick != nil {
			onClick()
		}
		macro := op.Record(gtx.Ops)
		dims := layout.Inset{
			Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(14), Right: unit.Dp(14),
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
		borderedRoundedRect(gtx, dims.Size, gtx.Dp(theme.RadiusSm), bg, border, borderPx)
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
			return layout.Inset{Top: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
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

func passwordField(gtx layout.Context, th *material.Theme, theme *AppTheme, e *widget.Editor, hint string, visible *bool, toggle *widget.Clickable) layout.Dimensions {
	e.SingleLine = true
	e.Submit = true
	if visible == nil || !*visible {
		e.Mask = '*'
	} else {
		e.Mask = 0
	}
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return editorField(gtx, th, theme, e, hint)
		}),
		layout.Rigid(hgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if toggle == nil {
				return layout.Dimensions{}
			}
			label := "Show"
			if visible != nil && *visible {
				label = "Hide"
			}
			return secondaryButton(gtx, th, theme, toggle, label, func() {
				if visible != nil {
					*visible = !*visible
				}
			})
		}),
	)
}

func consumeEditorSubmit(gtx layout.Context, e *widget.Editor, onSubmit func()) {
	if ev, ok := e.Update(gtx); ok {
		if _, ok := ev.(widget.SubmitEvent); ok && onSubmit != nil {
			onSubmit()
		}
	}
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
			return layout.Inset{Top: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
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
						children = append(children, layout.Rigid(vgap(theme)))
					}
					return layout.Flex{Axis: layout.Vertical, Spacing: layout.SpaceStart}.Layout(gtx, children...)
				})
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

func divider(gtx layout.Context, theme *AppTheme) layout.Dimensions {
	h := gtx.Dp(unit.Dp(1))
	w := gtx.Constraints.Max.X
	fillRect(gtx, image.Pt(w, h), theme.Border)
	return layout.Dimensions{Size: image.Pt(w, h)}
}

func progressBar(gtx layout.Context, theme *AppTheme, progress float64) layout.Dimensions {
	if progress < 0 {
		progress = 0.5
	}
	if progress > 1 {
		progress = 1
	}
	w := gtx.Constraints.Max.X
	h := gtx.Dp(unit.Dp(8))
	radius := h / 2
	fillRoundedRect(gtx, image.Pt(w, h), radius, theme.SurfaceAlt)
	fillW := int(float64(w) * progress)
	if fillW > 0 {
		fillRoundedRect(gtx, image.Pt(fillW, h), radius, theme.Link)
	}
	return layout.Dimensions{Size: image.Pt(w, h)}
}

func emptyState(gtx layout.Context, th *material.Theme, theme *AppTheme, message string) layout.Dimensions {
	return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Top: unit.Dp(24), Bottom: unit.Dp(24),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return mutedLabel(gtx, th, theme, message)
			})
		})
	})
}
