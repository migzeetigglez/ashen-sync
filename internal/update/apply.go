package update

import (
	"context"
	"fmt"
	"runtime"
)

// CanApplyInPlace reports whether this OS supports replacing the running binary.
func CanApplyInPlace() bool {
	switch runtime.GOOS {
	case "windows", "linux":
		return true
	default:
		return false
	}
}

// Apply downloads the release asset, replaces the install binaries, and relaunches
// the GUI with a What's New notes file. This function does not return on success
// for Unix (exec); on Windows it starts a helper and returns nil so the caller can exit.
func Apply(ctx context.Context, info Info) error {
	if !info.Available {
		return fmt.Errorf("no update available")
	}
	if !CanApplyInPlace() {
		return fmt.Errorf("in-app updates are not supported on %s — open %s", runtime.GOOS, info.HTMLURL)
	}
	if info.AssetURL == "" {
		return fmt.Errorf("release %s has no downloadable asset for this OS", info.Tag)
	}
	stage, err := DownloadAndStage(ctx, info)
	if err != nil {
		return err
	}
	// Staging cleanup is owned by the platform apply helpers after swap.
	return applyPlatform(ctx, info, stage)
}
