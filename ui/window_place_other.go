//go:build !linux

package ui

func (u *UI) attemptWindowCenter() bool {
	return false
}
