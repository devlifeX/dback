//go:build linux && cgo

package ui

/*
#cgo pkg-config: x11
#include <X11/Xlib.h>
*/
import "C"
import "unsafe"

func x11MoveWindow(display unsafe.Pointer, window uintptr, x, y int) bool {
	if display == nil || window == 0 {
		return false
	}
	dpy := (*C.Display)(display)
	win := C.Window(window)
	C.XMoveWindow(dpy, win, C.int(x), C.int(y))
	C.XFlush(dpy)
	return true
}
