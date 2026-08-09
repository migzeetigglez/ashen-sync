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
	framePad       float32 = 12
	frameOuter     float32 = 2
	frameInnerGap  float32 = 4
	frameInner     float32 = 1
	frameCornerLen float32 = 14
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
	t.TextSize = theme.Size(theme.SizeNameCaptionText) + 1
	t.Alignment = fyne.TextAlignCenter
	return container.NewCenter(t)
}

// framedPanel wraps content in a DS3-style double gold frame on a dark panel.
// If title is non-empty, a gold caption and hairline sit above the content.
func framedPanel(title string, content fyne.CanvasObject) fyne.CanvasObject {
	return newMenuFrame(title, content)
}

// menuFrame paints panel fill, double gold border, and corner ticks behind content.
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

	corners := makeCornerTicks()
	objs := []fyne.CanvasObject{fill, outer, inner}
	objs = append(objs, corners...)
	objs = append(objs, f.content)

	return &menuFrameRenderer{
		f:       f,
		fill:    fill,
		outer:   outer,
		inner:   inner,
		corners: corners,
		objects: objs,
	}
}

func makeCornerTicks() []fyne.CanvasObject {
	ticks := make([]fyne.CanvasObject, 0, 16)
	for i := 0; i < 8; i++ {
		ticks = append(ticks, canvas.NewRectangle(ds3Gold), canvas.NewRectangle(ds3Gold))
	}
	return ticks
}

type menuFrameRenderer struct {
	f       *menuFrame
	fill    *canvas.Rectangle
	outer   *canvas.Rectangle
	inner   *canvas.Rectangle
	corners []fyne.CanvasObject
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

	layoutCornerTicks(r.corners, size, margin+1)

	inset := framePad + frameOuter + frameInnerGap
	r.f.content.Move(fyne.NewPos(inset, inset))
	r.f.content.Resize(fyne.NewSize(size.Width-inset*2, size.Height-inset*2))
}

func layoutCornerTicks(ticks []fyne.CanvasObject, size fyne.Size, m float32) {
	if len(ticks) < 16 {
		return
	}
	thick := float32(2)
	L := frameCornerLen
	w, h := size.Width, size.Height

	place := func(i int, x, y, ww, hh float32) {
		ticks[i].Move(fyne.NewPos(x, y))
		ticks[i].Resize(fyne.NewSize(ww, hh))
	}

	place(0, m, m, L, thick)
	place(1, m, m, thick, L)
	place(2, w-m-L, m, L, thick)
	place(3, w-m-thick, m, thick, L)
	place(4, m, h-m-thick, L, thick)
	place(5, m, h-m-L, thick, L)
	place(6, w-m-L, h-m-thick, L, thick)
	place(7, w-m-thick, h-m-L, thick, L)

	im := m + 6
	il := L - 4
	place(8, im, im, il, thick)
	place(9, im, im, thick, il)
	place(10, w-im-il, im, il, thick)
	place(11, w-im-thick, im, thick, il)
	place(12, im, h-im-thick, il, thick)
	place(13, im, h-im-il, thick, il)
	place(14, w-im-il, h-im-thick, il, thick)
	place(15, w-im-thick, h-im-il, thick, il)
}

func (r *menuFrameRenderer) MinSize() fyne.Size {
	pad := (framePad + frameOuter + frameInnerGap) * 2
	min := r.f.content.MinSize()
	return fyne.NewSize(min.Width+pad, min.Height+pad)
}

func (r *menuFrameRenderer) Objects() []fyne.CanvasObject { return r.objects }

func (r *menuFrameRenderer) Refresh() {
	r.fill.FillColor = ds3Panel
	r.fill.Refresh()
	r.outer.StrokeColor = ds3Gold
	r.outer.Refresh()
	r.inner.StrokeColor = ds3GoldMuted
	r.inner.Refresh()
	canvas.Refresh(r.f)
}
