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

func fontCinzelRegular() fyne.Resource {
	return fyne.NewStaticResource("Cinzel-Regular.ttf", cinzelRegularTTF)
}

func fontCinzelBold() fyne.Resource {
	return fyne.NewStaticResource("Cinzel-Bold.ttf", cinzelBoldTTF)
}

func fontBaskervilleRegular() fyne.Resource {
	return fyne.NewStaticResource("LibreBaskerville-Regular.ttf", baskervilleRegularTTF)
}

func fontBaskervilleBold() fyne.Resource {
	return fyne.NewStaticResource("LibreBaskerville-Bold.ttf", baskervilleBoldTTF)
}
