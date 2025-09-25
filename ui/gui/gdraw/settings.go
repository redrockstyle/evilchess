package gdraw

import (
	"evilchess/ui/gui/gbase"
	"evilchess/ui/gui/gctx"
	"evilchess/ui/gui/ghelper"
	"evilchess/ui/gui/ghelper/glang"
	"fmt"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/sqweek/dialog"
)

type GUISettingsDrawer struct {
	langIndex  int // 0 = en, 1 = ru
	themeIndex int // 0 = light, 1 = dark
	engineMode int // 0 = internal, 1 = uci

	// messagebox
	msg *ghelper.MessageBox

	buttons []*ghelper.Button

	// index of buttons
	btnLangEnIdx     int
	btnLangRuIdx     int
	btnThemeLightIdx int
	btnThemeDarkIdx  int
	btnEngineIntIdx  int
	btnEngineUciIdx  int
	btnBrowseIdx     int
	btnDebugIdx      int
	btnApplyIdx      int
	btnBackIdx       int

	// internal ui state
	prevMouseDown bool
	browseActive  bool

	prevTime time.Time
}

func NewGUISettingsDrawer(ctx *gctx.GUIGameContext) *GUISettingsDrawer {
	sd := &GUISettingsDrawer{
		prevTime: time.Now(),
	}
	if ctx.Theme == gbase.DarkPalette {
		sd.themeIndex = 1
	}
	if ctx.Config.UCIPath != "" && ctx.Config.Engine == "external" {
		if err := IsCorrectEngine(ctx); err == nil {
			sd.engineMode = 1
		} else {
			ctx.Logx.Error("invalid engine config")
			ctx.Config.UCIPath = ""
			ctx.Config.Engine = "internal"
		}
	}
	if ctx.AssetsWorker.Lang().GetLang() == glang.RU {
		sd.langIndex = 1
	}
	sd.makeLayout(ctx)
	sd.refreshButtons(ctx)
	sd.msg = &ghelper.MessageBox{}
	return sd
}

func (sd *GUISettingsDrawer) Update(ctx *gctx.GUIGameContext) (SceneType, error) {
	// mouse handling
	mx, my := ebiten.CursorPosition()
	mouseDown := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	justClicked := mouseDown && !sd.prevMouseDown
	justReleased := !mouseDown && sd.prevMouseDown
	sd.prevMouseDown = mouseDown

	now := time.Now()
	dt := now.Sub(sd.prevTime).Seconds()
	sd.prevTime = now

	// if message box open -> handle clicks on it
	if sd.msg.Open {
		if justClicked {
			// check OK button area in modal coords (we place it centered)
			// Modal geometry: centered rectangle
			bounds := text.BoundString(ctx.AssetsWorker.Fonts().Normal, ctx.AssetsWorker.Lang().T("settings.save.success"))
			sd.msg.CollapseMessageInRect(ctx.Config.WindowW, ctx.Config.WindowH, bounds.Dx(), bounds.Dy())
		}
		// animate open/close
		sd.msg.AnimateMessage()
		return SceneNotChanged, nil
	}

	// HandleInput + UpdateAnim
	for i, b := range sd.buttons {
		clicked := b.HandleInput(mx, my, justClicked, justReleased)
		b.UpdateAnim(dt)
		if clicked {
			switch i {
			case sd.btnLangEnIdx:
				sd.langIndex = 0
				ctx.Config.Lang = "en"
				ctx.AssetsWorker.Lang().SetLang(glang.EN)
				sd.refreshButtons(ctx)
			case sd.btnLangRuIdx:
				sd.langIndex = 1
				ctx.Config.Lang = "ru"
				ctx.AssetsWorker.Lang().SetLang(glang.RU)
				sd.refreshButtons(ctx)
			case sd.btnThemeLightIdx:
				sd.themeIndex = 0
				ctx.Theme = gbase.LightPalette
				sd.refreshButtons(ctx)
			case sd.btnThemeDarkIdx:
				sd.themeIndex = 1
				ctx.Theme = gbase.DarkPalette
				sd.refreshButtons(ctx)
			case sd.btnEngineIntIdx:
				sd.engineMode = 0
				ctx.Config.UCIPath = ""
				sd.refreshButtons(ctx)
			case sd.btnEngineUciIdx:
				sd.engineMode = 1
				sd.refreshButtons(ctx)
			case sd.btnBrowseIdx:
				// open dialog if UCI used
				if sd.engineMode == 1 {
					sd.browseActive = true
					var err error
					ctx.Config.UCIPath, err = dialog.File().Title("Select UCI engine binary").Load()
					if err != nil {
						ctx.Logx.Errorf("error dialog: %v", err)
					}
					if ctx.Config.UCIPath != "" {
						if err = IsCorrectEngine(ctx); err != nil {
							ctx.Config.UCIPath = ""
							ctx.Logx.Error("selected file is not engine")
							sd.msg.ShowMessage(ctx.AssetsWorker.Lang().T("settings.engine.failed"), nil)
						}
					}
					sd.browseActive = false
				}
			case sd.btnDebugIdx:
				// toggle debug in config immediately
				ctx.Config.Debug = !ctx.Config.Debug
				sd.refreshButtons(ctx)
			case sd.btnApplyIdx:
				// save Config
				if sd.engineMode == 1 {
					ctx.Config.Engine = "external"
				} else {
					ctx.Config.Engine = "internal"
					ctx.Config.UCIPath = ""
				}
				ctx.Config.Theme = ctx.Theme.String()
				err := ctx.Config.Save()
				if err != nil {
					sd.msg.ShowMessage(ctx.AssetsWorker.Lang().T("settings.save.failed"), nil)
				} else {
					sd.msg.ShowMessage(ctx.AssetsWorker.Lang().T("settings.save.success"), nil)
				}
			case sd.btnBackIdx:
				return SceneMenu, nil
			}
		}
	}

	// escape -> redo
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return SceneMenu, nil
	}

	return SceneNotChanged, nil
}

func (sd *GUISettingsDrawer) Draw(ctx *gctx.GUIGameContext, screen *ebiten.Image) {
	// background
	screen.Fill(ctx.Theme.Bg)

	// titles
	titlesX := 40
	titlesY := 80
	spacingY := 74
	text.Draw(screen, ctx.AssetsWorker.Lang().T("settings.title"), ctx.AssetsWorker.Fonts().Bold, titlesX, titlesY, ctx.Theme.MenuText)
	text.Draw(screen, ctx.AssetsWorker.Lang().T("settings.lang"), ctx.AssetsWorker.Fonts().Pixel, titlesX+20, titlesY+spacingY, ctx.Theme.MenuText)
	text.Draw(screen, ctx.AssetsWorker.Lang().T("settings.theme"), ctx.AssetsWorker.Fonts().Pixel, titlesX+20, titlesY+2*spacingY, ctx.Theme.MenuText)
	text.Draw(screen, ctx.AssetsWorker.Lang().T("settings.engine"), ctx.AssetsWorker.Fonts().Pixel, titlesX+20, titlesY+3*spacingY, ctx.Theme.MenuText)

	// draw buttions
	for i, b := range sd.buttons {
		// skip
		if i == sd.btnBrowseIdx && sd.engineMode == 0 {
			continue
		}
		// debug up if browse skiped
		if i == sd.btnDebugIdx && sd.engineMode == 0 {
			b.Y = sd.buttons[sd.btnBrowseIdx].Y
		} else if i == sd.btnDebugIdx && sd.engineMode == 1 {
			// debug down if browse used
			b.Y = sd.buttons[sd.btnBrowseIdx].Y + b.H + 18
		}
		b.DrawAnimated(screen, ctx.AssetsWorker.Fonts().PixelLow, ctx.Theme)
	}

	// text browse
	if sd.engineMode == 1 {
		if sd.btnBrowseIdx >= 0 && sd.btnBrowseIdx < len(sd.buttons) {
			b := sd.buttons[sd.btnBrowseIdx]
			display := ctx.AssetsWorker.Lang().T("settings.engine.no_file")
			if ctx.Config.UCIPath != "" {
				display = filepath.Base(ctx.Config.UCIPath)
			}
			if sd.browseActive {
				display = ctx.AssetsWorker.Lang().T("settings.engine.selecting")
			}
			text.Draw(screen, display, ctx.AssetsWorker.Fonts().PixelLow, b.X+12, b.Y+b.H/2+6, ctx.Theme.ButtonText)
		}
	}

	// if message box open -> draw overlay and modal
	if sd.msg.Open || sd.msg.Animating {
		DrawModal(ctx, sd.msg.Scale, sd.msg.Text, screen)
	}

	// debug TPS
	if ctx.Config.Debug {
		ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f", ebiten.ActualTPS()))
	}
}

// create buttons
func (sd *GUISettingsDrawer) makeLayout(ctx *gctx.GUIGameContext) {
	// sizes
	btnW := 220
	btnH := 56
	spacingX := 20 // horizontal
	spacingY := 18 // vertical
	sd.buttons = []*ghelper.Button{}

	startX := 260
	startY := 120

	// Lang: en
	enImg := ghelper.RenderRoundedRect(btnW, btnH, 12, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)
	ben := &ghelper.Button{
		Label: ctx.AssetsWorker.Lang().T("settings.lang.en"),
		X:     startX,
		Y:     startY,
		W:     btnW,
		H:     btnH,
		Image: enImg,
		Scale: 1.0, TargetScale: 1.0, OffsetY: 0, TargetOffsetY: 0, AnimSpeed: 8.0,
	}
	sd.btnLangEnIdx = len(sd.buttons)
	sd.buttons = append(sd.buttons, ben)
	// Lang: ru (to the right)
	ruImg := ghelper.RenderRoundedRect(btnW, btnH, 12, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)
	bru := &ghelper.Button{
		Label: ctx.AssetsWorker.Lang().T("settings.lang.ru"),
		X:     startX + btnW + spacingX,
		Y:     startY,
		W:     btnW,
		H:     btnH,
		Image: ruImg,
		Scale: 1.0, TargetScale: 1.0, OffsetY: 0, TargetOffsetY: 0, AnimSpeed: 8.0,
	}
	sd.btnLangRuIdx = len(sd.buttons)
	sd.buttons = append(sd.buttons, bru)

	// Theme: Light
	themeY := startY + btnH + spacingY
	lightImg := ghelper.RenderRoundedRect(btnW, btnH, 12, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)
	bLight := &ghelper.Button{
		Label: ctx.AssetsWorker.Lang().T("settings.theme.light"),
		X:     startX,
		Y:     themeY,
		W:     btnW,
		H:     btnH,
		Image: lightImg,
		Scale: 1.0, TargetScale: 1.0, OffsetY: 0, TargetOffsetY: 0, AnimSpeed: 8.0,
	}
	sd.btnThemeLightIdx = len(sd.buttons)
	sd.buttons = append(sd.buttons, bLight)
	// Theme: Dark (to the right)
	darkImg := ghelper.RenderRoundedRect(btnW, btnH, 12, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)
	bDark := &ghelper.Button{
		Label: ctx.AssetsWorker.Lang().T("settings.theme.dark"),
		X:     startX + btnW + spacingX,
		Y:     themeY,
		W:     btnW,
		H:     btnH,
		Image: darkImg,
		Scale: 1.0, TargetScale: 1.0, OffsetY: 0, TargetOffsetY: 0, AnimSpeed: 8.0,
	}
	sd.btnThemeDarkIdx = len(sd.buttons)
	sd.buttons = append(sd.buttons, bDark)

	// Engine: internal
	engineY := themeY + btnH + spacingY
	intImg := ghelper.RenderRoundedRect(btnW, btnH, 12, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)
	bInt := &ghelper.Button{
		Label: ctx.AssetsWorker.Lang().T("settings.engine.internal"),
		X:     startX,
		Y:     engineY,
		W:     btnW,
		H:     btnH,
		Image: intImg,
		Scale: 1.0, TargetScale: 1.0, OffsetY: 0, TargetOffsetY: 0, AnimSpeed: 8.0,
	}
	sd.btnEngineIntIdx = len(sd.buttons)
	sd.buttons = append(sd.buttons, bInt)
	// Engine: external
	uciImg := ghelper.RenderRoundedRect(btnW, btnH, 12, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)
	bUci := &ghelper.Button{
		Label: ctx.AssetsWorker.Lang().T("settings.engine.uci"),
		X:     startX + btnW + spacingX,
		Y:     engineY,
		W:     btnW,
		H:     btnH,
		Image: uciImg,
		Scale: 1.0, TargetScale: 1.0, OffsetY: 0, TargetOffsetY: 0, AnimSpeed: 8.0,
	}
	sd.btnEngineUciIdx = len(sd.buttons)
	sd.buttons = append(sd.buttons, bUci)

	// Browse wide field
	// browseW, browseH := 560, 56
	browseY := engineY + btnH + spacingY
	browseW := btnW*2 + spacingX
	browseImg := ghelper.RenderRoundedRect(browseW, btnH, 12, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)
	bBrowse := &ghelper.Button{
		Label: "", X: startX, Y: browseY, W: browseW, H: btnH, Image: browseImg,
		Scale: 1.0, TargetScale: 1.0, OffsetY: 0, TargetOffsetY: 0, AnimSpeed: 8.0,
	}
	sd.btnBrowseIdx = len(sd.buttons)
	sd.buttons = append(sd.buttons, bBrowse)
	// Debug toggle (below browse)
	debugY := browseY + btnH + spacingY
	debugImg := ghelper.RenderRoundedRect(btnW, btnH, 12, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)
	bDebug := &ghelper.Button{
		Label: ctx.AssetsWorker.Lang().T("settings.debug.off"),
		X:     startX, Y: debugY, W: btnW, H: btnH, Image: debugImg,
		Scale: 1.0, TargetScale: 1.0, OffsetY: 0, TargetOffsetY: 0, AnimSpeed: 8.0,
	}
	sd.btnDebugIdx = len(sd.buttons)
	sd.buttons = append(sd.buttons, bDebug)

	// Apply / Back (right-bottom)
	applyW, applyH := 160, 56
	applyX := ctx.Config.WindowW - applyW - 60
	applyY := ctx.Config.WindowH - applyH - 60
	applyImg := ghelper.RenderRoundedRect(applyW, applyH, 12, ctx.Theme.Accent, ctx.Theme.ButtonStroke, 3)
	bApply := &ghelper.Button{
		Label: ctx.AssetsWorker.Lang().T("button.save"),
		X:     applyX, Y: applyY, W: applyW, H: applyH, Image: applyImg,
		Scale: 1.0, TargetScale: 1.0, OffsetY: 0, TargetOffsetY: 0, AnimSpeed: 8.0,
	}
	sd.btnApplyIdx = len(sd.buttons)
	sd.buttons = append(sd.buttons, bApply)

	backX := ctx.Config.WindowW - applyW - 240
	backY := applyY
	backImg := ghelper.RenderRoundedRect(applyW, applyH, 12, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)
	bBack := &ghelper.Button{
		Label: ctx.AssetsWorker.Lang().T("button.back"),
		X:     backX, Y: backY, W: applyW, H: applyH, Image: backImg,
		Scale: 1.0, TargetScale: 1.0, OffsetY: 0, TargetOffsetY: 0, AnimSpeed: 8.0,
	}
	sd.btnBackIdx = len(sd.buttons)
	sd.buttons = append(sd.buttons, bBack)
}

// update accent buttons
func (sd *GUISettingsDrawer) refreshButtons(ctx *gctx.GUIGameContext) {
	for i, b := range sd.buttons {
		fill := ctx.Theme.ButtonFill
		stroke := ctx.Theme.ButtonStroke

		// check accent
		if i == sd.btnLangEnIdx {
			b.Label = ctx.AssetsWorker.Lang().T("settings.lang.en")
			if sd.langIndex == 0 {
				fill = ctx.Theme.Accent
			}
		}
		if i == sd.btnLangRuIdx {
			b.Label = ctx.AssetsWorker.Lang().T("settings.lang.ru")
			if sd.langIndex == 1 {
				fill = ctx.Theme.Accent
			}
		}
		if i == sd.btnThemeLightIdx {
			b.Label = ctx.AssetsWorker.Lang().T("settings.theme.light")
			if sd.themeIndex == 0 {
				fill = ctx.Theme.Accent
			}
		}
		if i == sd.btnThemeDarkIdx {
			b.Label = ctx.AssetsWorker.Lang().T("settings.theme.dark")
			if sd.themeIndex == 1 {
				fill = ctx.Theme.Accent
			}
		}
		if i == sd.btnEngineIntIdx {
			b.Label = ctx.AssetsWorker.Lang().T("settings.engine.internal")
			if sd.engineMode == 0 {
				fill = ctx.Theme.Accent
			}
		}
		if i == sd.btnEngineUciIdx {
			b.Label = ctx.AssetsWorker.Lang().T("settings.engine.uci")
			if sd.engineMode == 1 {
				fill = ctx.Theme.Accent
			}
		}
		if i == sd.btnBrowseIdx && sd.engineMode == 0 {
			fill = ctx.Theme.ButtonFill
		}
		if i == sd.btnBackIdx {
			b.Label = ctx.AssetsWorker.Lang().T("button.back")
		}
		if i == sd.btnApplyIdx {
			b.Label = ctx.AssetsWorker.Lang().T("button.save")
		}
		// debug button label
		if i == sd.btnDebugIdx {
			if ctx.Config.Debug {
				b.Label = ctx.AssetsWorker.Lang().T("settings.debug.on")
				fill = ctx.Theme.Accent
			} else {
				b.Label = ctx.AssetsWorker.Lang().T("settings.debug.off")
			}
		}
		// render background
		b.Image = ghelper.RenderRoundedRect(b.W, b.H, 12, fill, stroke, 3)
	}
}
