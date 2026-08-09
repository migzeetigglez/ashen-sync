package update

import (
	"fmt"
	"runtime"
)

// AssetNameForGOOS returns the expected GitHub release asset filename.
func AssetNameForGOOS(goos string) (string, error) {
	switch goos {
	case "windows":
		return "save-channeler-windows-amd64.zip", nil
	case "linux":
		return "save-channeler-linux-amd64.tar.gz", nil
	default:
		return "", fmt.Errorf("in-app updates are not supported on %s", goos)
	}
}

// CurrentAssetName is AssetNameForGOOS(runtime.GOOS).
func CurrentAssetName() (string, error) {
	return AssetNameForGOOS(runtime.GOOS)
}

// BinaryName is the app binary inside the package for goos.
func BinaryName(goos string) string {
	if goos == "windows" {
		return "save-channeler.exe"
	}
	return "save-channeler"
}

// RcloneName is the bundled rclone binary name for goos.
func RcloneName(goos string) string {
	if goos == "windows" {
		return "rclone.exe"
	}
	return "rclone"
}
