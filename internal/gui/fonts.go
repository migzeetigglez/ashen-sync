//go:build !nogui

package gui

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed fonts/Cinzel-Regular.ttf
var cinzelRegularTTF []byte

//go:embed fonts/Cinzel-Bold.ttf
var cinzelBoldTTF []byte

//go:embed fonts/LibreBaskerville-Regular.ttf
var baskervilleRegularTTF []byte

//go:embed fonts/LibreBaskerville-Bold.ttf
var baskervilleBoldTTF []byte

// Cached once — theme.Font() is called constantly during measure/paint.
var (
	resCinzelRegular      = fyne.NewStaticResource("Cinzel-Regular.ttf", cinzelRegularTTF)
	resCinzelBold         = fyne.NewStaticResource("Cinzel-Bold.ttf", cinzelBoldTTF)
	resBaskervilleRegular = fyne.NewStaticResource("LibreBaskerville-Regular.ttf", baskervilleRegularTTF)
	resBaskervilleBold    = fyne.NewStaticResource("LibreBaskerville-Bold.ttf", baskervilleBoldTTF)
)

func fontCinzelRegular() fyne.Resource      { return resCinzelRegular }
func fontCinzelBold() fyne.Resource         { return resCinzelBold }
func fontBaskervilleRegular() fyne.Resource { return resBaskervilleRegular }
func fontBaskervilleBold() fyne.Resource    { return resBaskervilleBold }
