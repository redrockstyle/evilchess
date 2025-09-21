package gui

import (
	"evilchess/src"
	"evilchess/src/logx"
	"evilchess/ui/gui/ddraw"
	"evilchess/ui/gui/ddraw/dmenu"
	"evilchess/ui/gui/tools/lang"

	"github.com/hajimehoshi/ebiten/v2"
)

type GUIProcessing struct {
	current ddraw.Scene
	ctx     ddraw.GameContext
}

func NewGUI(b *src.GameBuilder, logx logx.Logger) (*GUIProcessing, error) {
	helper, err := ddraw.NewGUIHelperDraw()
	if err != nil {
		return nil, err
	}
	lw, err := lang.NewGUILangWorker("assets/lang")
	if err != nil {
		return nil, err
	}
	ctx := ddraw.GameContext{
		Builder: b,
		Helper:  helper,
		Lang:    lw,
		Window:  struct{ W, H int }{ddraw.WindowW, ddraw.WindowH},
		Logx:    logx,
	}
	return &GUIProcessing{
		current: dmenu.NewGUIMenuDrawer(&ctx),
		ctx:     ctx,
	}, nil
}

func (gp *GUIProcessing) Run() error {
	ebiten.SetWindowSize(gp.ctx.Window.W, gp.ctx.Window.H)
	ebiten.SetWindowTitle("EvilChess")
	return ebiten.RunGame(gp)
}

func (gp *GUIProcessing) Update() error {
	next, err := gp.current.Update(&gp.ctx)
	if err != nil {
		return err
	}
	if next != nil {
		gp.current = next
	}
	return nil
}

func (gp *GUIProcessing) Draw(screen *ebiten.Image) {
	gp.current.Draw(&gp.ctx, screen)
}

func (gp *GUIProcessing) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return gp.ctx.Window.W, gp.ctx.Window.H
}
