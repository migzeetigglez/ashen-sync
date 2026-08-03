//go:build !nogui

package gui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/migzeetigglez/ashen-sync/internal/config"
)

// uiScaleOptions are labeled presets for the GUI scale control.
var uiScaleOptions = []struct {
	Label string
	Value float32
}{
	{"100% (Normal)", 1.0},
	{"125%", 1.25},
	{"150% (Recommended on 4K)", 1.5},
	{"175%", 1.75},
	{"200% (Large)", 2.0},
	{"250%", 2.5},
}

func uiScaleLabels() []string {
	out := make([]string, len(uiScaleOptions))
	for i, o := range uiScaleOptions {
		out[i] = o.Label
	}
	return out
}

func labelForScale(scale float32) string {
	for _, o := range uiScaleOptions {
		if almostEqual(o.Value, scale) {
			return o.Label
		}
	}
	return fmt.Sprintf("%.0f%%", scale*100)
}

func scaleForLabel(label string) (float32, bool) {
	for _, o := range uiScaleOptions {
		if o.Label == label {
			return o.Value, true
		}
	}
	return 0, false
}

func almostEqual(a, b float32) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 0.01
}

// applyStartupScale writes Fyne's user scale before the app loads settings.
// We avoid setting FYNE_SCALE so the in-app control can update live via settings.json.
func applyStartupScale(scale float32) {
	_ = writeFyneScale(scale)
}

func clampScale(scale float32) float32 {
	if scale < 0.5 {
		return 0.5
	}
	if scale > 3 {
		return 3
	}
	return scale
}

// writeFyneScale updates ~/.config/fyne/settings.json so the running app
// (and future launches) pick up the scale via Fyne's settings watcher.
func writeFyneScale(scale float32) error {
	schema := fyneapp.SettingsSchema{}
	path := schema.StoragePath()
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &schema)
	}
	schema.Scale = clampScale(scale)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(&schema, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// newUIScaleControl builds a labeled select that persists ui_scale and applies it live.
func newUIScaleControl(cfg *config.Config, w fyne.Window, onSaved func(config.Config)) fyne.CanvasObject {
	sel := ttwidget.NewSelect(uiScaleLabels(), nil)
	sel.SetSelected(labelForScale(cfg.UIScaleOrDefault()))
	sel.SetToolTip("Make the whole UI larger or smaller. You may need to reopen Save Channeler for it to fully apply.")
	sel.OnChanged = func(label string) {
		scale, ok := scaleForLabel(label)
		if !ok {
			return
		}
		cfg.UIScale = scale
		if err := cfg.Save(); err != nil {
			dialog.ShowError(err, w)
			return
		}
		// FYNE_SCALE overrides settings.json — clear it so the watcher can apply.
		_ = os.Unsetenv("FYNE_SCALE")
		if err := writeFyneScale(scale); err != nil {
			dialog.ShowError(err, w)
			return
		}
		onSaved(*cfg)
		dialog.ShowInformation(
			"UI scale updated",
			fmt.Sprintf("Scale set to %s.\n\nIf widgets still look off, close and reopen Save Channeler.", label),
			w,
		)
	}
	return container.NewBorder(nil, nil, widget.NewLabel("UI scale"), nil, sel)
}
