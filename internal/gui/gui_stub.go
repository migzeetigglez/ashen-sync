//go:build nogui

package gui

import "fmt"

// Options is a stub for nogui builds.
type Options struct {
	WhatsNewPath string
}

// Run is unavailable in nogui builds (used for cross-compiling CLI from macOS).
func Run() error {
	return RunWith(Options{})
}

// RunWith is unavailable in nogui builds.
func RunWith(Options) error {
	return fmt.Errorf("GUI disabled in this build")
}
