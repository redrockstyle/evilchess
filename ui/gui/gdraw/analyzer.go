package gdraw

import (
	"evilchess/ui/gui/gbase"
	"evilchess/ui/gui/ghelper"

	"github.com/hajimehoshi/ebiten/v2"
)

type GUIAnalyzeDrawer struct {
}

func NewGUIAnalyzeDrawer(ctx *ghelper.GUIGameContext) *GUIAnalyzeDrawer {
	return &GUIAnalyzeDrawer{}
}

func (pd GUIAnalyzeDrawer) Update(ctx *ghelper.GUIGameContext) (SceneType, error) {
	return SceneNotChanged, gbase.ErrExit
}

func (pd GUIAnalyzeDrawer) Draw(ctx *ghelper.GUIGameContext, screen *ebiten.Image) {
	return
}
