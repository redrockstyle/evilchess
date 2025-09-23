package gdraw

import (
	"evilchess/src/engine/uci"
	"evilchess/ui/gui/gctx"
	"evilchess/ui/gui/ghelper"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
)

// ---- Scene ----

type Scene interface {
	Update(ctx *gctx.GUIGameContext) (SceneType, error)
	Draw(ctx *gctx.GUIGameContext, screen *ebiten.Image)
}

type SceneType int

const (
	SceneMenu SceneType = iota
	ScenePlay
	SceneEditor
	SceneAnalyzer
	SceneSettings
	SceneNotChanged
)

func (t SceneType) ToScene(s Scene, ctx *gctx.GUIGameContext) Scene {
	switch t {
	case SceneMenu:
		s = NewGUIMenuDrawer(ctx)
	case ScenePlay:
		s = NewGUIPlayDrawer(ctx)
	case SceneEditor:
		s = NewGUIEditDrawer(ctx)
	case SceneAnalyzer:
		s = NewGUIAnalyzeDrawer(ctx)
	case SceneSettings:
		s = NewGUISettingsDrawer(ctx)
	case SceneNotChanged:
	default:
	}
	return s
}

func DrawModal(ctx *gctx.GUIGameContext, scale float64, message string, screen *ebiten.Image) {

	// dim background
	// draw full-screen translucent rectangle
	overlay := ebiten.NewImage(ctx.Config.WindowW, ctx.Config.WindowH)
	overlay.Fill(ctx.Theme.ModalBg)
	screen.DrawImage(overlay, nil)

	bounds := text.BoundString(ctx.AssetsWorker.Fonts().Normal, message)
	textW := bounds.Dx()
	textH := bounds.Dy()

	paddingX := 64
	paddingY := 120

	mw := textW + paddingX
	mh := textH + paddingY
	// modal rectangle centered with scale
	// mw, mh := 520, 220
	if scale < 0 {
		scale = 0
	}
	if scale > 1 {
		scale = 1
	}
	currW := int(float64(mw) * scale)
	currH := int(float64(mh) * scale)
	if currW < 6 {
		currW = 6
	}
	if currH < 6 {
		currH = 6
	}
	mx := (ctx.Config.WindowW - currW) / 2
	my := (ctx.Config.WindowH - currH) / 2

	// render a rounded rect for modal
	modalImg := ghelper.RenderRoundedRect(currW, currH, 16, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(mx), float64(my))
	screen.DrawImage(modalImg, op)

	// draw message text and OK button (only if fully opened)
	if scale > 0.85 {
		// text centered
		text.Draw(screen, message, ctx.AssetsWorker.Fonts().Normal, mx+32, my+60, ctx.Theme.MenuText)
		// OK button
		okW, okH := 120, 44
		okX := mx + (currW-okW)/2
		okY := my + currH - 56
		okImg := ghelper.RenderRoundedRect(okW, okH, 16, ctx.Theme.Accent, ctx.Theme.ButtonStroke, 3)
		op2 := &ebiten.DrawImageOptions{}
		op2.GeoM.Translate(float64(okX), float64(okY))
		screen.DrawImage(okImg, op2)
		text.Draw(screen, ctx.AssetsWorker.Lang().T("button.ok"), ctx.AssetsWorker.Fonts().PixelLow, okX+36, okY+28, color.White)
	}
}

func IsCorrectEngine(ctx *gctx.GUIGameContext) error {
	e := uci.NewUCIExec(ctx.Logx, ctx.Config.UCIPath)
	if err := e.Init(); err != nil {
		return err
	}
	defer e.Close()
	return nil
}
