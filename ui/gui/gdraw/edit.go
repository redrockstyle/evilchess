package gdraw

import (
	"evilchess/ui/gui/gbase"
	"evilchess/ui/gui/gctx"

	"github.com/hajimehoshi/ebiten/v2"
)

type GUIEditDrawer struct {
}

func NewGUIEditDrawer(ctx *gctx.GUIGameContext) *GUIEditDrawer {
	return &GUIEditDrawer{}
}

func (pd GUIEditDrawer) Update(ctx *gctx.GUIGameContext) (SceneType, error) {
	return SceneNotChanged, gbase.ErrExit
}

func (pd GUIEditDrawer) Draw(ctx *gctx.GUIGameContext, screen *ebiten.Image) {
	return
}
