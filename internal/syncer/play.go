package syncer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/migzeetigglez/ashen-sync/internal/detect"
	"github.com/migzeetigglez/ashen-sync/internal/processx"
)

// Play pulls (if possible), launches the Steam game, waits for exit, then pushes.
func (e Engine) Play(ctx context.Context, gameID string, prefer Prefer) error {
	g, ok := detect.ByID(gameID)
	if !ok {
		return fmt.Errorf("unknown game: %s", gameID)
	}

	e.logf("pulling before launch...")
	if err := e.Pull(ctx, gameID, prefer); err != nil {
		if _, ok := IsConflict(err); ok {
			return err
		}
		// First-time / empty remote: continue with local
		e.logf("pull skipped: %v", err)
	}

	e.logf("launching Steam app %s (%s)", g.AppID, g.Name)
	if err := launchSteam(g.AppID, e.Cfg.SteamExecutable); err != nil {
		return err
	}

	// Wait until process appears, then until it exits.
	deadline := time.Now().Add(2 * time.Minute)
	seen := false
	for time.Now().Before(deadline) {
		running, err := processx.Running(g.ProcessNames)
		if err != nil {
			return err
		}
		if running {
			seen = true
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	if !seen {
		e.logf("warning: game process not detected within 2m; waiting for manual exit signal via process poll")
	}

	for {
		running, err := processx.Running(g.ProcessNames)
		if err != nil {
			return err
		}
		if seen && !running {
			break
		}
		if !seen && running {
			seen = true
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}

	// Allow final flush to disk
	time.Sleep(2 * time.Second)
	e.logf("game exited; pushing saves...")
	return e.Push(ctx, gameID, prefer)
}

func launchSteam(appID, steamExe string) error {
	uri := "steam://rungameid/" + appID
	if steamExe != "" {
		return exec.Command(steamExe, "-applaunch", appID).Start()
	}
	switch runtime.GOOS {
	case "windows":
		return exec.Command("cmd", "/C", "start", "", uri).Start()
	case "darwin":
		return exec.Command("open", uri).Start()
	default:
		// Linux / Steam Deck
		if err := exec.Command("steam", uri).Start(); err == nil {
			return nil
		}
		if err := exec.Command("xdg-open", uri).Start(); err == nil {
			return nil
		}
		// Flatpak Steam
		if err := exec.Command("flatpak", "run", "com.valvesoftware.Steam", uri).Start(); err == nil {
			return nil
		}
		return fmt.Errorf("could not launch Steam; tried steam/xdg-open/flatpak. URI: %s", uri)
	}
}

// EnsureSteamFriendly is a noop placeholder for future Deck integrations.
func EnsureSteamFriendly() {
	_ = os.Getenv("SteamDeck")
}
