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
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/migzeetigglez/ashen-sync/internal/config"
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

	a := app.NewWithID("com.savechanneler.app")
	icon := appIcon()
	a.SetIcon(icon)
	w := a.NewWindow("Save Channeler")
	w.SetIcon(icon)
	w.Resize(fyne.NewSize(880, 560))

	eng, err := syncer.New(cfg)
	if err != nil {
		return err
	}

	logEntry := widget.NewMultiLineEntry()
	logEntry.Disable()
	logEntry.SetMinRowsVisible(6)

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

	// --- Left column ---
	title := widget.NewRichTextFromMarkdown("# Save Channeler")
	purpose := widget.NewLabel("Sync Dark Souls II & III saves across your PCs with rclone.")
	purpose.Wrapping = fyne.TextWrapWord

	remoteLabel := widget.NewLabel("")
	remoteLabel.Wrapping = fyne.TextWrapWord
	setupBtn := widget.NewButton("Link Google Drive", nil)
	setupBtn.Importance = widget.HighImportance
	resetBtn := widget.NewButton("Change destination…", nil)
	resetBtn.Importance = widget.LowImportance

	leftRemoteBox := container.NewVBox()

	refreshLeftRemote := func() {
		leftRemoteBox.Objects = nil
		if remoteConfigured(cfg) {
			remoteLabel.SetText("Remote: " + cfg.RemoteRoot())
			leftRemoteBox.Add(remoteLabel)
			leftRemoteBox.Add(resetBtn)
		} else {
			leftRemoteBox.Add(setupBtn)
		}
		leftRemoteBox.Refresh()
	}

	iconImg := canvas.NewImageFromResource(icon)
	iconImg.FillMode = canvas.ImageFillContain
	iconImg.SetMinSize(fyne.NewSize(220, 220))

	left := container.NewBorder(
		container.NewVBox(title, purpose, leftRemoteBox),
		nil, nil, nil,
		container.NewCenter(iconImg),
	)

	// --- Right column ---
	gameSelect := widget.NewSelect([]string{"ds2", "ds3", "ds2-vanilla"}, nil)
	gameSelect.SetSelected("ds3")
	statusLabel := widget.NewLabel("Status: —")
	statusLabel.Wrapping = fyne.TextWrapWord

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
		if !remoteConfigured(cfg) {
			fyne.Do(func() { statusLabel.SetText("Status: link Google Drive to get started") })
			return
		}
		id := gameSelect.Selected
		go func() {
			st, err := eng.Status(context.Background(), id)
			fyne.Do(func() {
				if err != nil {
					statusLabel.SetText("Status: " + err.Error())
					return
				}
				statusLabel.SetText(fmt.Sprintf(
					"Status: %s\nPath: %s\nLocal: %s  ·  Cloud: %s  ·  In sync: %v",
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
		if !remoteConfigured(cfg) {
			dialog.ShowInformation("Setup needed", "Link Google Drive first (button on the left).", w)
			return
		}
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

	refreshBtn := widget.NewButton("Refresh", refreshStatus)
	pullBtn := widget.NewButton("Pull", func() { runAction("pull", eng.Pull) })
	pushBtn := widget.NewButton("Push", func() { runAction("push", eng.Push) })
	playBtn := widget.NewButton("Play", func() { runAction("play", eng.Play) })
	watchBtn := widget.NewButton("Start Watcher", nil)
	watchBtn.OnTapped = func() {
		if !remoteConfigured(cfg) {
			dialog.ShowInformation("Setup needed", "Link Google Drive first (button on the left).", w)
			return
		}
		if watcher.Running() {
			watcher.Stop()
			watchBtn.SetText("Start Watcher")
			appendLog("watcher stopped")
			return
		}
		if err := watcher.Start(context.Background()); err != nil {
			dialog.ShowError(err, w)
			return
		}
		watchBtn.SetText("Stop Watcher")
		appendLog("watcher started")
	}

	applyConfig := func(newCfg config.Config) {
		cfg = newCfg
		newEng, eerr := syncer.New(cfg)
		if eerr != nil {
			dialog.ShowError(eerr, w)
			return
		}
		eng = newEng
		eng.Log = appendLog
		watcher.Engine = eng
		watcher.QuietPeriod = cfg.QuietPeriod()
		watcher.PullInterval = cfg.PullInterval()
		refreshLeftRemote()
		refreshStatus()
	}

	setupBtn.OnTapped = func() {
		pathEntry := widget.NewEntry()
		pathEntry.SetText("save-channeler")
		form := widget.NewForm(
			widget.NewFormItem("Folder in Drive", pathEntry),
		)
		dialog.ShowCustomConfirm("Link Google Drive", "Connect", "Cancel", form, func(ok bool) {
			if !ok {
				return
			}
			folder := strings.TrimSpace(pathEntry.Text)
			if folder == "" {
				folder = "save-channeler"
			}
			setupBtn.Disable()
			appendLog("Starting Google Drive setup…")
			prog := dialog.NewProgressInfinite("Linking Google Drive", "Waiting for Google sign-in in your browser…", w)
			prog.Show()
			go func() {
				newCfg, err := setupGoogleDrive(context.Background(), cfg.RclonePath, folder, appendLog)
				fyne.Do(func() {
					prog.Hide()
					setupBtn.Enable()
					if err != nil {
						dialog.ShowError(err, w)
						return
					}
					applyConfig(newCfg)
					dialog.ShowInformation("Connected", "Saves will sync to "+newCfg.RemoteRoot(), w)
				})
			}()
		}, w)
	}

	resetBtn.OnTapped = func() {
		msg := "Disconnect Google Drive and clear the sync destination?\n\n" +
			"Current: " + cfg.RemoteRoot() + "\n\n" +
			"• Signs out the linked Google account from Save Channeler\n" +
			"• Cloud save files are not deleted\n" +
			"• Next “Link Google Drive” will open Google so you can pick another account"
		dialog.ShowConfirm("Reset destination", msg, func(ok bool) {
			if !ok {
				return
			}
			if watcher.Running() {
				watcher.Stop()
				watchBtn.SetText("Start Watcher")
			}
			resetBtn.Disable()
			appendLog("Disconnecting Google Drive…")
			prog := dialog.NewProgressInfinite("Resetting", "Disconnecting Google Drive…", w)
			prog.Show()
			go func() {
				derr := disconnectGoogleDrive(context.Background(), cfg.RclonePath, appendLog)
				fyne.Do(func() {
					prog.Hide()
					resetBtn.Enable()
					if derr != nil {
						dialog.ShowError(derr, w)
						return
					}
					if err := cfg.ClearRemote(); err != nil {
						dialog.ShowError(err, w)
						return
					}
					appendLog("ready — link Google Drive with the account you want")
					applyConfig(cfg)
				})
			}()
		}, w)
	}

	gameRow := container.NewBorder(nil, nil, widget.NewLabel("Game"), refreshBtn, gameSelect)
	actions := container.NewVBox(pullBtn, pushBtn, playBtn, watchBtn)
	rightTop := container.NewVBox(gameRow, widget.NewSeparator(), statusLabel, widget.NewSeparator(), actions)
	right := container.NewBorder(rightTop, nil, nil, nil, container.NewPadded(container.NewVScroll(logEntry)))

	split := container.NewHSplit(
		container.NewPadded(left),
		container.NewPadded(right),
	)
	split.SetOffset(0.40)

	refreshLeftRemote()
	w.SetContent(split)
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
