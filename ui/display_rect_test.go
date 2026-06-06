package ui

import "testing"

func TestParseXrandrPrimary(t *testing.T) {
	out := `Screen 0: minimum 320 x 200, current 3840 x 1080, maximum 16384 x 16384
HDMI-1 connected 1920x1080+1920+0 (normal left inverted right x axis y axis) 510mm x 287mm
DP-1 connected primary 1920x1080+0+0 (normal left inverted right x axis y axis) 510mm x 287mm`
	rect, ok := parseXrandrPrimary(out)
	if !ok {
		t.Fatal("expected primary display")
	}
	if rect.X != 0 || rect.Y != 0 || rect.Width != 1920 || rect.Height != 1080 {
		t.Fatalf("unexpected primary rect: %#v", rect)
	}
}

func TestCenterOnDisplay(t *testing.T) {
	x, y := centerOnDisplay(displayRect{X: 1920, Y: 0, Width: 1920, Height: 1080}, 1200, 800)
	if x != 1920+(1920-1200)/2 || y != 0+(1080-800)/2 {
		t.Fatalf("unexpected center: %d,%d", x, y)
	}
}
