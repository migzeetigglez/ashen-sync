//go:build nogui

package main

import "fmt"

func runGUI(whatsNewPath string) error {
	_ = whatsNewPath
	return fmt.Errorf("GUI was not included in this build (built with -tags nogui); use CLI commands or a full desktop build")
}

func guiUsageLine() string {
	return "  save-channeler update [--check]\n"
}
