//go:build !nogui

package gui

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/migzeetigglez/ashen-sync/internal/config"
	"github.com/migzeetigglez/ashen-sync/internal/rclonex"
)

const defaultRemoteName = "gdrive"

// disconnectGoogleDrive removes the Save Channeler rclone remote so the next
// link can sign in with a different Google account.
func disconnectGoogleDrive(ctx context.Context, rcloneBin string, log func(string)) error {
	if rcloneBin == "" {
		rcloneBin = "rclone"
	}
	rc := rclonex.Client{Bin: rcloneBin}
	remotes, err := rc.ListRemotes(ctx)
	if err != nil {
		// No config yet — already disconnected.
		if strings.Contains(strings.ToLower(err.Error()), "config") ||
			strings.Contains(strings.ToLower(err.Error()), "not found") {
			return nil
		}
		return err
	}
	found := false
	for _, r := range remotes {
		if r == defaultRemoteName {
			found = true
			break
		}
	}
	if !found {
		log("No Google Drive remote to disconnect.")
		return nil
	}
	cmd := exec.CommandContext(ctx, rcloneBin, "config", "delete", defaultRemoteName)
	out, err := cmd.CombinedOutput()
	msg := strings.TrimSpace(string(out))
	if err != nil {
		if msg != "" {
			return fmt.Errorf("could not disconnect Google Drive: %s", msg)
		}
		return fmt.Errorf("could not disconnect Google Drive: %w", err)
	}
	if msg != "" {
		log(msg)
	}
	log("Disconnected Google account (rclone remote “gdrive” removed).")
	return nil
}

// setupGoogleDrive always starts a fresh Google OAuth (removing any existing
// gdrive remote first), creates the destination folder, and writes config.
func setupGoogleDrive(ctx context.Context, rcloneBin, folder string, log func(string)) (config.Config, error) {
	if rcloneBin == "" {
		rcloneBin = "rclone"
	}
	folder = strings.TrimSpace(folder)
	if folder == "" {
		folder = "save-channeler"
	}
	rc := rclonex.Client{Bin: rcloneBin}
	if _, err := rc.Version(ctx); err != nil {
		return config.Config{}, fmt.Errorf("rclone not found — install from https://rclone.org/install/ (%w)", err)
	}

	// Drop any previous Save Channeler Google link so account chooser appears.
	if err := disconnectGoogleDrive(ctx, rcloneBin, log); err != nil {
		return config.Config{}, err
	}

	log("Opening Google sign-in in your browser — pick the account you want…")
	cmd := exec.CommandContext(ctx, rcloneBin, "config", "create", defaultRemoteName, "drive")
	out, err := cmd.CombinedOutput()
	msg := strings.TrimSpace(string(out))
	if err != nil {
		if msg != "" {
			return config.Config{}, fmt.Errorf("rclone setup failed: %s", msg)
		}
		return config.Config{}, fmt.Errorf("rclone setup failed: %w", err)
	}
	if msg != "" {
		log(msg)
	}
	log("Google Drive linked as remote “gdrive”.")

	cfg := config.Default()
	cfg.RclonePath = rcloneBin
	cfg.Remote = defaultRemoteName
	cfg.RemotePath = folder

	mkdirCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	if err := rc.About(mkdirCtx, cfg.RemoteRoot()); err != nil {
		log("note: " + err.Error())
	}
	mkdir := exec.CommandContext(mkdirCtx, rcloneBin, "mkdir", cfg.RemoteRoot())
	if out, err := mkdir.CombinedOutput(); err != nil {
		log(strings.TrimSpace(string(out)))
	}

	if err := cfg.Save(); err != nil {
		return config.Config{}, err
	}
	log("Saved config → " + mustConfigPath())
	return cfg, nil
}

func mustConfigPath() string {
	p, err := config.Path()
	if err != nil {
		return "(config path unknown)"
	}
	return p
}

func remoteConfigured(cfg config.Config) bool {
	return cfg.Remote != ""
}
