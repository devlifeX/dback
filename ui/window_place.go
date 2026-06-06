package ui

import (
	"image"
	"runtime"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/unit"
)

const (
	windowWidth        = unit.Dp(1200)
	windowHeight       = unit.Dp(800)
	windowMinW         = unit.Dp(900)
	windowMinH         = unit.Dp(600)
	maxWindowCenterTry = 30
)

func (u *UI) configureWindow() {
	u.window.Option(
		app.Title("DBack - DB Sync Manager"),
		app.Size(windowWidth, windowHeight),
		app.MinSize(windowMinW, windowMinH),
	)
}

func (u *UI) centerWindowForSize(size image.Point) {
	if !validWindowSize(size) {
		return
	}
	u.pendingCenterSize = size
	if u.attemptWindowCenter() {
		return
	}
	// Gio ActionCenter uses the full X11 virtual desktop on multi-monitor setups.
	if runtime.GOOS != "linux" && !u.windowCentered {
		u.window.Perform(system.ActionCenter)
		u.windowCentered = true
	}
}
