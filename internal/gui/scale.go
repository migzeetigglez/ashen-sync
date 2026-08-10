//go:build !nogui

package gui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

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

// applyStartupScale sets Fyne's user scale before the app loads settings.
func applyStartupScale(scale float32) {
	_ = applyLiveScale(scale)
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

func scaleEnvValue(scale float32) string {
	return strconv.FormatFloat(float64(clampScale(scale)), 'f', -1, 32)
}

// writeFyneScale updates ~/.config/fyne/settings.json for future launches.
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

// applyLiveScale persists Fyne settings and forces an immediate canvas rescale.
// Fyne has no public SetScale; the glfw driver reads FYNE_SCALE on reloadScale(),
// which is triggered by Settings change listeners (e.g. SetTheme).
func applyLiveScale(scale float32) error {
	scale = clampScale(scale)
	if err := writeFyneScale(scale); err != nil {
		return err
	}
	_ = os.Setenv("FYNE_SCALE", scaleEnvValue(scale))
	if a := fyne.CurrentApp(); a != nil {
		if th := a.Settings().Theme(); th != nil {
			a.Settings().SetTheme(th)
		}
		if c := a.Driver().AllWindows(); len(c) > 0 {
			for _, w := range c {
				if content := w.Content(); content != nil {
					content.Refresh()
				}
			}
		}
	}
	return nil
}

// newUIScaleControl builds a labeled select that persists ui_scale and applies it live.
func newUIScaleControl(cfg *config.Config, w fyne.Window, onSaved func(config.Config)) fyne.CanvasObject {
	sel := ttwidget.NewSelect(uiScaleLabels(), nil)
	sel.SetSelected(labelForScale(cfg.UIScaleOrDefault()))
	sel.SetToolTip("Make the whole UI larger or smaller. Changes apply immediately.")
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
		if err := applyLiveScale(scale); err != nil {
			dialog.ShowError(err, w)
			return
		}
		onSaved(*cfg)
	}
	return container.NewBorder(nil, nil, widget.NewLabel("UI scale"), nil, sel)
}
