//go:build !nogui

package gui

import (
	"testing"

	"fyne.io/fyne/v2"
)

func TestFontResourcesCached(t *testing.T) {
	a := fontCinzelBold()
	b := fontCinzelBold()
	if a != b {
		t.Fatal("fontCinzelBold should return the same Resource instance")
	}
	c := fontBaskervilleRegular()
	d := fontBaskervilleRegular()
	if c != d {
		t.Fatal("fontBaskervilleRegular should return the same Resource instance")
	}
}

func BenchmarkFontLookupCached(b *testing.B) {
	b.ReportAllocs()
	var r fyne.Resource
	for i := 0; i < b.N; i++ {
		r = fontCinzelBold()
	}
	_ = r
}

func BenchmarkFontLookupAllocating(b *testing.B) {
	// Old behavior: NewStaticResource on every theme.Font() call.
	b.ReportAllocs()
	var r fyne.Resource
	for i := 0; i < b.N; i++ {
		r = fyne.NewStaticResource("Cinzel-Bold.ttf", cinzelBoldTTF)
	}
	_ = r
}
