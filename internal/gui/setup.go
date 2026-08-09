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

const (
	remoteGoogleDrive = "gdrive"
	remoteProtonDrive = "proton"
)

// disconnectRcloneRemote removes a named rclone remote if present.
func disconnectRcloneRemote(ctx context.Context, rcloneBin, name string, log func(string)) error {
	if rcloneBin == "" {
		rcloneBin = "rclone"
	}
	rc := rclonex.Client{Bin: rcloneBin}
	remotes, err := rc.ListRemotes(ctx)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "config") ||
			strings.Contains(strings.ToLower(err.Error()), "not found") {
			return nil
		}
		return err
	}
	found := false
	for _, r := range remotes {
		if r == name {
			found = true
			break
		}
	}
	if !found {
		log(fmt.Sprintf("No rclone remote %q to remove.", name))
		return nil
	}
	cmd := exec.CommandContext(ctx, rcloneBin, "config", "delete", name)
	out, err := cmd.CombinedOutput()
	msg := strings.TrimSpace(string(out))
	if err != nil {
		if msg != "" {
			return fmt.Errorf("could not remove remote %q: %s", name, msg)
		}
		return fmt.Errorf("could not remove remote %q: %w", name, err)
	}
	if msg != "" {
		log(msg)
	}
	log(fmt.Sprintf("Removed rclone remote %q.", name))
	return nil
}

func disconnectGoogleDrive(ctx context.Context, rcloneBin string, log func(string)) error {
	return disconnectRcloneRemote(ctx, rcloneBin, remoteGoogleDrive, log)
}

func disconnectProtonDrive(ctx context.Context, rcloneBin string, log func(string)) error {
	return disconnectRcloneRemote(ctx, rcloneBin, remoteProtonDrive, log)
}

// disconnectConfiguredRemote clears whichever destination Save Channeler was using.
func disconnectConfiguredRemote(ctx context.Context, rcloneBin string, cfg config.Config, log func(string)) error {
	switch cfg.Remote {
	case remoteGoogleDrive:
		return disconnectGoogleDrive(ctx, rcloneBin, log)
	case remoteProtonDrive:
		return disconnectProtonDrive(ctx, rcloneBin, log)
	case "":
		return nil
	default:
		return disconnectRcloneRemote(ctx, rcloneBin, cfg.Remote, log)
	}
}

func obscurePassword(ctx context.Context, rcloneBin, plain string) (string, error) {
	cmd := exec.CommandContext(ctx, rcloneBin, "obscure", plain)
	out, err := cmd.CombinedOutput()
	msg := strings.TrimSpace(string(out))
	if err != nil {
		if msg != "" {
			return "", fmt.Errorf("rclone obscure failed: %s", msg)
		}
		return "", fmt.Errorf("rclone obscure failed: %w", err)
	}
	if msg == "" {
		return "", fmt.Errorf("rclone obscure returned empty output")
	}
	return msg, nil
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

	_ = disconnectProtonDrive(ctx, rcloneBin, log)
	if err := disconnectGoogleDrive(ctx, rcloneBin, log); err != nil {
		return config.Config{}, err
	}

	log("Opening Google sign-in in your browser — pick the account you want…")
	cmd := exec.CommandContext(ctx, rcloneBin, "config", "create", remoteGoogleDrive, "drive")
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
	if existing, err := config.Load(); err == nil {
		cfg = existing
	}
	cfg.RclonePath = rcloneBin
	cfg.Remote = remoteGoogleDrive
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

// setupProtonDrive links Proton Drive (proton.me) via rclone’s protondrive backend.
// Password is obscured for rclone config. User should have logged into Proton Drive
// in a browser at least once so encryption keys exist.
func setupProtonDrive(ctx context.Context, rcloneBin, username, password, twoFA, folder string, log func(string)) (config.Config, error) {
	if rcloneBin == "" {
		rcloneBin = "rclone"
	}
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	twoFA = strings.TrimSpace(twoFA)
	folder = strings.TrimSpace(folder)
	if folder == "" {
		folder = "save-channeler"
	}
	if username == "" || password == "" {
		return config.Config{}, fmt.Errorf("Proton email and password are required")
	}

	rc := rclonex.Client{Bin: rcloneBin}
	if _, err := rc.Version(ctx); err != nil {
		return config.Config{}, fmt.Errorf("rclone not found — install from https://rclone.org/install/ (%w)", err)
	}

	_ = disconnectGoogleDrive(ctx, rcloneBin, log)
	if err := disconnectProtonDrive(ctx, rcloneBin, log); err != nil {
		return config.Config{}, err
	}

	obscured, err := obscurePassword(ctx, rcloneBin, password)
	if err != nil {
		return config.Config{}, err
	}

	log("Linking Proton Drive (no browser auth — using form credentials)…")
	args := []string{
		"config", "create", remoteProtonDrive, "protondrive",
		"username", username,
		"password", obscured,
	}
	if twoFA != "" {
		args = append(args, "2fa", twoFA)
	}
	cmd := exec.CommandContext(ctx, rcloneBin, args...)
	out, err := cmd.CombinedOutput()
	msg := strings.TrimSpace(string(out))
	if err != nil {
		if msg != "" {
			return config.Config{}, fmt.Errorf("Proton Drive setup failed: %s\n\nTip: log into drive.proton.me in a browser once first (encryption keys), and use a fresh 2FA code if enabled.", msg)
		}
		return config.Config{}, fmt.Errorf("Proton Drive setup failed: %w", err)
	}
	if msg != "" {
		log(msg)
	}
	log("Proton Drive linked as remote “proton”.")

	cfg := config.Default()
	if existing, err := config.Load(); err == nil {
		cfg = existing
	}
	cfg.RclonePath = rcloneBin
	cfg.Remote = remoteProtonDrive
	cfg.RemotePath = folder

	mkdirCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	if err := rc.About(mkdirCtx, cfg.RemoteRoot()); err != nil {
		log("note: " + err.Error())
	}
	log("Creating folder “" + folder + "” in Proton Drive…")
	mkdir := exec.CommandContext(mkdirCtx, rcloneBin, "mkdir", cfg.RemoteRoot())
	if out, err := mkdir.CombinedOutput(); err != nil {
		log(strings.TrimSpace(string(out)))
	} else {
		log("Folder ready at " + cfg.RemoteRoot() + " (refresh drive.proton.me if it doesn’t show yet)")
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

func remoteKindLabel(cfg config.Config) string {
	switch cfg.Remote {
	case remoteGoogleDrive:
		return "Google Drive"
	case remoteProtonDrive:
		return "Proton Drive"
	case "":
		return "none"
	default:
		return cfg.Remote
	}
}
