//go:build !nogui

package gui

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"net/url"
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
	fynetooltip "github.com/dweymouth/fyne-tooltip"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/migzeetigglez/ashen-sync/internal/config"
	"github.com/migzeetigglez/ashen-sync/internal/detect"
	"github.com/migzeetigglez/ashen-sync/internal/mods"
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

	baseTheme := a.Settings().Theme()
	if baseTheme == nil {
		baseTheme = theme.DefaultTheme()
	}

	logRich := widget.NewRichText()
	logRich.Wrapping = fyne.TextWrapBreak
	logScroll := container.NewVScroll(logRich)
	// Don't let long lines dictate window/split MinSize — fill allocated space only.
	logPane := container.New(&shrinkFillLayout{}, logScroll)

	const (
		logToneNormal   = iota
		logToneValidate // Validate Saves — distinct color in the activity log
	)

	var mu sync.Mutex
	appendLogTone := func(line string, tone int) {
		fyne.Do(func() {
			mu.Lock()
			defer mu.Unlock()
			colorName := theme.ColorNameForeground
			switch tone {
			case logToneValidate:
				colorName = theme.ColorNamePrimary
			}
			seg := &widget.TextSegment{
				Text: time.Now().Format("15:04:05") + " " + line + "\n",
				Style: widget.RichTextStyle{
					ColorName: colorName,
					SizeName:  theme.SizeNameText,
					TextStyle: fyne.TextStyle{Monospace: true},
				},
			}
			logRich.Segments = append(logRich.Segments, seg)
			logRich.Refresh()
			logScroll.ScrollToBottom()
		})
	}
	appendLog := func(line string) {
		appendLogTone(line, logToneNormal)
	}
	eng.Log = appendLog

	var applyConfig func(config.Config)

	// --- Right column controls needed early for local-path refresh ---
	gameSelect := ttwidget.NewSelect([]string{"ds2", "ds3", "ds2-vanilla"}, nil)
	gameSelect.SetSelected("ds3")
	gameSelect.SetToolTip("Which game’s saves to sync and show mods for.")
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
	remoteBtn := ttwidget.NewButton("Set", nil)
	remoteBtn.SetToolTip("Link Google Drive as the cloud sync destination, or change / disconnect the linked account. The path below updates per selected game.")

	localPathEntry := widget.NewEntry()
	localPathEntry.SetPlaceHolder("Select local save folder…")
	localPathEntry.Disable()
	localPathView := container.NewThemeOverride(localPathEntry, contrastLogTheme{base: baseTheme})
	localBtn := ttwidget.NewButton("Browse", nil)
	localBtn.SetToolTip("Choose the local save folder for the selected game.")

	// Match label + button column widths so the outlined fields align.
	labelColW := maxFloat32(
		widget.NewLabel("Game:").MinSize().Width,
		widget.NewLabel("Remote:").MinSize().Width,
		widget.NewLabel("Local:").MinSize().Width,
	)
	btnColW := maxFloat32(
		widget.NewButton("Change", nil).MinSize().Width,
		widget.NewButton("Browse", nil).MinSize().Width,
		widget.NewButton("Set", nil).MinSize().Width,
	)

	refreshLeftRemote := func() {
		if !remoteConfigured(cfg) {
			remotePathEntry.SetText("")
			remotePathEntry.SetPlaceHolder("Select remote save folder…")
			remoteBtn.SetText("Set")
			return
		}
		remoteBtn.SetText("Change")
		id := gameSelect.Selected
		if id == "" {
			id = "ds3"
		}
		// Per-game cloud path under the Drive root (same root for all games).
		var full string
		if loc, err := detect.FindGame(id, cfg.GameOverrides); err == nil && loc.SteamID != "" {
			full = fmt.Sprintf("%s/%s/%s", cfg.RemoteRoot(), loc.Game.RemoteKey, loc.SteamID)
		} else if g, ok := detect.ByID(id); ok {
			full = fmt.Sprintf("%s/%s/…", cfg.RemoteRoot(), g.RemoteKey)
		} else {
			full = cfg.RemoteRoot()
		}
		remotePathEntry.SetText(displayPath(full))
	}

	var refreshModsList func(gameID, saveDir string)

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
			localPathEntry.SetText(displayPath(path))
			localBtn.SetText("Change")
		}
		refreshModsList(id, path)
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

	gameRow := container.NewBorder(
		nil, nil,
		fixedWidth(labelColW, widget.NewLabel("Game:")),
		nil,
		gameSelect,
	)
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

	autoDetectBtn := ttwidget.NewButton("Auto-detect save folder", nil)
	autoDetectBtn.Importance = widget.HighImportance
	autoDetectBtn.SetToolTip("Search this PC for a Dark Souls save folder and set it as Local.")
	validateSavesBtn := ttwidget.NewButton("Validate Saves", nil)
	validateSavesBtn.Importance = widget.HighImportance
	validateSavesBtn.SetToolTip("Re-check whether the local and cloud saves match for the selected game.")
	pillTheme := pillButtonTheme{base: baseTheme}
	pillRow := container.NewGridWithColumns(2,
		container.NewThemeOverride(autoDetectBtn, pillTheme),
		container.NewThemeOverride(validateSavesBtn, pillTheme),
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

	modsHeading := widget.NewLabelWithStyle("Mods", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	modsEmpty := widget.NewLabel("No mod log in this save folder.")
	modsEmpty.Wrapping = fyne.TextWrapWord
	modsListBox := container.NewVBox()
	modsBody := container.NewVBox(modsEmpty, modsListBox)
	modsScroll := container.NewVScroll(modsBody)
	modsScroll.SetMinSize(fyne.NewSize(0, 120))
	modsSection := container.NewBorder(modsHeading, nil, nil, nil, modsScroll)

	refreshModsUI := func() {
		modsListBox.Refresh()
		modsBody.Refresh()
		modsScroll.Refresh()
		modsSection.Refresh()
	}

	refreshModsList = func(gameID, saveDir string) {
		modsListBox.Objects = nil
		g, ok := detect.ByID(gameID)
		if !ok {
			modsHeading.SetText("Mods")
			modsEmpty.SetText("Mods are only tracked for DS2, DS2 SotFS, and DS3.")
			modsEmpty.Show()
			refreshModsUI()
			return
		}
		label := shortGameLabel(g.ID)
		modsHeading.SetText("Mods (" + label + ")")

		var logEntries []mods.Entry
		if saveDir != "" && saveDirLooksLikeGame(saveDir, g) {
			_, entries, err := mods.Load(saveDir)
			if err != nil {
				modsEmpty.SetText("Could not read mod log: " + err.Error())
				modsEmpty.Show()
				refreshModsUI()
				return
			}
			logEntries = entries
		}

		exeDir := detect.FindExeDir(g)
		autoEntries := mods.DetectInstall(g.ID, exeDir)
		entries := mods.Merge(autoEntries, logEntries)

		if len(entries) == 0 {
			modsEmpty.SetText("No mods detected for this " + label + " install.")
			modsEmpty.Show()
			refreshModsUI()
			return
		}
		modsEmpty.Hide()
		for _, e := range entries {
			e := e
			if e.URL != "" {
				u, perr := url.Parse(e.URL)
				if perr == nil && u.Scheme != "" {
					modsListBox.Add(widget.NewHyperlink(e.Name, u))
					continue
				}
			}
			modsListBox.Add(widget.NewLabel("• " + e.Name))
		}
		refreshModsUI()
	}

	scaleRow := newUIScaleControl(&cfg, w, func(updated config.Config) {
		cfg = updated
	})

	leftTop := container.NewVBox(
		scaleRow,
		widget.NewSeparator(),
		title,
		purpose,
		container.NewCenter(iconImg),
		widget.NewSeparator(),
		gameRow,
		remoteRow,
		localRow,
		autoDetectGap,
		pillRow,
		widget.NewSeparator(),
	)
	// Mods fill remaining height and scroll when the list is long.
	leftInner := container.NewBorder(leftTop, nil, nil, nil, modsSection)
	left := container.New(&shrinkFillLayout{}, leftInner)

	var refreshStatus func()

	showConflict := func(c *syncer.ConflictError) {
		fyne.Do(func() {
			msg := widget.NewLabel(c.Error() + "\n\n" +
				"Use local — keep this PC’s save and upload it to the cloud (local file stays).\n" +
				"Use remote — download the cloud save over this PC’s copy (a snapshot is kept first).\n\n" +
				"Cancel / close — do nothing; your local save is unchanged.")
			msg.Wrapping = fyne.TextWrapWord

			d := dialog.NewCustomWithoutButtons("Save conflict", msg, w)
			resolve := func(keepLocal bool) {
				d.Hide()
				go func() {
					var err error
					if keepLocal {
						appendLog("conflict: keeping local, uploading…")
						err = eng.Push(context.Background(), c.GameID, syncer.PreferLocal)
					} else {
						appendLog("conflict: using remote, downloading…")
						err = eng.Pull(context.Background(), c.GameID, syncer.PreferRemote)
					}
					if err != nil {
						appendLog("conflict resolve failed: " + err.Error())
						return
					}
					appendLog("conflict resolve ok")
					if refreshStatus != nil {
						refreshStatus()
					}
				}()
			}
			cancelBtn := widget.NewButton("Cancel", func() { d.Hide() })
			remoteBtn := widget.NewButton("Use remote", func() { resolve(false) })
			localBtn := widget.NewButton("Use local", func() { resolve(true) })
			localBtn.Importance = widget.HighImportance
			d.SetButtons([]fyne.CanvasObject{cancelBtn, remoteBtn, localBtn})
			d.Resize(fyne.NewSize(540, 300))
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
		OnConflict: func(c *syncer.ConflictError, action string) {
			appendLog("conflict (" + action + "): " + c.Error())
			showConflict(c)
		},
	}

	refreshStatus = func() {
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
					displayPath(st.Location.Path),
					trunc(st.LocalHash),
					trunc(st.RemoteHash),
					st.InSync,
				))
			})
		}()
	}
	validateSavesBtn.OnTapped = func() {
		id := gameSelect.Selected
		if id == "" {
			id = "ds3"
		}
		if !remoteConfigured(cfg) {
			appendLogTone("Validate Saves: set a remote destination first", logToneValidate)
			statusLabel.SetText("Status: set a remote destination to get started")
			return
		}
		appendLogTone("Validate Saves: checking "+id+"…", logToneValidate)
		validateSavesBtn.Disable()
		go func() {
			st, err := eng.Status(context.Background(), id)
			fyne.Do(func() {
				validateSavesBtn.Enable()
				if err != nil {
					statusLabel.SetText("Status: " + err.Error())
					appendLogTone("Validate Saves failed: "+err.Error(), logToneValidate)
					return
				}
				statusLabel.SetText(fmt.Sprintf(
					"Status: %s\nPath: %s\nLocal: %s  ·  Cloud: %s  ·  In sync: %v",
					st.Location.Game.ID,
					displayPath(st.Location.Path),
					trunc(st.LocalHash),
					trunc(st.RemoteHash),
					st.InSync,
				))
				if st.InSync {
					appendLogTone(fmt.Sprintf(
						"Validate Saves: %s in sync (local=%s cloud=%s)",
						st.Location.Game.ID, trunc(st.LocalHash), trunc(st.RemoteHash),
					), logToneValidate)
					return
				}
				if st.RemoteHash == "" {
					appendLogTone(fmt.Sprintf(
						"Validate Saves: %s has no cloud save yet (local=%s)",
						st.Location.Game.ID, trunc(st.LocalHash),
					), logToneValidate)
					return
				}
				appendLogTone(fmt.Sprintf(
					"Validate Saves: %s out of sync (local=%s cloud=%s)",
					st.Location.Game.ID, trunc(st.LocalHash), trunc(st.RemoteHash),
				), logToneValidate)
			})
		}()
	}
	gameSelect.OnChanged = func(string) {
		refreshLeftRemote()
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
				showConflict(c)
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

	pullBtn := ttwidget.NewButton("Pull", func() { runAction("pull", eng.Pull) })
	pullBtn.SetToolTip("Download this game’s cloud save onto this PC.")
	pushBtn := ttwidget.NewButton("Push", func() { runAction("push", eng.Push) })
	pushBtn.SetToolTip("Upload this PC’s save for this game to the cloud.")
	watchBtn := ttwidget.NewButton("Start Watcher", nil)
	watchBtn.SetToolTip("Background auto-sync: push after local saves settle (and only when the game isn’t running), and pull from the cloud on an interval.")
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

	actions := container.NewVBox(pullBtn, pushBtn, watchBtn)
	clearActivityBtn := ttwidget.NewButton("Clear Activity", func() {
		mu.Lock()
		defer mu.Unlock()
		logRich.Segments = nil
		logRich.Refresh()
	})
	clearActivityBtn.SetToolTip("Clear the activity log.")
	rightTop := container.NewVBox(
		statusLabel,
		widget.NewSeparator(),
		actions,
	)
	rightBottom := container.NewPadded(clearActivityBtn)
	rightInner := container.NewBorder(rightTop, rightBottom, nil, nil, container.NewPadded(logPane))
	right := container.New(&shrinkFillLayout{}, rightInner)

	split := container.NewHSplit(
		container.NewPadded(left),
		container.NewPadded(right),
	)
	split.SetOffset(0.40)

	refreshLeftRemote()
	refreshLeftLocal()
	w.SetContent(fynetooltip.AddWindowToolTipLayer(split, w.Canvas()))
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

// displayPath shortens long paths so Entry/Label MinSize does not force the window wide.
func displayPath(s string) string {
	const max = 48
	if s == "" || len(s) <= max {
		return s
	}
	return "…" + s[len(s)-(max-1):]
}

// shrinkFillLayout reports a small MinSize so parents (HSplit / window) can shrink,
// while always giving children the full allocated size.
type shrinkFillLayout struct{}

func (shrinkFillLayout) MinSize(_ []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(160, 120)
}

func (shrinkFillLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objects {
		o.Resize(size)
		o.Move(fyne.NewPos(0, 0))
	}
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
			if (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
				return false
			}
		}
	}
	return true
}

func shortGameLabel(id string) string {
	switch id {
	case "ds2":
		return "DS2 SotFS"
	case "ds2-vanilla":
		return "DS2"
	case "ds3":
		return "DS3"
	default:
		return id
	}
}

func saveDirLooksLikeGame(saveDir string, g detect.Game) bool {
	clean := filepath.Clean(saveDir)
	// .../DarkSoulsIII/<steamid> or .../DarkSoulsII/<steamid>
	base := filepath.Base(clean)
	parent := filepath.Base(filepath.Dir(clean))
	if strings.EqualFold(parent, g.FolderName) && looksLikeSteamIDDir(base) {
		return true
	}
	// User may have selected the DarkSoulsIII parent itself
	if strings.EqualFold(base, g.FolderName) {
		return true
	}
	return false
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
