package main

import (
	_ "embed"
	"log"
	"os"

	"dback/internal/debug"
	"dback/ui"
)

//go:embed logo.png
var logoBytes []byte

// appVersion is set at build time via -ldflags; defaults to "3.6.10" for local runs.
var appVersion = "3.6.10"

func main() {
	args := os.Args[1:]
	if debugEnabledFromArgs(args) || debug.EnabledFromEnv() {
		debug.Enable()
		log.SetOutput(os.Stderr)
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
		log.Println("dback debug mode enabled (stderr activity logging)")
	}
	os.Args = append([]string{os.Args[0]}, stripDebugFlag(args)...)

	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic: %v\n%s", r, debug.Stack())
			panic(r)
		}
	}()

	ui.New(logoBytes, appVersion).Run()
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
