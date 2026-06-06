//go:build linux

package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var xrandrGeometryRE = regexp.MustCompile(`(\d+)x(\d+)\+(-?\d+)\+(-?\d+)`)

func (u *UI) attemptWindowCenter() bool {
	if u.windowCentered || !validWindowSize(u.pendingCenterSize) {
		return u.windowCentered
	}
	rect, ok := primaryDisplayRectLinux()
	if !ok {
		u.bumpWindowCenterTries()
		return false
	}
	x, y := centerOnDisplay(rect, u.pendingCenterSize.X, u.pendingCenterSize.Y)
	moved := false
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		moved = moveWindowWayland(os.Getpid(), x, y)
	} else if u.x11Display != nil && u.x11Window != 0 {
		moved = x11MoveWindow(u.x11Display, u.x11Window, x, y)
	}
	if !moved {
		moved = moveWindowX11(os.Getpid(), x, y)
	}
	if moved {
		u.windowCentered = true
		return true
	}
	u.bumpWindowCenterTries()
	return false
}

func (u *UI) bumpWindowCenterTries() {
	u.windowCenterAttempts++
	if u.windowCenterAttempts >= maxWindowCenterTry {
		u.windowCentered = true
	}
}

func primaryDisplayRectLinux() (displayRect, bool) {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		if rect, ok := primaryDisplayRectSway(); ok {
			return rect, true
		}
		if rect, ok := primaryDisplayRectHyprland(); ok {
			return rect, true
		}
		return displayRect{}, false
	}
	return primaryDisplayRectX11()
}

func primaryDisplayRectX11() (displayRect, bool) {
	out, err := exec.Command("xrandr", "--current").Output()
	if err != nil {
		return displayRect{}, false
	}
	return parseXrandrPrimary(string(out))
}

func parseXrandrPrimary(output string) (displayRect, bool) {
	var fallback displayRect
	hasFallback := false
	for _, line := range strings.Split(output, "\n") {
		if !strings.Contains(line, " connected") {
			continue
		}
		rect, ok := parseXrandrLine(line)
		if !ok {
			continue
		}
		if strings.Contains(line, " primary ") {
			return rect, true
		}
		if !hasFallback {
			fallback = rect
			hasFallback = true
		}
	}
	return fallback, hasFallback
}

func parseXrandrLine(line string) (displayRect, bool) {
	m := xrandrGeometryRE.FindStringSubmatch(line)
	if len(m) != 5 {
		return displayRect{}, false
	}
	w, err1 := strconv.Atoi(m[1])
	h, err2 := strconv.Atoi(m[2])
	x, err3 := strconv.Atoi(m[3])
	y, err4 := strconv.Atoi(m[4])
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		return displayRect{}, false
	}
	return displayRect{X: x, Y: y, Width: w, Height: h}, true
}

type swayOutput struct {
	Primary bool `json:"primary"`
	Focused bool `json:"focused"`
	Rect    struct {
		X      int `json:"x"`
		Y      int `json:"y"`
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"rect"`
}

func primaryDisplayRectSway() (displayRect, bool) {
	out, err := exec.Command("swaymsg", "-t", "get_outputs").Output()
	if err != nil {
		return displayRect{}, false
	}
	var outputs []swayOutput
	if err := json.Unmarshal(out, &outputs); err != nil {
		return displayRect{}, false
	}
	var focused, primary, first *swayOutput
	for i := range outputs {
		o := &outputs[i]
		if o.Rect.Width == 0 || o.Rect.Height == 0 {
			continue
		}
		if first == nil {
			first = o
		}
		if o.Focused {
			focused = o
		}
		if o.Primary {
			primary = o
		}
	}
	chosen := primary
	if chosen == nil {
		chosen = focused
	}
	if chosen == nil {
		chosen = first
	}
	if chosen == nil {
		return displayRect{}, false
	}
	return displayRect{
		X:      chosen.Rect.X,
		Y:      chosen.Rect.Y,
		Width:  chosen.Rect.Width,
		Height: chosen.Rect.Height,
	}, true
}

type hyprMonitor struct {
	X        int  `json:"x"`
	Y        int  `json:"y"`
	Width    int  `json:"width"`
	Height   int  `json:"height"`
	Focused  bool `json:"focused"`
	Disabled bool `json:"disabled"`
}

func primaryDisplayRectHyprland() (displayRect, bool) {
	out, err := exec.Command("hyprctl", "monitors", "-j").Output()
	if err != nil {
		return displayRect{}, false
	}
	var monitors []hyprMonitor
	if err := json.Unmarshal(out, &monitors); err != nil {
		return displayRect{}, false
	}
	var focused, first *hyprMonitor
	for i := range monitors {
		m := &monitors[i]
		if m.Disabled || m.Width == 0 || m.Height == 0 {
			continue
		}
		if first == nil {
			first = m
		}
		if m.Focused {
			focused = m
		}
	}
	chosen := focused
	if chosen == nil {
		chosen = first
	}
	if chosen == nil {
		return displayRect{}, false
	}
	return displayRect{
		X:      chosen.X,
		Y:      chosen.Y,
		Width:  chosen.Width,
		Height: chosen.Height,
	}, true
}

func moveWindowX11(pid, x, y int) bool {
	wid, ok := xdotoolWindowID(pid)
	if !ok {
		return false
	}
	if err := exec.Command("xdotool", "windowmove", wid, strconv.Itoa(x), strconv.Itoa(y)).Run(); err != nil {
		return false
	}
	return true
}

func moveWindowWayland(pid, x, y int) bool {
	xs := strconv.Itoa(x)
	ys := strconv.Itoa(y)
	if err := exec.Command("swaymsg", fmt.Sprintf("[pid=%d] move absolute position %s %s", pid, xs, ys)).Run(); err == nil {
		return true
	}
	addr, ok := hyprlandClientAddress(pid)
	if !ok {
		return false
	}
	cmd := fmt.Sprintf("dispatch movewindowpixel exact %s %s,address:%s", xs, ys, addr)
	return exec.Command("hyprctl", cmd).Run() == nil
}

func xdotoolWindowID(pid int) (string, bool) {
	out, err := exec.Command("xdotool", "search", "--pid", strconv.Itoa(pid)).Output()
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line, true
		}
	}
	return "", false
}

type hyprClient struct {
	PID     int    `json:"pid"`
	Address string `json:"address"`
}

func hyprlandClientAddress(pid int) (string, bool) {
	out, err := exec.Command("hyprctl", "clients", "-j").Output()
	if err != nil {
		return "", false
	}
	var clients []hyprClient
	if err := json.Unmarshal(out, &clients); err != nil {
		return "", false
	}
	for _, c := range clients {
		if c.PID == pid && c.Address != "" {
			return c.Address, true
		}
	}
	return "", false
}
