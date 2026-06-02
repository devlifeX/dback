package ui

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"dback/models"

	"gioui.org/io/clipboard"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

type connectionStepUIStatus int

const (
	stepPending connectionStepUIStatus = iota
	stepRunning
	stepOK
	stepFailed
)

type connectionStepUI struct {
	label  string
	status connectionStepUIStatus
}

type hostConnectionTestState struct {
	running   bool
	server    connectionStepUI
	database  connectionStepUI
	lastError string
	cancel    context.CancelFunc
}

func (u *UI) startHostConnectionTest(p models.Profile) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	u.hostConnTest = hostConnectionTestState{
		running: true,
		server: connectionStepUI{
			label:  serverConnectionStepLabel(p),
			status: stepRunning,
		},
		database: connectionStepUI{
			label:  "Database connection",
			status: stepPending,
		},
		cancel: cancel,
	}
	u.showDialog(DialogState{Kind: DialogConnectionTest})
	u.invalidate()

	go u.runHostConnectionTest(ctx, cancel, p)
}

func (u *UI) runHostConnectionTest(ctx context.Context, cancel context.CancelFunc, p models.Profile) {
	defer cancel()

	err := u.core.TestServerConnection(ctx, p)
	if errors.Is(err, context.Canceled) {
		return
	}
	if err != nil {
		u.hostConnTest.server.status = stepFailed
		u.hostConnTest.lastError = sanitizeError(err)
		u.hostConnTest.running = false
		u.invalidate()
		return
	}

	u.hostConnTest.server.status = stepOK
	u.hostConnTest.database.status = stepRunning
	u.invalidate()

	err = u.core.TestDatabaseConnection(ctx, p)
	if errors.Is(err, context.Canceled) {
		return
	}
	if err != nil {
		u.hostConnTest.database.status = stepFailed
		u.hostConnTest.lastError = sanitizeError(err)
	} else {
		u.hostConnTest.database.status = stepOK
	}
	u.hostConnTest.running = false
	u.invalidate()
}

func serverConnectionStepLabel(p models.Profile) string {
	if p.IsLocalhost() {
		return "Local shell"
	}
	if p.UsesWordPress() {
		return "WordPress REST API"
	}
	return "Server connection"
}

func (u *UI) layoutConnectionTestCard(gtx layout.Context, th *material.Theme, theme *AppTheme) layout.Dimensions {
	st := u.hostConnTest
	return card(gtx, theme, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.H6(th, "Testing Connection")
				lbl.Color = theme.Text
				return lbl.Layout(gtx)
			}),
			layout.Rigid(spacer(theme, unit.Dp(20))),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return u.layoutConnectionTestTimeline(gtx, th, theme, st)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if strings.TrimSpace(st.lastError) == "" {
					return layout.Dimensions{}
				}
				return spacer(theme, unit.Dp(16))(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return u.layoutConnectionTestError(gtx, th, theme, st.lastError)
			}),
			layout.Rigid(spacer(theme, unit.Dp(20))),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					if st.running {
						return secondaryButton(gtx, th, theme, &u.connectionTestCancelBtn, "Cancel", func() {
							if st.cancel != nil {
								st.cancel()
							}
							u.hostConnTest.running = false
							u.closeDialog()
						})
					}
					return secondaryButton(gtx, th, theme, &u.connectionTestCloseBtn, "Close", func() {
						u.closeDialog()
					})
				})
			}),
		)
	})
}

func (u *UI) layoutConnectionTestTimeline(gtx layout.Context, th *material.Theme, theme *AppTheme, st hostConnectionTestState) layout.Dimensions {
	dotCol := gtx.Dp(unit.Dp(24))
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.layoutConnectionTestStepRow(gtx, th, theme, st.server, dotCol)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = dotCol
					gtx.Constraints.Max.X = dotCol
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return timelineConnector(gtx, theme)
					})
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.layoutConnectionTestStepRow(gtx, th, theme, st.database, dotCol)
		}),
	)
}

func (u *UI) layoutConnectionTestStepRow(gtx layout.Context, th *material.Theme, theme *AppTheme, step connectionStepUI, dotCol int) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = dotCol
			gtx.Constraints.Max.X = dotCol
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return statusDot(gtx, theme, connectionStepDotStatus(step.status))
			})
		}),
		layout.Rigid(hgap(theme)),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Body1(th, step.label)
			lbl.Color = theme.Text
			return lbl.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Body2(th, connectionStepStatusText(step.status))
			lbl.Color = theme.TextMuted
			return lbl.Layout(gtx)
		}),
	)
}

func (u *UI) layoutConnectionTestError(gtx layout.Context, th *material.Theme, theme *AppTheme, errText string) layout.Dimensions {
	errText = strings.TrimSpace(errText)
	if errText == "" {
		return layout.Dimensions{}
	}
	displayText := compactErrorText(errText, 3)
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return connectionTestErrorBox(gtx, th, theme, displayText)
		}),
		layout.Rigid(vgap(theme)),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if u.connectionTestCopyBtn.Clicked(gtx) {
						gtx.Execute(clipboard.WriteCmd{
							Type: "application/text",
							Data: io.NopCloser(bytes.NewReader([]byte(errText))),
						})
					}
					return linkButton(gtx, th, theme, &u.connectionTestCopyBtn, "Copy", nil)
				}),
			)
		}),
	)
}

func connectionTestErrorBox(gtx layout.Context, th *material.Theme, theme *AppTheme, text string) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{
		Top: unit.Dp(10), Bottom: unit.Dp(10),
		Left: unit.Dp(12), Right: unit.Dp(12),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Caption(th, text)
		lbl.Color = theme.Danger
		return lbl.Layout(gtx)
	})
	call := macro.Stop()
	borderedRoundedRect(gtx, dims.Size, gtx.Dp(theme.RadiusSm), theme.DangerSoft, theme.Danger, gtx.Dp(unit.Dp(1)))
	call.Add(gtx.Ops)
	return dims
}

func compactErrorText(text string, maxLines int) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) <= maxLines {
		return strings.Join(lines, "\n")
	}
	return strings.Join(lines[:maxLines], "\n") + "…"
}

func connectionStepDotStatus(status connectionStepUIStatus) stepDotStatus {
	switch status {
	case stepRunning:
		return dotRunning
	case stepOK:
		return dotOK
	case stepFailed:
		return dotFailed
	default:
		return dotPending
	}
}

func connectionStepStatusText(status connectionStepUIStatus) string {
	switch status {
	case stepRunning:
		return "Checking…"
	case stepOK:
		return "Connected"
	case stepFailed:
		return "Failed"
	default:
		return "Waiting…"
	}
}
