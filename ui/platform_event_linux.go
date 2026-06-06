//go:build linux

package ui

import "gioui.org/app"

func (u *UI) handlePlatformEvent(e any) {
	xev, ok := e.(app.X11ViewEvent)
	if !ok || !xev.Valid() {
		return
	}
	u.x11Display = xev.Display
	u.x11Window = xev.Window
	u.attemptWindowCenter()
}
