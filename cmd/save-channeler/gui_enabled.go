//go:build !nogui

package main

import "github.com/migzeetigglez/ashen-sync/internal/gui"

func runGUI(whatsNewPath string) error {
	return gui.RunWith(gui.Options{WhatsNewPath: whatsNewPath})
}

func guiUsageLine() string {
	return "  save-channeler gui [--whats-new <notes.json>]\n  save-channeler update [--check]\n"
}
