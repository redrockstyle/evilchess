package gdraw

import (
	"evilchess/ui/gui/gbase"
	"evilchess/ui/gui/ghelper"

	"github.com/hajimehoshi/ebiten/v2"
)

type GUIEditDrawer struct {
}

func NewGUIEditDrawer(ctx *ghelper.GUIGameContext) *GUIEditDrawer {
	return &GUIEditDrawer{}
}

func (pd GUIEditDrawer) Update(ctx *ghelper.GUIGameContext) (SceneType, error) {
	return SceneNotChanged, gbase.ErrExit
}

func (pd GUIEditDrawer) Draw(ctx *ghelper.GUIGameContext, screen *ebiten.Image) {
	return
}
