package ghelper

import (
	"evilchess/src/chesslib/base"
	"evilchess/src/ui/gui/ghelper/gfont"
	"evilchess/src/ui/gui/ghelper/gimages"
	"evilchess/src/ui/gui/ghelper/glang"
	"evilchess/src/ui/gui/gbase/gconf"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

type GUIAssetsWorker struct {
	fonts       *gfont.Fonts
	pieceImages map[base.Piece]*ebiten.Image
	iconImages  map[int]*ebiten.Image
	icons       map[int]image.Image
	lang        *glang.GUILangWorker
}

func NewGUIAssetsWorker(rootDirAssets string, cfg *gconf.Config) (*GUIAssetsWorker, error) {
	pieceImages, err := gimages.LoadImagePieceAssets("assets/images")
	if err != nil {
		return nil, err
	}
	iconImages, err := gimages.LoadImageIconAssets("assets/images")
	if err != nil {
		return nil, err
	}
	icons, err := gimages.LoadIconAssets("assets/images")
	if err != nil {
		return nil, err
	}
	l, err := glang.NewGUILangWorker("assets/lang", cfg)
	if err != nil {
		return nil, err
	}
	f, err := gfont.LoadFonts("assets/font")
	if err != nil {
		return nil, err
	}
	return &GUIAssetsWorker{fonts: f, pieceImages: pieceImages, iconImages: iconImages, icons: icons, lang: l}, nil
}

func (aw *GUIAssetsWorker) Piece(p base.Piece) *ebiten.Image {
	return aw.pieceImages[p]
}
func (aw *GUIAssetsWorker) Lang() *glang.GUILangWorker {
	return aw.lang
}
func (aw *GUIAssetsWorker) Icon(x int) *ebiten.Image {
	return aw.iconImages[x]
}
func (aw *GUIAssetsWorker) IconNative(x int) image.Image {
	return aw.icons[x]
}
func (aw *GUIAssetsWorker) Fonts() *gfont.Fonts {
	return aw.fonts
}
