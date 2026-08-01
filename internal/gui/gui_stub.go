//go:build nogui

package gui

import "fmt"

// Run is unavailable in nogui builds (used for cross-compiling CLI from macOS).
func Run() error {
	return fmt.Errorf("GUI disabled in this build")
}
