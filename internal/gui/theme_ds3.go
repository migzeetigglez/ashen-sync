//go:build !nogui

package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// DS3-inspired palette (drawn geometry / theme colors — not game assets).
var (
	ds3Bg          = color.NRGBA{R: 0x0d, G: 0x0c, B: 0x0a, A: 0xff} // near-black charcoal
	ds3Panel       = color.NRGBA{R: 0x1e, G: 0x1a, B: 0x14, A: 0xff} // lifted panel
	ds3Input       = color.NRGBA{R: 0x14, G: 0x12, B: 0x0e, A: 0xff}
	// Tarnished brass — darker/browner than bright antique gold.
	ds3Gold        = color.NRGBA{R: 0x8f, G: 0x75, B: 0x3d, A: 0xff} // #8f753d
	ds3GoldMuted   = color.NRGBA{R: 0x6a, G: 0x58, B: 0x2e, A: 0xff} // idle borders
	ds3GoldDim     = color.NRGBA{R: 0x8f, G: 0x75, B: 0x3d, A: 0x55} // hairlines
	ds3Bronze      = color.NRGBA{R: 0x4a, G: 0x3a, B: 0x1c, A: 0xff} // pressed / hover dark
	ds3Parchment   = color.NRGBA{R: 0xe8, G: 0xdc, B: 0xc8, A: 0xff}
	ds3Ash         = color.NRGBA{R: 0x9a, G: 0x8f, B: 0x7c, A: 0xff}
	ds3AshDim      = color.NRGBA{R: 0x6e, G: 0x64, B: 0x54, A: 0xff}
	ds3Crimson     = color.NRGBA{R: 0x8b, G: 0x1e, B: 0x1e, A: 0xff}
	ds3Overlay     = color.NRGBA{R: 0x0d, G: 0x0c, B: 0x0a, A: 0xee}
	ds3Shadow      = color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x99}
	ds3Button      = color.NRGBA{R: 0x2a, G: 0x24, B: 0x1a, A: 0xff}
	ds3ButtonHover = color.NRGBA{R: 0x3a, G: 0x32, B: 0x22, A: 0xff}
	ds3Ember       = color.NRGBA{R: 0xb8, G: 0x92, B: 0x48, A: 0xff} // validate / emphasis in log
)

// ds3Theme is an app-wide Dark Souls III–inspired Fyne theme.
type ds3Theme struct {
	base fyne.Theme
}

func newDS3Theme() fyne.Theme {
	base := theme.DefaultTheme()
	return ds3Theme{base: base}
}

func (t ds3Theme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	// Always dark — ignore light variant requests.
	_ = v
	switch n {
	case theme.ColorNameBackground:
		return ds3Bg
	case theme.ColorNameOverlayBackground:
		return ds3Overlay
	case theme.ColorNameForeground:
		return ds3Parchment
	case theme.ColorNameDisabled:
		return ds3AshDim
	case theme.ColorNamePlaceHolder:
		return ds3Ash
	// HighImportance buttons use Primary at rest and Hover on mouse-over —
	// rest dark, highlight tarnished gold (swapped from the earlier look).
	case theme.ColorNamePrimary:
		return ds3ButtonHover
	case theme.ColorNameFocus:
		return ds3Gold
	case theme.ColorNameHover:
		return ds3Gold
	case theme.ColorNamePressed:
		return ds3Bronze
	case theme.ColorNameButton:
		return ds3Button
	case theme.ColorNameInputBackground:
		return ds3Input
	case theme.ColorNameInputBorder:
		return ds3GoldMuted
	case theme.ColorNameSeparator:
		return ds3GoldDim
	case theme.ColorNameShadow:
		return ds3Shadow
	case theme.ColorNameError:
		return ds3Crimson
	case theme.ColorNameWarning:
		return ds3Gold
	case theme.ColorNameSuccess:
		return ds3GoldMuted
	case theme.ColorNameHyperlink:
		return ds3Gold
	case theme.ColorNameHeaderBackground:
		return ds3Panel
	case theme.ColorNameMenuBackground:
		return ds3Panel
	case theme.ColorNameScrollBar:
		return ds3GoldMuted
	case theme.ColorNameScrollBarBackground:
		return ds3Input
	case theme.ColorNameSelection:
		return color.NRGBA{R: 0x8f, G: 0x75, B: 0x3d, A: 0x44}
	default:
		return t.base.Color(n, theme.VariantDark)
	}
}

func (t ds3Theme) Font(s fyne.TextStyle) fyne.Resource {
	if s.Monospace {
		return t.base.Font(s)
	}
	if s.Bold || s.Symbol {
		// Display / headings — Cinzel.
		return fontCinzelBold()
	}
	if s.Italic {
		return fontBaskervilleRegular()
	}
	return fontBaskervilleRegular()
}

func (t ds3Theme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return t.base.Icon(n)
}

func (t ds3Theme) Size(n fyne.ThemeSizeName) float32 {
	switch n {
	case theme.SizeNameHeadingText:
		return 26
	case theme.SizeNameSubHeadingText:
		return 16
	case theme.SizeNameText:
		return 13
	case theme.SizeNameCaptionText:
		return 11
	case theme.SizeNamePadding:
		return 6
	case theme.SizeNameInnerPadding:
		return 8
	case theme.SizeNameInputRadius:
		return 2
	case theme.SizeNameSelectionRadius:
		return 2
	default:
		return t.base.Size(n)
	}
}

// readableDisabledTheme keeps disabled entry text as parchment (path fields).
type readableDisabledTheme struct {
	base fyne.Theme
}

func (t readableDisabledTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	if n == theme.ColorNameDisabled {
		return ds3Parchment
	}
	return t.base.Color(n, v)
}
func (t readableDisabledTheme) Font(s fyne.TextStyle) fyne.Resource { return t.base.Font(s) }
func (t readableDisabledTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return t.base.Icon(n)
}
func (t readableDisabledTheme) Size(n fyne.ThemeSizeName) float32 { return t.base.Size(n) }

// activityLogTheme: parchment monospace log (disabled Entry stays readable + selectable).
type activityLogTheme struct {
	base fyne.Theme
}

func (t activityLogTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	switch n {
	case theme.ColorNameForeground, theme.ColorNameDisabled:
		return ds3Parchment
	case theme.ColorNamePrimary:
		return ds3Ember
	case theme.ColorNameInputBackground:
		return ds3Panel
	case theme.ColorNameInputBorder, theme.ColorNameFocus:
		return color.NRGBA{A: 0} // no form-field chrome on the log
	case theme.ColorNameSelection:
		// Strong highlight so drag-select is obvious on the dark log panel.
		return color.NRGBA{R: 0xb8, G: 0x92, B: 0x48, A: 0xcc}
	default:
		return t.base.Color(n, v)
	}
}
func (t activityLogTheme) Font(s fyne.TextStyle) fyne.Resource {
	s.Monospace = true
	return t.base.Font(s)
}
func (t activityLogTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return t.base.Icon(n)
}
func (t activityLogTheme) Size(n fyne.ThemeSizeName) float32 { return t.base.Size(n) }
