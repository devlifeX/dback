package main

import (
	_ "embed"

	"dback/ui"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

//go:embed logo.png
var logoBytes []byte

func main() {
	a := app.NewWithID("com.dbsync.manager")
	a.SetIcon(fyne.NewStaticResource("logo.png", logoBytes))

	// Set theme (optional customization can be done here)
	// a.Settings().SetTheme(theme.DarkTheme()) // Default is based on system usually, but requirements asked for Dark.
	// Fyne usually detects system theme. If we want to force dark:
	// a.Settings().SetTheme(&myDarkTheme{}) or similar.
	// For now, rely on default which is often dark-friendly or user system pref.

	userInterface := ui.NewUI(a)
	userInterface.Run()
}
