//go:build !nogui

package gui

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
)

// updateBannerHost holds the dismissible update strip and animates its collapse.
type updateBannerHost struct {
	widget.BaseWidget
	inner     fyne.CanvasObject
	height    float32 // animated height; 0 when collapsed
	fullH     float32
	visible   bool
	animating bool
}

func newUpdateBannerHost() *updateBannerHost {
	h := &updateBannerHost{}
	h.ExtendBaseWidget(h)
	h.Hide()
	return h
}

func (h *updateBannerHost) showBanner(inner fyne.CanvasObject) {
	h.inner = inner
	h.visible = true
	h.animating = false
	min := inner.MinSize()
	h.fullH = min.Height + 12
	if h.fullH < 44 {
		h.fullH = 44
	}
	h.height = h.fullH
	h.Show()
	h.Refresh()
}

func (h *updateBannerHost) dismissAnimated(parentRefresh func()) {
	if !h.visible || h.animating {
		return
	}
	h.animating = true
	start := h.height
	if start < 1 {
		start = h.fullH
	}
	if start < 1 {
		start = 44
	}
	anim := fyne.NewAnimation(280*time.Millisecond, func(f float32) {
		fyne.Do(func() {
			h.height = start * (1 - f)
			if h.height < 0 {
				h.height = 0
			}
			h.Refresh()
			if parentRefresh != nil {
				parentRefresh()
			}
			if f >= 1 {
				h.visible = false
				h.animating = false
				h.inner = nil
				h.Hide()
				h.Refresh()
				if parentRefresh != nil {
					parentRefresh()
				}
			}
		})
	})
	anim.Curve = fyne.AnimationEaseInOut
	anim.Start()
}

func (h *updateBannerHost) CreateRenderer() fyne.WidgetRenderer {
	return &updateBannerHostRenderer{h: h}
}

type updateBannerHostRenderer struct {
	h *updateBannerHost
}

func (r *updateBannerHostRenderer) Destroy() {}
func (r *updateBannerHostRenderer) Objects() []fyne.CanvasObject {
	if r.h.inner == nil {
		return nil
	}
	return []fyne.CanvasObject{r.h.inner}
}
func (r *updateBannerHostRenderer) Layout(size fyne.Size) {
	if r.h.inner == nil {
		return
	}
	r.h.inner.Resize(fyne.NewSize(size.Width, size.Height))
	r.h.inner.Move(fyne.NewPos(0, 0))
}
func (r *updateBannerHostRenderer) MinSize() fyne.Size {
	if !r.h.visible || r.h.height <= 0 {
		return fyne.NewSize(0, 0)
	}
	w := float32(200)
	if r.h.inner != nil {
		w = r.h.inner.MinSize().Width
	}
	return fyne.NewSize(w, r.h.height)
}
func (r *updateBannerHostRenderer) Refresh() {
	if r.h.inner != nil {
		r.h.inner.Refresh()
	}
	canvas.Refresh(r.h)
}

// buildUpdateBanner creates the visible strip: message, Update CTA, dismiss (X).
func buildUpdateBanner(version string, onUpdate, onDismiss func()) fyne.CanvasObject {
	msg := canvas.NewText(fmt.Sprintf("Update available — Save Channeler %s", version), ds3Parchment)
	msg.TextStyle = fyne.TextStyle{Bold: true}
	msg.FontSource = fontCinzelBold()
	msg.TextSize = theme.Size(theme.SizeNameText)
	msg.Alignment = fyne.TextAlignLeading

	updateCTA := ttwidget.NewButton("Update", onUpdate)
	updateCTA.Importance = widget.HighImportance
	updateCTA.SetToolTip("Download and install this release.")

	dismiss := ttwidget.NewButton("✕", onDismiss)
	dismiss.Importance = widget.LowImportance
	dismiss.SetToolTip("Dismiss this notice.")

	row := container.NewBorder(nil, nil,
		container.NewPadded(msg),
		container.NewHBox(updateCTA, dismiss),
		nil,
	)

	fill := canvas.NewRectangle(color.NRGBA{R: 0x2a, G: 0x24, B: 0x18, A: 0xff})
	border := canvas.NewRectangle(color.Transparent)
	border.StrokeColor = ds3Gold
	border.StrokeWidth = 1.5

	framed := container.NewStack(fill, border, container.NewPadded(row))
	return container.NewPadded(framed)
}
