//go:build nogui

package main

import "fmt"

func runGUI() error {
	return fmt.Errorf("GUI was not included in this build (built with -tags nogui); use CLI commands or a full desktop build")
}

func guiUsageLine() string {
	return ""
}
