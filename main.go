package main

import (
	_ "embed"
	"log"
	"os"

	"dback/internal/debug"
	"dback/ui"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

//go:embed logo.png
var logoBytes []byte

func main() {
	args := os.Args[1:]
	if debugEnabledFromArgs(args) || debug.EnabledFromEnv() {
		debug.Enable()
		log.SetOutput(os.Stderr)
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
		if os.Getenv("FYNE_LOG") == "" {
			_ = os.Setenv("FYNE_LOG", "info")
		}
		log.Println("dback debug mode enabled (stderr activity logging)")
	}
	os.Args = append([]string{os.Args[0]}, stripDebugFlag(args)...)

	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic: %v\n%s", r, debug.Stack())
			panic(r)
		}
	}()

	a := app.NewWithID("com.dbsync.manager")
	logo := fyne.NewStaticResource("logo.png", logoBytes)
	a.SetIcon(logo)

	userInterface := ui.NewUI(a, logo)
	userInterface.Run()
}

func debugEnabledFromArgs(args []string) bool {
	for _, arg := range args {
		if arg == "--debug" || arg == "-debug" {
			return true
		}
	}
	return false
}

func stripDebugFlag(args []string) []string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--debug" || arg == "-debug" {
			continue
		}
		out = append(out, arg)
	}
	return out
}
