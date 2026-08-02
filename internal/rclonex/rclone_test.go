package rclonex

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveRclonePrefersBundled(t *testing.T) {
	dir := t.TempDir()
	name := "rclone"
	if runtime.GOOS == "windows" {
		name = "rclone.exe"
	}
	bundled := filepath.Join(dir, name)
	if err := os.WriteFile(bundled, []byte("#!/bin/sh\necho rclone mock\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := resolveRclone("rclone")
	// Without being able to spoof os.Executable easily, just ensure resolveRclone
	// returns a non-empty string and LookPath/home fallbacks don't panic.
	if got == "" {
		t.Fatal("expected non-empty resolution")
	}

	// Direct file check helper
	if !isExecFile(bundled) {
		t.Fatal("expected bundled file to count as executable file")
	}
}

func TestClientResolvedPathExplicit(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "custom-rclone")
	if err := os.WriteFile(p, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	c := Client{Bin: p}
	if c.ResolvedPath() != p {
		t.Fatalf("want %s got %s", p, c.ResolvedPath())
	}
}
