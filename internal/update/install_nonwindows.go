//go:build !windows

package update

import "fmt"

func installWindows(path string, progress ProgressFunc) error {
	return fmt.Errorf("windows install is not available on this platform")
}
