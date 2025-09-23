package gdraw

import (
	"evilchess/ui/gui/gbase"
	"evilchess/ui/gui/gctx"

	"github.com/hajimehoshi/ebiten/v2"
)

type GUIAnalyzeDrawer struct {
}

func NewGUIAnalyzeDrawer(ctx *gctx.GUIGameContext) *GUIAnalyzeDrawer {
	return &GUIAnalyzeDrawer{}
}

func (pd GUIAnalyzeDrawer) Update(ctx *gctx.GUIGameContext) (SceneType, error) {
	return SceneNotChanged, gbase.ErrExit
}

func (pd GUIAnalyzeDrawer) Draw(ctx *gctx.GUIGameContext, screen *ebiten.Image) {
	return
}
