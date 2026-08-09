package update

import (
	"fmt"
	"os"
	"path/filepath"
)

// InstallDir returns the directory containing the running executable
// (after resolving symlinks).
func InstallDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(exe)
	if dir == "" || dir == "." {
		return "", fmt.Errorf("could not determine install directory")
	}
	return dir, nil
}

// ExecutablePath returns the resolved path of the running binary.
func ExecutablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}
