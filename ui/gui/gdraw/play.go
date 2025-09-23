package gdraw

import (
	"evilchess/ui/gui/gbase"
	"evilchess/ui/gui/gctx"

	"github.com/hajimehoshi/ebiten/v2"
)

type GUIPlayDrawer struct {
}

func NewGUIPlayDrawer(ctx *gctx.GUIGameContext) *GUIPlayDrawer {
	return &GUIPlayDrawer{}
}

func (pd GUIPlayDrawer) Update(ctx *gctx.GUIGameContext) (SceneType, error) {
	return SceneNotChanged, gbase.ErrExit
}

func (pd GUIPlayDrawer) Draw(ctx *gctx.GUIGameContext, screen *ebiten.Image) {
	return
}
