package gdraw

import (
	"evilchess/ui/gui/gctx"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---- Scene ----

type Scene interface {
	Update(ctx *gctx.GUIGameContext) (SceneType, error)
	Draw(ctx *gctx.GUIGameContext, screen *ebiten.Image)
}

type SceneType int

const (
	SceneMenu SceneType = iota
	SceneEditor
	SceneAnalyzer
	SceneSettings
	SceneNotChanged
)

func (t SceneType) ToScene(s Scene, ctx *gctx.GUIGameContext) Scene {
	switch t {
	case SceneMenu:
		s = NewGUIMenuDrawer(ctx)
	case SceneEditor:
	case SceneAnalyzer:
	case SceneSettings:
		s = NewGUISettingsDrawer(ctx)
	case SceneNotChanged:
	default:
	}
	return s
}
