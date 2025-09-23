package gui

import (
	"evilchess/src"
	"evilchess/src/logx"
	"evilchess/ui/gui/gbase/gconf"
	"evilchess/ui/gui/gctx"
	"evilchess/ui/gui/gdraw"
	"evilchess/ui/gui/ghelper"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

type GUIProcessing struct {
	sc  gdraw.Scene
	ctx *gctx.GUIGameContext
}

func NewGUI(b *src.GameBuilder, rootDirAssets string, logx logx.Logger) (*GUIProcessing, error) {
	cfg, err := gconf.NewGUIConfig()
	if err != nil {
		return nil, err
	}
	as, err := ghelper.NewGUIAssetsWorker(rootDirAssets, cfg)
	if err != nil {
		return nil, err
	}
	ctx := gctx.NewGUIGameContext(b, as, cfg, logx)
	return &GUIProcessing{sc: gdraw.NewGUIMenuDrawer(ctx), ctx: ctx}, nil
}

func (gp *GUIProcessing) Run() error {
	ebiten.SetWindowIcon([]image.Image{
		gp.ctx.AssetsWorker.IconNative(16),
		gp.ctx.AssetsWorker.IconNative(32),
		gp.ctx.AssetsWorker.IconNative(48),
		gp.ctx.AssetsWorker.IconNative(60),
	})
	ebiten.SetWindowSize(gp.ctx.Config.WindowW, gp.ctx.Config.WindowH)
	ebiten.SetWindowTitle("EvilChess")
	return ebiten.RunGame(gp)
}

func (gp *GUIProcessing) Update() error {
	t, err := gp.sc.Update(gp.ctx)
	if err != nil {
		return err
	}
	gp.sc = t.ToScene(gp.sc, gp.ctx)
	return nil
}

func (gp *GUIProcessing) Draw(screen *ebiten.Image) {
	gp.sc.Draw(gp.ctx, screen)
}

func (gp *GUIProcessing) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return gp.ctx.Config.WindowW, gp.ctx.Config.WindowH
}
