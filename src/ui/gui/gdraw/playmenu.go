package gdraw

import (
	"evilchess/src/ui/gui/ghelper"
	"fmt"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
)

type GUIPlayMenuDrawer struct {
	msg     *ghelper.MessageBox
	buttons []*ghelper.Button

	// index of buttons
	btnUseEngineIdx int
	btnNoEngineIdx  int
	btnUseClockIdx  int
	btnNoClockIdx   int
	btnAsWhiteIdx   int
	btnAsRandomIdx  int
	btnAsBlackIdx   int
	btnStrictIdx    int
	btnStartIdx     int
	btnSaveIdx      int
	btnBackIdx      int

	// clock
	timeWheel *ghelper.NumberWheel
	// level engine
	levelWheel *ghelper.NumberWheel

	// offset point
	startX int
	startY int

	prevMouseDown bool
	lastTick      time.Time
}

func NewGUIPlayMenuDrawer(ctx *ghelper.GUIGameContext) *GUIPlayMenuDrawer {
	spacingX, spacingY, x, y, w, h := 20, 30, 40, 80, 160, 56
	defaultTimer, defaultStrength := 5, 4
	pmd := &GUIPlayMenuDrawer{startX: x, startY: y, lastTick: time.Now()}

	if ctx.Config.Clock > 0 {
		defaultTimer = ctx.Config.Clock
	}

	if ctx.Config.Strength > 0 && ctx.Config.Strength <= 10 {
		defaultStrength = ctx.Config.Strength
	}

	// init level wheel
	pmd.levelWheel = ghelper.NewNumberWheel(
		x+220+w+45+spacingX, y+60,
		w, h*2+spacingY,
		0, 10, 1, defaultStrength, 5,
		ctx.AssetsWorker.Fonts().Pixel, "playmenu.engine.lvl",
	)
	pmd.levelWheel.SetOnChange(func(v int) {
		if ctx.Config.Strength = v; ctx.Config.Strength == 0 {
			pmd.levelWheel.Title = "playmenu.unlimited"
		} else {
			pmd.levelWheel.Title = "playmenu.engine.lvl"
		}
	})
	pmd.levelWheel.AllowWrap(true)
	// init time wheel
	pmd.timeWheel = ghelper.NewNumberWheel(
		x+220+w+45+spacingX, y+60+h*2+spacingY*2,
		w, h*2+spacingY,
		0, 60, 1, defaultTimer, 5,
		ctx.AssetsWorker.Fonts().Pixel, "playmenu.time.min",
	)
	pmd.timeWheel.SetOnChange(func(v int) {
		if ctx.Config.Clock = v; ctx.Config.Clock == 0 {
			pmd.timeWheel.Title = "playmenu.unlimited"
		} else {
			pmd.timeWheel.Title = "playmenu.time.min"
		}
	})
	pmd.timeWheel.AllowWrap(true)

	// buttons
	pmd.buttons = []*ghelper.Button{}
	// engine bottuns
	pmd.btnUseEngineIdx, pmd.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("playmenu.engine.use"), x+220, y+60, w+30, h, pmd.buttons)
	pmd.btnNoEngineIdx, pmd.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("playmenu.engine.empty"), x+220, y+60+h+spacingY, w+30, h, pmd.buttons)
	// clock bottuns
	pmd.btnUseClockIdx, pmd.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("playmenu.time.use"), x+220, y+60+h*2+spacingY*2, w+30, h, pmd.buttons)
	pmd.btnNoClockIdx, pmd.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("playmenu.time.empty"), x+220, y+60+h*3+spacingY*3, w+30, h, pmd.buttons)
	// "play as" bottuns
	pmd.btnAsWhiteIdx, pmd.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("playmenu.as.white"), x+220, y+60+h*4+spacingY*4, w+30, h, pmd.buttons)
	pmd.btnAsRandomIdx, pmd.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("playmenu.as.random"), x+220+w+30+spacingX, y+60+h*4+spacingY*4, w+30, h, pmd.buttons)
	pmd.btnAsBlackIdx, pmd.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("playmenu.as.black"), x+220+(w+30)*2+spacingX*2, y+60+h*4+spacingY*4, w+30, h, pmd.buttons)
	// strict button
	pmd.btnStrictIdx, pmd.buttons = ghelper.AppendButton(ctx, "", x+220, y+60+h*5+spacingY*5, w+30, h, pmd.buttons)
	// navigate bottuns
	pmd.btnStartIdx, pmd.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("playmenu.start"), ctx.Config.WindowW-w-60, ctx.Config.WindowH-h-60, w, h, pmd.buttons)
	pmd.btnSaveIdx, pmd.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("button.save"), ctx.Config.WindowW-240-w, ctx.Config.WindowH-h-60, w, h, pmd.buttons)
	pmd.btnBackIdx, pmd.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("button.back"), ctx.Config.WindowW-420-w, ctx.Config.WindowH-h-60, w, h, pmd.buttons)

	pmd.refreshButtons(ctx, -1)
	pmd.msg = &ghelper.MessageBox{}
	return pmd
}

func (pmd *GUIPlayMenuDrawer) Update(ctx *ghelper.GUIGameContext) (SceneType, error) {
	// for animation
	now := time.Now()
	dt := now.Sub(pmd.lastTick).Seconds()
	pmd.lastTick = now

	// user input checkers
	mx, my := ebiten.CursorPosition()
	mouseDown := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	justPressed := mouseDown && !pmd.prevMouseDown
	justReleased := !mouseDown && pmd.prevMouseDown
	pmd.prevMouseDown = mouseDown

	// if message box open -> handle clicks on it
	if pmd.msg.Open {
		pmd.msg.Update(ctx, mx, my, justReleased)
		pmd.msg.AnimateMessage()
		return SceneNotChanged, nil
	}

	pmd.timeWheel.Update(ctx)
	pmd.levelWheel.Update(ctx)

	for i, b := range pmd.buttons {
		clicked := b.HandleInput(mx, my, justPressed, !mouseDown && b.Pressed == true)
		b.UpdateAnim(dt)
		if clicked {
			switch i {
			case pmd.btnUseEngineIdx:
				ctx.Config.UseEngine = true
			case pmd.btnNoEngineIdx:
				ctx.Config.UseEngine = false
			case pmd.btnUseClockIdx:
				ctx.Config.UseClock = true
			case pmd.btnNoClockIdx:
				ctx.Config.UseClock = false
			case pmd.btnAsWhiteIdx:
				ctx.Config.PlayAs = "white"
			case pmd.btnAsRandomIdx:
				ctx.Config.PlayAs = "random"
			case pmd.btnAsBlackIdx:
				ctx.Config.PlayAs = "black"
			case pmd.btnStrictIdx:
				ctx.Config.Training = !ctx.Config.Training
			case pmd.btnStartIdx:
				ctx.IsReady = false
				return ScenePlay, nil
			case pmd.btnSaveIdx:
				if err := ctx.Config.Save(); err != nil {
					ctx.Logx.Errorf("confg save failed: %v", err)
					pmd.msg.ShowMessage(ctx.AssetsWorker.Lang().T("settings.save.failed"), nil)
				} else {
					pmd.msg.ShowMessage(ctx.AssetsWorker.Lang().T("settings.save.success"), nil)
				}
				break
			case pmd.btnBackIdx:
				return SceneMenu, nil
			}
			pmd.refreshButtons(ctx, i)
			break
		}
	}

	// escape -> redo
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return SceneMenu, nil
	}

	return SceneNotChanged, nil
}

func (pmd *GUIPlayMenuDrawer) Draw(ctx *ghelper.GUIGameContext, screen *ebiten.Image) {
	// background
	screen.Fill(ctx.Theme.Bg)

	// titles
	spacingY := 150
	text.Draw(screen, ctx.AssetsWorker.Lang().T("playmenu.title"), ctx.AssetsWorker.Fonts().Bold, pmd.startX, pmd.startY, ctx.Theme.MenuText)
	text.Draw(screen, ctx.AssetsWorker.Lang().T("playmenu.engine"), ctx.AssetsWorker.Fonts().Pixel, pmd.startX+20, pmd.startY+spacingY-10, ctx.Theme.MenuText)
	text.Draw(screen, ctx.AssetsWorker.Lang().T("playmenu.time"), ctx.AssetsWorker.Fonts().Pixel, pmd.startX+20, pmd.startY+spacingY*2+10, ctx.Theme.MenuText)
	text.Draw(screen, ctx.AssetsWorker.Lang().T("playmenu.as"), ctx.AssetsWorker.Fonts().Pixel, pmd.startX+20, pmd.startY+spacingY*3-10, ctx.Theme.MenuText)
	text.Draw(screen, ctx.AssetsWorker.Lang().T("playmenu.training"), ctx.AssetsWorker.Fonts().Pixel, pmd.startX+20, pmd.startY+spacingY*3+spacingY/2, ctx.Theme.MenuText)

	pmd.timeWheel.Draw(ctx, screen)
	pmd.levelWheel.Draw(ctx, screen)

	// draw UI buttons (animated via b.DrawAnimated)
	for _, b := range pmd.buttons {
		b.DrawAnimated(screen, ctx.AssetsWorker.Fonts().PixelLow, ctx.Theme)
	}

	// if message box open -> draw overlay and modal
	// if pmd.msg.Open || pmd.msg.Animating {
	// 	DrawModal(ctx, pmd.msg.Scale, pmd.msg.Text, screen)
	// }
	pmd.msg.Draw(ctx, screen)


	if ctx.Config.Debug {
		ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f", ebiten.ActualTPS()))
	}
}

func (pmd *GUIPlayMenuDrawer) refreshButtons(ctx *ghelper.GUIGameContext, btnIdx int) {
	refreshAccent := func(b *ghelper.Button) *ebiten.Image {
		return ghelper.RenderRoundedRect(b.W, b.H, 12, ctx.Theme.Accent, ctx.Theme.ButtonStroke, 3)
	}
	refreshDefault := func(b *ghelper.Button) *ebiten.Image {
		return ghelper.RenderRoundedRect(b.W, b.H, 12, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)
	}

	selector := func(b *ghelper.Button, i int) {
		switch i {
		case pmd.btnUseEngineIdx, pmd.btnNoEngineIdx:
			if ctx.Config.UseEngine {
				b.Image = refreshAccent(b)
				pmd.buttons[pmd.btnNoEngineIdx].Image = refreshDefault(b)
			} else {
				b.Image = refreshAccent(b)
				pmd.buttons[pmd.btnUseEngineIdx].Image = refreshDefault(b)
			}
		case pmd.btnUseClockIdx, pmd.btnNoClockIdx:
			if ctx.Config.UseClock {
				b.Image = refreshAccent(b)
				pmd.buttons[pmd.btnNoClockIdx].Image = refreshDefault(b)
			} else {
				b.Image = refreshAccent(b)
				pmd.buttons[pmd.btnUseClockIdx].Image = refreshDefault(b)
			}
		case pmd.btnAsWhiteIdx, pmd.btnAsRandomIdx, pmd.btnAsBlackIdx:
			if ctx.Config.PlayAs == "white" {
				b.Image = refreshAccent(b)
				pmd.buttons[pmd.btnAsRandomIdx].Image = refreshDefault(b)
				pmd.buttons[pmd.btnAsBlackIdx].Image = refreshDefault(b)
			} else if ctx.Config.PlayAs == "random" {
				b.Image = refreshAccent(b)
				pmd.buttons[pmd.btnAsWhiteIdx].Image = refreshDefault(b)
				pmd.buttons[pmd.btnAsBlackIdx].Image = refreshDefault(b)
			} else if ctx.Config.PlayAs == "black" {
				b.Image = refreshAccent(b)
				pmd.buttons[pmd.btnAsWhiteIdx].Image = refreshDefault(b)
				pmd.buttons[pmd.btnAsRandomIdx].Image = refreshDefault(b)
			}
		case pmd.btnStrictIdx:
			if ctx.Config.Training {
				b.Label = ctx.AssetsWorker.Lang().T("playmenu.training.on")
				b.Image = refreshAccent(b)
			} else {
				b.Label = ctx.AssetsWorker.Lang().T("playmenu.training.off")
				b.Image = refreshDefault(b)
			}
		}
	}

	if btnIdx == -1 {
		for i, b := range pmd.buttons {
			selector(b, i)
		}
	} else {
		selector(pmd.buttons[btnIdx], btnIdx)
	}
}
