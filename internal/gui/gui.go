//go:build !nogui

package gui

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
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

	applyStartupScale(cfg.UIScaleOrDefault())

	a := app.NewWithID("com.savechanneler.app")
	icon := appIcon()
	a.SetIcon(icon)
	w := a.NewWindow("Save Channeler")
	w.SetIcon(icon)
	// Logical size; FYNE_SCALE / settings.json multiply physical pixels.
	w.Resize(fyne.NewSize(960, 620))
	w.SetFixedSize(false)

	eng, err := syncer.New(cfg)
	if err != nil {
		return err
	}

	logEntry := widget.NewMultiLineEntry()
	logEntry.Disable()
	logEntry.SetMinRowsVisible(6)
	logEntry.TextStyle = fyne.TextStyle{Monospace: true}
	// Disabled Entry uses ColorNameDisabled (low contrast). Force foreground.
	baseTheme := a.Settings().Theme()
	if baseTheme == nil {
		baseTheme = theme.DefaultTheme()
	}
	logView := container.NewThemeOverride(logEntry, contrastLogTheme{base: baseTheme})

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

	var applyConfig func(config.Config)

	// --- Right column controls needed early for local-path refresh ---
	gameSelect := widget.NewSelect([]string{"ds2", "ds3", "ds2-vanilla"}, nil)
	gameSelect.SetSelected("ds3")
	statusLabel := widget.NewLabel("Status: —")
	statusLabel.Wrapping = fyne.TextWrapWord

	// --- Left column ---
	title := widget.NewLabelWithStyle("Save Channeler", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	purpose := widget.NewLabelWithStyle(
		"Sync Dark Souls II & III saves across your PCs with rclone.",
		fyne.TextAlignCenter,
		fyne.TextStyle{},
	)
	purpose.Wrapping = fyne.TextWrapWord

	// Uniform path rows: Label | outlined field | action button
	remotePathEntry := widget.NewEntry()
	remotePathEntry.SetPlaceHolder("Select remote save folder…")
	remotePathEntry.Disable()
	remotePathView := container.NewThemeOverride(remotePathEntry, contrastLogTheme{base: baseTheme})
	remoteBtn := widget.NewButton("Set", nil)

	localPathEntry := widget.NewEntry()
	localPathEntry.SetPlaceHolder("Select local save folder…")
	localPathEntry.Disable()
	localPathView := container.NewThemeOverride(localPathEntry, contrastLogTheme{base: baseTheme})
	localBtn := widget.NewButton("Browse", nil)

	// Match label + button column widths so the outlined fields align.
	labelColW := maxFloat32(
		widget.NewLabel("Remote:").MinSize().Width,
		widget.NewLabel("Local:").MinSize().Width,
	)
	btnColW := maxFloat32(
		widget.NewButton("Change", nil).MinSize().Width,
		widget.NewButton("Browse", nil).MinSize().Width,
		widget.NewButton("Set", nil).MinSize().Width,
	)

	refreshLeftRemote := func() {
		if remoteConfigured(cfg) {
			remotePathEntry.SetText(cfg.RemoteRoot())
			remoteBtn.SetText("Change")
		} else {
			remotePathEntry.SetText("")
			remotePathEntry.SetPlaceHolder("Select remote save folder…")
			remoteBtn.SetText("Set")
		}
	}

	refreshLeftLocal := func() {
		id := gameSelect.Selected
		if id == "" {
			id = "ds3"
		}
		path := ""
		if cfg.GameOverrides != nil {
			path = cfg.GameOverrides[id]
		}
		if path == "" {
			if loc, err := detect.FindGame(id, cfg.GameOverrides); err == nil && loc.Exists {
				path = loc.Path
			}
		}
		if path == "" {
			localPathEntry.SetText("")
			localPathEntry.SetPlaceHolder("Select local save folder…")
			localBtn.SetText("Browse")
		} else {
			localPathEntry.SetText(path)
			localBtn.SetText("Change")
		}
	}

	localBtn.OnTapped = func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if uri == nil {
				return
			}
			path := uri.Path()
			if path == "" {
				return
			}
			// If user picked the parent DarkSoulsIII folder, prefer the steamID child.
			if info, err := os.Stat(path); err == nil && info.IsDir() {
				if kids, _ := os.ReadDir(path); len(kids) > 0 {
					base := filepath.Base(path)
					if strings.EqualFold(base, "DarkSoulsIII") || strings.EqualFold(base, "DarkSoulsII") {
						for _, k := range kids {
							if k.IsDir() && looksLikeSteamIDDir(k.Name()) {
								path = filepath.Join(path, k.Name())
								break
							}
						}
					}
				}
			}
			id := gameSelect.Selected
			if id == "" {
				id = "ds3"
			}
			if cfg.GameOverrides == nil {
				cfg.GameOverrides = map[string]string{}
			}
			cfg.GameOverrides[id] = path
			if err := cfg.Save(); err != nil {
				dialog.ShowError(err, w)
				return
			}
			appendLog("local path for " + id + " → " + path)
			applyConfig(cfg)
		}, w)
	}

	remoteRow := container.NewBorder(
		nil, nil,
		fixedWidth(labelColW, widget.NewLabel("Remote:")),
		fixedWidth(btnColW, remoteBtn),
		remotePathView,
	)
	localRow := container.NewBorder(
		nil, nil,
		fixedWidth(labelColW, widget.NewLabel("Local:")),
		fixedWidth(btnColW, localBtn),
		localPathView,
	)

	autoDetectBtn := widget.NewButton("Auto-detect save folder", nil)
	autoDetectBtn.Importance = widget.HighImportance
	pillTheme := pillButtonTheme{base: baseTheme}
	autoDetectPill := container.NewCenter(
		container.NewThemeOverride(autoDetectBtn, pillTheme),
	)

	// runAutoDetect finds/validates a save folder. quietSkip hides the "already set" dialog (first-run).
	runAutoDetect := func(quietSkip bool) {
		id := gameSelect.Selected
		if id == "" {
			id = "ds3"
		}
		autoDetectBtn.Disable()
		appendLog("auto-detecting " + id + "…")
		go func() {
			loc, files, err := detect.AutoDetect(id)
			fyne.Do(func() {
				autoDetectBtn.Enable()
				if err != nil {
					appendLog("auto-detect failed: " + err.Error())
					dialog.ShowInformation(
						"No save found",
						err.Error()+"\n\nLaunch the game once to create a save, or use Browse to pick the folder manually.",
						w,
					)
					return
				}
				if err := detect.ValidateSaveDir(loc.Path); err != nil {
					appendLog("auto-detect invalid: " + err.Error())
					dialog.ShowError(fmt.Errorf("found folder but validation failed: %w", err), w)
					return
				}
				fileList := strings.Join(files, ", ")
				if fileList == "" {
					fileList = "(none)"
				}
				current := ""
				if cfg.GameOverrides != nil {
					current = cfg.GameOverrides[id]
				}
				if current == loc.Path {
					appendLog("auto-detect: already set → " + loc.Path)
					if !quietSkip {
						dialog.ShowInformation(
							"Already set",
							fmt.Sprintf("Local path is already:\n%s\n\nSave files: %s", loc.Path, fileList),
							w,
						)
					}
					return
				}
				msg := fmt.Sprintf(
					"Found %s save folder:\n\n%s\n\nSave files: %s\n\nSet this as the local path?",
					loc.Game.Name, loc.Path, fileList,
				)
				appendLog("auto-detect found → " + loc.Path)
				dialog.ShowConfirm("Use detected save folder?", msg, func(ok bool) {
					if !ok {
						appendLog("auto-detect dismissed")
						return
					}
					if cfg.GameOverrides == nil {
						cfg.GameOverrides = map[string]string{}
					}
					cfg.GameOverrides[id] = loc.Path
					if err := cfg.Save(); err != nil {
						dialog.ShowError(err, w)
						return
					}
					appendLog("local path for " + id + " → " + loc.Path)
					applyConfig(cfg)
				}, w)
			})
		}()
	}

	autoDetectBtn.OnTapped = func() { runAutoDetect(false) }

	iconImg := canvas.NewImageFromResource(icon)
	iconImg.FillMode = canvas.ImageFillContain
	iconImg.SetMinSize(fyne.NewSize(220, 220))

	autoDetectGap := canvas.NewRectangle(color.Transparent)
	autoDetectGap.SetMinSize(fyne.NewSize(1, 15))

	left := container.NewVBox(
		title,
		purpose,
		container.NewCenter(iconImg),
		widget.NewSeparator(),
		remoteRow,
		localRow,
		autoDetectGap,
		autoDetectPill,
	)

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
			fyne.Do(func() { statusLabel.SetText("Status: set a remote destination to get started") })
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
	gameSelect.OnChanged = func(string) {
		refreshLeftLocal()
		refreshStatus()
	}

	runAction := func(name string, fn func(context.Context, string, syncer.Prefer) error) {
		if !remoteConfigured(cfg) {
			dialog.ShowInformation("Setup needed", "Set a remote destination first (Remote row on the left).", w)
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
			dialog.ShowInformation("Setup needed", "Set a remote destination first (Remote row on the left).", w)
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

	applyConfig = func(newCfg config.Config) {
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
		refreshLeftLocal()
		refreshStatus()
	}

	linkRemote := func() {
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
			remoteBtn.Disable()
			appendLog("Starting Google Drive setup…")
			prog := dialog.NewProgressInfinite("Linking Google Drive", "Waiting for Google sign-in in your browser…", w)
			prog.Show()
			go func() {
				newCfg, err := setupGoogleDrive(context.Background(), eng.Rclone.ResolvedPath(), folder, appendLog)
				fyne.Do(func() {
					prog.Hide()
					remoteBtn.Enable()
					if err != nil {
						dialog.ShowError(err, w)
						return
					}
					applyConfig(newCfg)
					id := gameSelect.Selected
					if id == "" {
						id = "ds3"
					}
					if cfg.GameOverrides == nil || strings.TrimSpace(cfg.GameOverrides[id]) == "" {
						runAutoDetect(true)
					}
					dialog.ShowInformation("Connected", "Saves will sync to "+newCfg.RemoteRoot(), w)
				})
			}()
		}, w)
	}

	changeRemote := func() {
		msg := "Disconnect Google Drive and clear the sync destination?\n\n" +
			"Current: " + cfg.RemoteRoot() + "\n\n" +
			"• Signs out the linked Google account from Save Channeler\n" +
			"• Cloud save files are not deleted\n" +
			"• Next “Set” will open Google so you can pick another account"
		dialog.ShowConfirm("Change remote", msg, func(ok bool) {
			if !ok {
				return
			}
			if watcher.Running() {
				watcher.Stop()
				watchBtn.SetText("Start Watcher")
			}
			remoteBtn.Disable()
			appendLog("Disconnecting Google Drive…")
			prog := dialog.NewProgressInfinite("Resetting", "Disconnecting Google Drive…", w)
			prog.Show()
			go func() {
				derr := disconnectGoogleDrive(context.Background(), eng.Rclone.ResolvedPath(), appendLog)
				fyne.Do(func() {
					prog.Hide()
					remoteBtn.Enable()
					if derr != nil {
						dialog.ShowError(derr, w)
						return
					}
					if err := cfg.ClearRemote(); err != nil {
						dialog.ShowError(err, w)
						return
					}
					appendLog("ready — use Set to link Google Drive")
					applyConfig(cfg)
				})
			}()
		}, w)
	}

	remoteBtn.OnTapped = func() {
		if remoteConfigured(cfg) {
			changeRemote()
			return
		}
		linkRemote()
	}

	gameRow := container.NewBorder(nil, nil, widget.NewLabel("Game"), refreshBtn, gameSelect)
	actions := container.NewVBox(pullBtn, pushBtn, playBtn, watchBtn)
	scaleRow := newUIScaleControl(&cfg, w, func(updated config.Config) {
		cfg = updated
	})
	rightTop := container.NewVBox(
		gameRow,
		widget.NewSeparator(),
		statusLabel,
		widget.NewSeparator(),
		actions,
		widget.NewSeparator(),
		scaleRow,
	)
	right := container.NewBorder(rightTop, nil, nil, nil, container.NewPadded(container.NewVScroll(logView)))

	split := container.NewHSplit(
		container.NewPadded(left),
		container.NewPadded(right),
	)
	split.SetOffset(0.40)

	refreshLeftRemote()
	refreshLeftLocal()
	w.SetContent(split)
	refreshStatus()

	appendLog("rclone: " + eng.Rclone.ResolvedPath())

	localUnset := func() bool {
		id := gameSelect.Selected
		if id == "" {
			id = "ds3"
		}
		if cfg.GameOverrides != nil && strings.TrimSpace(cfg.GameOverrides[id]) != "" {
			return false
		}
		return true
	}

	a.Lifecycle().SetOnStarted(func() {
		go func() {
			time.Sleep(250 * time.Millisecond)
			fyne.Do(func() {
				if !remoteConfigured(cfg) {
					dialog.ShowInformation(
						"Welcome to Save Channeler",
						"Set your remote cloud folder first (Remote → Set).\n\n"+
							"After that, Save Channeler can auto-detect your Dark Souls save folder.",
						w,
					)
					return
				}
				if localUnset() {
					runAutoDetect(true)
				}
			})
		}()
	})

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

// contrastLogTheme paints disabled widgets with normal foreground so log text stays readable.
type contrastLogTheme struct {
	base fyne.Theme
}

func (t contrastLogTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	if n == theme.ColorNameDisabled {
		return t.base.Color(theme.ColorNameForeground, v)
	}
	return t.base.Color(n, v)
}

func (t contrastLogTheme) Font(s fyne.TextStyle) fyne.Resource { return t.base.Font(s) }
func (t contrastLogTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return t.base.Icon(n)
}
func (t contrastLogTheme) Size(n fyne.ThemeSizeName) float32 { return t.base.Size(n) }

func looksLikeSteamIDDir(name string) bool {
	if len(name) < 5 {
		return false
	}
	for _, r := range name {
		if r < '0' || r > '9' {
			// SteamID folders are often hex-ish like 011000011a0eed1a
			if (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
				return false
			}
		}
	}
	return true
}

func maxFloat32(vals ...float32) float32 {
	var m float32
	for i, v := range vals {
		if i == 0 || v > m {
			m = v
		}
	}
	return m
}

// fixedWidth wraps obj so MinSize width is at least w (keeps columns aligned across rows).
func fixedWidth(w float32, obj fyne.CanvasObject) fyne.CanvasObject {
	return container.New(&fixedWidthLayout{width: w}, obj)
}

type fixedWidthLayout struct {
	width float32
}

func (f *fixedWidthLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	obj := objects[0]
	min := obj.MinSize()
	// Center vertically; left-align content within the fixed column.
	y := (size.Height - min.Height) / 2
	if y < 0 {
		y = 0
	}
	obj.Move(fyne.NewPos(0, y))
	obj.Resize(fyne.NewSize(min.Width, min.Height))
}

func (f *fixedWidthLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	h := float32(0)
	if len(objects) > 0 {
		h = objects[0].MinSize().Height
	}
	return fyne.NewSize(f.width, h)
}

// pillButtonTheme slightly rounds the auto-detect button (not a full capsule).
type pillButtonTheme struct {
	base fyne.Theme
}

func (t pillButtonTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	return t.base.Color(n, v)
}
func (t pillButtonTheme) Font(s fyne.TextStyle) fyne.Resource { return t.base.Font(s) }
func (t pillButtonTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return t.base.Icon(n)
}
func (t pillButtonTheme) Size(n fyne.ThemeSizeName) float32 {
	if n == theme.SizeNameInputRadius {
		return 6
	}
	return t.base.Size(n)
}
