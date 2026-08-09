//go:build !nogui

package gui

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed icon.png
var iconPNG []byte

//go:embed brand-stack.png
var brandStackPNG []byte

func appIcon() fyne.Resource {
	return fyne.NewStaticResource("save-channeler.png", iconPNG)
}

// brandStackResource is the in-app hero mark: SAVE CHANNELER wordmark above the mascot.
func brandStackResource() fyne.Resource {
	return fyne.NewStaticResource("save-channeler-brand-stack.png", brandStackPNG)
}
