//go:build linux && !cgo

package ui

import "unsafe"

func x11MoveWindow(display unsafe.Pointer, window uintptr, x, y int) bool {
	return false
}
