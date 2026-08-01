//go:build !nogui

package gui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/migzeetigglez/ashen-sync/internal/config"
	"github.com/migzeetigglez/ashen-sync/internal/detect"
	"github.com/migzeetigglez/ashen-sync/internal/syncer"
	"github.com/migzeetigglez/ashen-sync/internal/watch"
)

func Run() error {
	cfg, err := config.Load()
	if errors.Is(err, os.ErrNotExist) {
		cfg = config.Default()
	} else if err != nil {
		return err
	}

	a := app.NewWithID("com.ashen-sync.app")
	w := a.NewWindow("Ashen Sync")
	w.Resize(fyne.NewSize(820, 600))

	eng, err := syncer.New(cfg)
	if err != nil {
		return err
	}

	logEntry := widget.NewMultiLineEntry()
	logEntry.Disable()
	logEntry.SetMinRowsVisible(14)

	var mu sync.Mutex
	appendLog := func(line string) {
		fyne.Do(func() {
			mu.Lock()
			defer mu.Unlock()
			cur := logEntry.Text
			if cur != "" {
				cur += "\n"
			}
			logEntry.SetText(cur + time.Now().Format("15:04:05") + " " + line)
		})
	}
	eng.Log = appendLog

	gameSelect := widget.NewSelect([]string{"ds2", "ds3", "ds2-vanilla"}, nil)
	gameSelect.SetSelected("ds3")

	statusLabel := widget.NewLabel("Status: —")
	statusLabel.Wrapping = fyne.TextWrapWord
	remoteLabel := widget.NewLabel(fmt.Sprintf("Remote: %s", cfg.RemoteRoot()))
	if cfg.Remote == "" {
		remoteLabel.SetText("Remote: not configured — run: ashen-sync init")
	}

	showConflict := func(c *syncer.ConflictError, name string, fn func(context.Context, syncer.Prefer) error) {
		fyne.Do(func() {
			msg := widget.NewLabel(c.Error() + "\n\nUse local (upload this machine) or remote (download cloud copy).")
			msg.Wrapping = fyne.TextWrapWord
			d := dialog.NewCustomConfirm("Save conflict", "Use local", "Use remote", msg, func(useLocal bool) {
				go func() {
					pref := syncer.PreferRemote
					if useLocal {
						pref = syncer.PreferLocal
					}
					if err := fn(context.Background(), pref); err != nil {
						appendLog(name + " failed: " + err.Error())
						return
					}
					appendLog(name + " ok")
				}()
			}, w)
			d.Resize(fyne.NewSize(480, 220))
			d.Show()
		})
	}

	watcher := &watch.Watcher{
		Engine:       eng,
		QuietPeriod:  cfg.QuietPeriod(),
		PullInterval: cfg.PullInterval(),
		OnEvent: func(ev watch.Event) {
			appendLog(fmt.Sprintf("[%s] %s", ev.Level, ev.Message))
		},
		OnConflict: func(c *syncer.ConflictError) {
			appendLog("conflict: " + c.Error())
			showConflict(c, "pull", func(ctx context.Context, prefer syncer.Prefer) error {
				return eng.Pull(ctx, c.GameID, prefer)
			})
		},
	}

	refreshStatus := func() {
		id := gameSelect.Selected
		go func() {
			st, err := eng.Status(context.Background(), id)
			fyne.Do(func() {
				if err != nil {
					statusLabel.SetText("Status: " + err.Error())
					return
				}
				statusLabel.SetText(fmt.Sprintf(
					"Status: %s | path=%s | local=%s | remote=%s | in_sync=%v",
					st.Location.Game.ID,
					st.Location.Path,
					trunc(st.LocalHash),
					trunc(st.RemoteHash),
					st.InSync,
				))
			})
		}()
	}
	gameSelect.OnChanged = func(string) { refreshStatus() }

	runAction := func(name string, fn func(context.Context, string, syncer.Prefer) error) {
		id := gameSelect.Selected
		go func() {
			appendLog(name + " " + id + "…")
			err := fn(context.Background(), id, syncer.PreferAbort)
			if c, ok := syncer.IsConflict(err); ok {
				showConflict(c, name, func(ctx context.Context, prefer syncer.Prefer) error {
					return fn(ctx, id, prefer)
				})
				return
			}
			if err != nil {
				appendLog(name + " failed: " + err.Error())
				return
			}
			appendLog(name + " ok")
			refreshStatus()
		}()
	}

	pullBtn := widget.NewButton("Pull", func() { runAction("pull", eng.Pull) })
	pushBtn := widget.NewButton("Push", func() { runAction("push", eng.Push) })
	playBtn := widget.NewButton("Play", func() { runAction("play", eng.Play) })

	watchBtn := widget.NewButton("Start watcher", nil)
	watchBtn.OnTapped = func() {
		if watcher.Running() {
			watcher.Stop()
			watchBtn.SetText("Start watcher")
			appendLog("watcher stopped")
			return
		}
		if err := watcher.Start(context.Background()); err != nil {
			dialog.ShowError(err, w)
			return
		}
		watchBtn.SetText("Stop watcher")
		appendLog("watcher started (debounced; skips while game runs)")
	}

	refreshBtn := widget.NewButton("Refresh", refreshStatus)

	gamesList := widget.NewLabel("")
	gamesList.Wrapping = fyne.TextWrapWord
	refreshGames := func() {
		locs, err := detect.FindSaves(cfg.GameOverrides)
		if err != nil {
			gamesList.SetText(err.Error())
			return
		}
		var b strings.Builder
		for _, loc := range locs {
			fmt.Fprintf(&b, "• %s — %s\n  %s\n  exists=%v  platform=%s\n\n",
				loc.Game.ID, loc.Game.Name, loc.Path, loc.Exists, loc.Platform)
		}
		if b.Len() == 0 {
			b.WriteString("No save folders detected yet.")
		}
		gamesList.SetText(b.String())
	}

	header := container.NewVBox(
		widget.NewRichTextFromMarkdown("## Ashen Sync\nOpen DS2/DS3 cloud saves via **your** rclone remote. No embedded API secrets."),
		remoteLabel,
		container.NewHBox(widget.NewLabel("Game"), gameSelect, refreshBtn),
		statusLabel,
		container.NewHBox(pullBtn, pushBtn, playBtn, watchBtn),
	)

	w.SetContent(container.NewBorder(header, nil, nil, nil,
		container.NewVSplit(container.NewVScroll(gamesList), container.NewVScroll(logEntry)),
	))
	refreshGames()
	refreshStatus()
	w.ShowAndRun()
	watcher.Stop()
	return nil
}

func trunc(h string) string {
	if h == "" {
		return "—"
	}
	if len(h) < 10 {
		return h
	}
	return h[:10]
}
