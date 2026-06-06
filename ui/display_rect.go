package ui

import "image"

type displayRect struct {
	X, Y, Width, Height int
}

func centerOnDisplay(rect displayRect, winW, winH int) (x, y int) {
	if winW > rect.Width {
		x = rect.X
	} else {
		x = rect.X + (rect.Width-winW)/2
	}
	if winH > rect.Height {
		y = rect.Y
	} else {
		y = rect.Y + (rect.Height-winH)/2
	}
	return x, y
}

func validWindowSize(size image.Point) bool {
	return size.X > 0 && size.Y > 0
}
