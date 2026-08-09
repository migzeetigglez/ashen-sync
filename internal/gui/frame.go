//go:build !nogui

package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	framePad      float32 = 12
	frameOuter    float32 = 2
	frameInnerGap float32 = 4
	frameInner    float32 = 1
)

// goldHairline is a thin antique-gold rule used between menu sections.
func goldHairline() fyne.CanvasObject {
	r := canvas.NewRectangle(ds3GoldDim)
	r.SetMinSize(fyne.NewSize(1, 1))
	return container.New(&hairlineLayout{height: 1}, r)
}

type hairlineLayout struct {
	height float32
}

func (h *hairlineLayout) MinSize(_ []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(1, h.height+8)
}

func (h *hairlineLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	y := (size.Height - h.height) / 2
	objects[0].Move(fyne.NewPos(0, y))
	objects[0].Resize(fyne.NewSize(size.Width, h.height))
}

// sectionCaption renders an uppercase menu subsection label in gold.
func sectionCaption(text string) fyne.CanvasObject {
	t := canvas.NewText(text, ds3Gold)
	t.TextStyle = fyne.TextStyle{Bold: true}
	t.FontSource = fontCinzelBold()
	t.TextSize = theme.Size(theme.SizeNameCaptionText) + 1
	t.Alignment = fyne.TextAlignCenter
	return container.NewCenter(t)
}

// framedPanel wraps content in a DS3-style double gold frame on a dark panel.
// If title is non-empty, a gold caption and hairline sit above the content.
func framedPanel(title string, content fyne.CanvasObject) fyne.CanvasObject {
	return newMenuFrame(title, content)
}

// menuFrame paints panel fill and a double gold border behind content.
// Corner-tick ornaments were removed — 16 extra rectangles per frame made
// every layout/refresh noticeably heavier with multiple panels on screen.
type menuFrame struct {
	widget.BaseWidget
	content fyne.CanvasObject
}

func newMenuFrame(title string, content fyne.CanvasObject) *menuFrame {
	inner := content
	if title != "" {
		inner = container.NewBorder(
			container.NewVBox(sectionCaption(title), goldHairline()),
			nil, nil, nil,
			content,
		)
	}
	f := &menuFrame{content: container.NewPadded(inner)}
	f.ExtendBaseWidget(f)
	return f
}

func (f *menuFrame) CreateRenderer() fyne.WidgetRenderer {
	fill := canvas.NewRectangle(ds3Panel)
	outer := canvas.NewRectangle(color.Transparent)
	outer.StrokeColor = ds3Gold
	outer.StrokeWidth = frameOuter
	inner := canvas.NewRectangle(color.Transparent)
	inner.StrokeColor = ds3GoldMuted
	inner.StrokeWidth = frameInner

	return &menuFrameRenderer{
		f:       f,
		fill:    fill,
		outer:   outer,
		inner:   inner,
		objects: []fyne.CanvasObject{fill, outer, inner, f.content},
	}
}

type menuFrameRenderer struct {
	f       *menuFrame
	fill    *canvas.Rectangle
	outer   *canvas.Rectangle
	inner   *canvas.Rectangle
	objects []fyne.CanvasObject
}

func (r *menuFrameRenderer) Destroy() {}

func (r *menuFrameRenderer) Layout(size fyne.Size) {
	r.fill.Resize(size)
	r.fill.Move(fyne.NewPos(0, 0))

	margin := float32(3)
	r.outer.Move(fyne.NewPos(margin, margin))
	r.outer.Resize(fyne.NewSize(size.Width-margin*2, size.Height-margin*2))

	im := margin + frameOuter + frameInnerGap
	r.inner.Move(fyne.NewPos(im, im))
	r.inner.Resize(fyne.NewSize(size.Width-im*2, size.Height-im*2))

	inset := framePad + frameOuter + frameInnerGap
	r.f.content.Move(fyne.NewPos(inset, inset))
	r.f.content.Resize(fyne.NewSize(size.Width-inset*2, size.Height-inset*2))
}

func (r *menuFrameRenderer) MinSize() fyne.Size {
	pad := (framePad + frameOuter + frameInnerGap) * 2
	min := r.f.content.MinSize()
	return fyne.NewSize(min.Width+pad, min.Height+pad)
}

func (r *menuFrameRenderer) Objects() []fyne.CanvasObject { return r.objects }

func (r *menuFrameRenderer) Refresh() {
	// Colors are constants — avoid cascading Refresh on every parent paint.
	canvas.Refresh(r.f)
}