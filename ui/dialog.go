package ui

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

func (u *UI) layoutDialog(gtx layout.Context, th *material.Theme) layout.Dimensions {
	theme := u.theme
	d := u.dialog

	fillRect(gtx, gtx.Constraints.Max, color.NRGBA{R: 0, G: 0, B: 0, A: 0x99})

	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = gtx.Dp(unit.Dp(420))
		return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.H6(th, d.Title)
					lbl.Color = theme.Text
					return lbl.Layout(gtx)
				}),
				layout.Rigid(vgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Body1(th, d.Message)
					lbl.Color = theme.TextMuted
					return lbl.Layout(gtx)
				}),
				layout.Rigid(vgap(theme)),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if d.Kind == DialogLoading {
						return progressBar(gtx, theme, -1)
					}
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if d.Kind != DialogConfirm {
								return layout.Dimensions{}
							}
							return secondaryButton(gtx, th, theme, &u.dialogCancelBtn, "Cancel", func() {
								if d.OnCancel != nil {
									d.OnCancel()
								}
								u.closeDialog()
							})
						}),
						layout.Rigid(hgap(theme)),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							label := "OK"
							if d.Kind == DialogConfirm {
								label = "Confirm"
							}
							return primaryButton(gtx, th, theme, &u.dialogOKBtn, label, func() {
								if d.OnOK != nil {
									d.OnOK()
								}
								u.closeDialog()
							})
						}),
					)
				}),
			)
		})
	})
}

func (u *UI) showConfirm(title, message string, onOK func()) {
	u.showDialog(DialogState{
		Kind:     DialogConfirm,
		Title:    title,
		Message:  message,
		OnOK:     onOK,
		OnCancel: func() {},
	})
}

func (u *UI) showError(err error) {
	if err == nil {
		return
	}
	u.showDialog(DialogState{
		Kind:    DialogError,
		Title:   "Error",
		Message: err.Error(),
	})
}

func (u *UI) showInfo(title, message string) {
	u.showDialog(DialogState{
		Kind:    DialogInfo,
		Title:   title,
		Message: message,
	})
}

func (u *UI) showLoading(title, message string) {
	u.showDialog(DialogState{
		Kind:    DialogLoading,
		Title:   title,
		Message: message,
	})
}
