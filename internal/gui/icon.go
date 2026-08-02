//go:build !nogui

package gui

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed icon.png
var iconPNG []byte

func appIcon() fyne.Resource {
	return fyne.NewStaticResource("save-channeler.png", iconPNG)
}
