//go:build !nogui

package gui

// Options configures GUI startup.
type Options struct {
	// WhatsNewPath is a JSON notes file from a completed self-update.
	WhatsNewPath string
}
