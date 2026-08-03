package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/migzeetigglez/ashen-sync/internal/config"
	"github.com/migzeetigglez/ashen-sync/internal/detect"
	"github.com/migzeetigglez/ashen-sync/internal/rclonex"
	"github.com/migzeetigglez/ashen-sync/internal/syncer"
	"github.com/migzeetigglez/ashen-sync/internal/watch"
)

func main() {
	if len(os.Args) < 2 {
		// Double-click / .app launch has no args — open the GUI when built with it.
		if err := runGUI(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n\n", err)
			usage()
			os.Exit(1)
		}
		return
	}
	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "help", "-h", "--help":
		usage()
	case "version", "-v", "--version":
		fmt.Println("save-channeler", syncer.ToolVersion)
	case "init":
		err = cmdInit()
	case "doctor":
		err = cmdDoctor()
	case "games":
		err = cmdGames()
	case "status":
		err = cmdStatus(args)
	case "pull":
		err = cmdPullPushPlay("pull", args)
	case "push":
		err = cmdPullPushPlay("push", args)
	case "play":
		err = cmdPullPushPlay("play", args)
	case "watch":
		err = cmdWatch()
	case "gui":
		err = runGUI()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`Save Channeler — open DS2/DS3 cloud saves via your rclone remote

Usage:
  save-channeler init
  save-channeler doctor
  save-channeler games
  save-channeler status [game]
  save-channeler pull [game] [--prefer local|remote]
  save-channeler push [game] [--prefer local|remote]
  save-channeler play <game> [--prefer local|remote]
  save-channeler watch
`)
	fmt.Print(guiUsageLine())
	fmt.Print(`  save-channeler version

Games: ds2 | ds3 | ds2-vanilla

Security: Save Channeler never embeds cloud API secrets. Configure rclone yourself
and point this tool at a dedicated remote folder.

Runtime targets: Windows, Linux, Steam Deck (not macOS — DS2/DS3 are not native there).
`)
}

func loadCfg() (config.Config, error) {
	cfg, err := config.Load()
	if errors.Is(err, os.ErrNotExist) {
		return config.Default(), fmt.Errorf("config not found; run `save-channeler init`")
	}
	return cfg, err
}

func engine() (syncer.Engine, error) {
	cfg, err := loadCfg()
	if err != nil {
		return syncer.Engine{}, err
	}
	if err := cfg.Validate(); err != nil {
		return syncer.Engine{}, err
	}
	eng, err := syncer.New(cfg)
	if err != nil {
		return eng, err
	}
	eng.Log = func(s string) { fmt.Println(s) }
	return eng, nil
}

func cmdInit() error {
	cfg := config.Default()
	if existing, err := config.Load(); err == nil {
		cfg = existing
	}

	rc := rclonex.Client{Bin: cfg.RclonePath}
	ctx := context.Background()
	ver, err := rc.Version(ctx)
	if err != nil {
		return fmt.Errorf("rclone not found (%v). Install from https://rclone.org/install/ then re-run init", err)
	}
	fmt.Println("Found", ver)

	remotes, err := rc.ListRemotes(ctx)
	if err != nil {
		return err
	}
	if len(remotes) == 0 {
		return errors.New("no rclone remotes configured. Run `rclone config` and create one (use a dedicated folder), then re-run init")
	}
	fmt.Println("Remotes:")
	for i, r := range remotes {
		fmt.Printf("  [%d] %s\n", i+1, r)
	}

	in := bufio.NewReader(os.Stdin)
	fmt.Print("Remote name: ")
	remote, _ := in.ReadString('\n')
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return errors.New("remote is required")
	}

	fmt.Printf("Remote path [%s]: ", cfg.RemotePath)
	path, _ := in.ReadString('\n')
	path = strings.TrimSpace(path)
	if path != "" {
		cfg.RemotePath = path
	}
	cfg.Remote = remote

	root := cfg.RemoteRoot()
	fmt.Println("Checking access to", root, "…")
	if err := rc.About(ctx, root); err != nil {
		fmt.Println("note: about failed (folder may be empty/new):", err)
	}

	if err := cfg.Save(); err != nil {
		return err
	}
	p, _ := config.Path()
	fmt.Println("Wrote", p)
	fmt.Println("Tip: use a dedicated cloud folder only for save-channeler.")
	return nil
}

func cmdDoctor() error {
	cfg, err := config.Load()
	if errors.Is(err, os.ErrNotExist) {
		cfg = config.Default()
		fmt.Println("config: missing (run init)")
	} else if err != nil {
		return err
	} else {
		p, _ := config.Path()
		fmt.Println("config:", p)
		fmt.Println("remote:", cfg.RemoteRoot())
	}

	rc := rclonex.Client{Bin: cfg.RclonePath}
	ctx := context.Background()
	fmt.Println("rclone binary:", rc.ResolvedPath())
	if ver, err := rc.Version(ctx); err != nil {
		fmt.Println("rclone: NOT FOUND —", err)
		fmt.Println("hint: packaged builds ship rclone next to save-channeler; or set rclone_path in config")
	} else {
		fmt.Println("rclone:", ver)
		if cfg.Remote != "" {
			if err := rc.About(ctx, cfg.RemoteRoot()); err != nil {
				fmt.Println("rclone remote check:", err)
			} else {
				fmt.Println("rclone remote check: ok")
			}
		}
	}

	locs, err := detect.FindSaves(cfg.GameOverrides)
	if err != nil {
		return err
	}
	fmt.Println("saves:")
	for _, loc := range locs {
		fmt.Printf("  %s exists=%v platform=%s\n    %s\n", loc.Game.ID, loc.Exists, loc.Platform, loc.Path)
	}
	return nil
}

func cmdGames() error {
	cfg, _ := config.Load()
	if cfg.GameOverrides == nil {
		cfg = config.Default()
	}
	locs, err := detect.FindSaves(cfg.GameOverrides)
	if err != nil {
		return err
	}
	for _, loc := range locs {
		fmt.Printf("%s\t%s\texists=%v\t%s\n", loc.Game.ID, loc.Game.AppID, loc.Exists, loc.Path)
	}
	return nil
}

func cmdStatus(args []string) error {
	eng, err := engine()
	if err != nil {
		return err
	}
	games := []string{"ds2", "ds3", "ds2-vanilla"}
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		games = []string{args[0]}
	}
	ctx := context.Background()
	for _, g := range games {
		st, err := eng.Status(ctx, g)
		if err != nil {
			fmt.Printf("%s: %v\n", g, err)
			continue
		}
		fmt.Printf("%s\n  path: %s\n  local: %s\n  remote: %s\n  last_sync: %s\n  in_sync: %v\n",
			g, st.Location.Path, trunc(st.LocalHash), trunc(st.RemoteHash), trunc(st.LastSync), st.InSync)
	}
	return nil
}

func cmdPullPushPlay(kind string, args []string) error {
	prefer := syncer.PreferAbort
	var game string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--prefer" && i+1 < len(args) {
			prefer = syncer.ResolvePrefer(args[i+1])
			i++
			continue
		}
		if !strings.HasPrefix(a, "-") && game == "" {
			game = a
		}
	}
	if game == "" {
		if kind == "play" {
			return errors.New("play requires a game id (ds2|ds3)")
		}
		game = "ds3"
	}
	eng, err := engine()
	if err != nil {
		return err
	}
	ctx := context.Background()
	var run func(context.Context, string, syncer.Prefer) error
	switch kind {
	case "pull":
		run = eng.Pull
	case "push":
		run = eng.Push
	case "play":
		run = eng.Play
	}
	err = run(ctx, game, prefer)
	if c, ok := syncer.IsConflict(err); ok {
		return fmt.Errorf("%w\nResolve with --prefer local or --prefer remote", c)
	}
	return err
}

func cmdWatch() error {
	eng, err := engine()
	if err != nil {
		return err
	}
	w := &watch.Watcher{
		Engine:       eng,
		QuietPeriod:  eng.Cfg.QuietPeriod(),
		PullInterval: eng.Cfg.PullInterval(),
		OnEvent: func(ev watch.Event) {
			fmt.Printf("%s [%s] %s\n", ev.Time.Format(time.RFC3339), ev.Level, ev.Message)
		},
		OnConflict: func(c *syncer.ConflictError, action string) {
			fmt.Printf("CONFLICT during %s: %s\nUse save-channeler %s --prefer local|remote\n", action, c.Error(), action)
		},
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := w.Start(ctx); err != nil {
		return err
	}
	fmt.Println("watcher running; Ctrl+C to stop")
	<-ctx.Done()
	w.Stop()
	return nil
}

func trunc(h string) string {
	if h == "" {
		return "—"
	}
	if len(h) < 12 {
		return h
	}
	return h[:12]
}
