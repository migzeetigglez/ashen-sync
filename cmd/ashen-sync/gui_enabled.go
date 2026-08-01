//go:build !nogui

package main

import "github.com/michaelteague/ashen-sync/internal/gui"

func runGUI() error {
	return gui.Run()
}

func guiUsageLine() string {
	return "  ashen-sync gui\n"
}
