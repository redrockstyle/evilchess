package ghelper

import (
	"evilchess/src/base"
	"evilchess/ui/gui/gbase/gconf"
	"evilchess/ui/gui/ghelper/gimages"
	"evilchess/ui/gui/ghelper/glang"

	"github.com/hajimehoshi/ebiten/v2"
)

type GUIAssetsWorker struct {
	pieceImages map[base.Piece]*ebiten.Image
	lang        *glang.GUILangWorker
}

func NewGUIAssetsWorker(rootDirAssets string, cfg *gconf.GUIConfigWorker) (*GUIAssetsWorker, error) {
	imgs, err := gimages.LoadImageAssets("assets/images")
	if err != nil {
		return nil, err
	}
	l, err := glang.NewGUILangWorker("assets/lang", cfg)
	if err != nil {
		return nil, err
	}
	return &GUIAssetsWorker{pieceImages: imgs, lang: l}, nil
}

func (aw *GUIAssetsWorker) Piece(p base.Piece) *ebiten.Image {
	return aw.pieceImages[p]
}
func (aw *GUIAssetsWorker) Lang() *glang.GUILangWorker {
	return aw.lang
}
