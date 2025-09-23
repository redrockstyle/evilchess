package gdraw

import (
	"evilchess/ui/gui/gbase"
	"evilchess/ui/gui/gctx"
	"evilchess/ui/gui/ghelper"
	"fmt"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/sqweek/dialog"
)

type GUISettingsDrawer struct {
	themeIndex int // 0 = light, 1 = dark
	engineMode int // 0 = internal, 1 = uci
	uciPath    string

	// messagebox
	msg gbase.MessageBox

	buttons []*gbase.Button

	// index of buttons
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
	fileChosenCh  chan string
	browseActive  bool

	prevTime time.Time
}

func NewGUISettingsDrawer(ctx *gctx.GUIGameContext) *GUISettingsDrawer {
	sd := &GUISettingsDrawer{
		fileChosenCh: make(chan string, 1),
		prevTime:     time.Now(),
	}
	if ctx.Theme == gbase.DarkPalette {
		sd.themeIndex = 1
	}
	sd.makeLayout(ctx)
	sd.refreshButtons(ctx)
	return sd
}

func (sd *GUISettingsDrawer) Update(ctx *gctx.GUIGameContext) (SceneType, error) {
	select {
	case p := <-sd.fileChosenCh:
		sd.browseActive = false
		if p != "" {
			sd.uciPath = p
		}
	default:
	}

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
			textW := bounds.Dx()
			textH := bounds.Dy()

			paddingX := 64
			paddingY := 120

			mw := textW + paddingX
			mh := textH + paddingY

			mx := (ctx.ConfigWorker.Config.WindowW - mw) / 2
			my := (ctx.ConfigWorker.Config.WindowH - mh) / 2
			okW, okH := 120, 44
			okX := mx + (mw-okW)/2
			okY := my + mh - 56
			mxPos, myPos := ebiten.CursorPosition()
			if ghelper.PointInRect(mxPos, myPos, okX, okY, okW, okH) {
				// start closing animation
				sd.msg.Opening = false
				sd.msg.Animating = true
				// call close handler after animation ends
				if sd.msg.OnClose == nil {
					sd.msg.OnClose = func() {}
				}
			}
		}
		// animate open/close
		ghelper.AnimateMessage(&sd.msg)
		return SceneNotChanged, nil
	}

	// HandleInput + UpdateAnim
	for i, b := range sd.buttons {
		clicked := b.HandleInput(mx, my, justClicked, justReleased)
		b.UpdateAnim(dt)
		if clicked {
			switch i {
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
				sd.uciPath = ""
				sd.refreshButtons(ctx)
			case sd.btnEngineUciIdx:
				sd.engineMode = 1
				sd.refreshButtons(ctx)
			case sd.btnBrowseIdx:
				// open dialog if UCI used
				if sd.engineMode == 1 {
					sd.browseActive = true
					go func(ch chan<- string) {
						path, err := dialog.File().Title("Select UCI engine binary").Load()
						if err != nil {
							ch <- ""
							return
						}
						ch <- path
					}(sd.fileChosenCh)
				}
			case sd.btnDebugIdx:
				// toggle debug in config immediately
				ctx.ConfigWorker.Config.Debug = !ctx.ConfigWorker.Config.Debug
				sd.refreshButtons(ctx)
			case sd.btnApplyIdx:
				// save ConfigWorker
				if sd.engineMode == 1 {
					ctx.ConfigWorker.Config.Engine = "external"
					ctx.ConfigWorker.Config.UCIPath = sd.uciPath
				} else {
					ctx.ConfigWorker.Config.Engine = "internal"
					ctx.ConfigWorker.Config.UCIPath = ""
				}
				err := ctx.ConfigWorker.Save()
				if err != nil {
					ghelper.ShowMessage(&sd.msg, ctx.AssetsWorker.Lang().T("settings.save.failed"), nil)
				} else {
					ghelper.ShowMessage(&sd.msg, ctx.AssetsWorker.Lang().T("settings.save.success"), nil)

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
	text.Draw(screen, ctx.AssetsWorker.Lang().T("settings.title"), ctx.AssetsWorker.Fonts().Bold, 40, 80, ctx.Theme.MenuText)
	text.Draw(screen, ctx.AssetsWorker.Lang().T("settings.theme"), ctx.AssetsWorker.Fonts().Pixel, 60, 150, ctx.Theme.MenuText)
	text.Draw(screen, ctx.AssetsWorker.Lang().T("settings.engine"), ctx.AssetsWorker.Fonts().Pixel, 60, 230, ctx.Theme.MenuText)

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

	// if message box open -> draw overlay and modal
	if sd.msg.Open || sd.msg.Animating {
		DrawModal(ctx, sd.msg.Scale, sd.msg.Text, screen)
	}

	// text browse
	if sd.engineMode == 1 {
		if sd.btnBrowseIdx >= 0 && sd.btnBrowseIdx < len(sd.buttons) {
			b := sd.buttons[sd.btnBrowseIdx]
			display := ctx.AssetsWorker.Lang().T("settings.engine.no_file")
			if sd.uciPath != "" {
				display = filepath.Base(sd.uciPath)
			}
			if sd.browseActive {
				display = ctx.AssetsWorker.Lang().T("settings.engine.selecting")
			}
			text.Draw(screen, display, ctx.AssetsWorker.Fonts().PixelLow, b.X+12, b.Y+b.H/2+6, ctx.Theme.ButtonText)
		}
	}

	// debug TPS
	if ctx.ConfigWorker.Config.Debug {
		ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f", ebiten.ActualTPS()))
	}
}

// create buttons
func (sd *GUISettingsDrawer) makeLayout(ctx *gctx.GUIGameContext) {
	// sizes
	startX := 260
	startY := 120
	themeW, themeH := 220, 56
	sd.buttons = []*gbase.Button{}

	// Theme: Light
	lightImg := ghelper.RenderRoundedRect(themeW, themeH, 12, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)
	bLight := &gbase.Button{
		Label: ctx.AssetsWorker.Lang().T("settings.theme.light"),
		X:     startX,
		Y:     startY,
		W:     themeW,
		H:     themeH,
		Image: lightImg,
		Scale: 1.0, TargetScale: 1.0, OffsetY: 0, TargetOffsetY: 0, AnimSpeed: 8.0,
	}
	sd.btnThemeLightIdx = len(sd.buttons)
	sd.buttons = append(sd.buttons, bLight)

	// Theme: Dark (to the right)
	darkImg := ghelper.RenderRoundedRect(themeW, themeH, 12, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)
	bDark := &gbase.Button{
		Label: ctx.AssetsWorker.Lang().T("settings.theme.dark"),
		X:     startX + themeW + 20,
		Y:     startY,
		W:     themeW,
		H:     themeH,
		Image: darkImg,
		Scale: 1.0, TargetScale: 1.0, OffsetY: 0, TargetOffsetY: 0, AnimSpeed: 8.0,
	}
	sd.btnThemeDarkIdx = len(sd.buttons)
	sd.buttons = append(sd.buttons, bDark)

	// Engine tiles (same size)
	engineY := startY + 75
	intImg := ghelper.RenderRoundedRect(themeW, themeH, 12, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)
	uciImg := ghelper.RenderRoundedRect(themeW, themeH, 12, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)

	bInt := &gbase.Button{
		Label: ctx.AssetsWorker.Lang().T("settings.engine.internal"),
		X:     startX,
		Y:     engineY,
		W:     themeW,
		H:     themeH,
		Image: intImg,
		Scale: 1.0, TargetScale: 1.0, OffsetY: 0, TargetOffsetY: 0, AnimSpeed: 8.0,
	}
	bUci := &gbase.Button{
		Label: ctx.AssetsWorker.Lang().T("settings.engine.uci"),
		X:     startX + themeW + 20,
		Y:     engineY,
		W:     themeW,
		H:     themeH,
		Image: uciImg,
		Scale: 1.0, TargetScale: 1.0, OffsetY: 0, TargetOffsetY: 0, AnimSpeed: 8.0,
	}
	sd.btnEngineIntIdx = len(sd.buttons)
	sd.buttons = append(sd.buttons, bInt)
	sd.btnEngineUciIdx = len(sd.buttons)
	sd.buttons = append(sd.buttons, bUci)

	// Browse wide field
	browseW, browseH := 560, 56
	browseX := startX
	browseY := engineY + themeH + 18
	browseImg := ghelper.RenderRoundedRect(browseW, browseH, 12, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)
	bBrowse := &gbase.Button{
		Label: "", X: browseX, Y: browseY, W: browseW, H: browseH, Image: browseImg,
		Scale: 1.0, TargetScale: 1.0, OffsetY: 0, TargetOffsetY: 0, AnimSpeed: 8.0,
	}
	sd.btnBrowseIdx = len(sd.buttons)
	sd.buttons = append(sd.buttons, bBrowse)

	// Debug toggle (below browse)
	debugX := startX
	debugY := browseY + browseH + 18
	debugImg := ghelper.RenderRoundedRect(themeW, themeH, 12, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)
	bDebug := &gbase.Button{
		Label: ctx.AssetsWorker.Lang().T("settings.debug.off"),
		X:     debugX, Y: debugY, W: themeW, H: themeH, Image: debugImg,
		Scale: 1.0, TargetScale: 1.0, OffsetY: 0, TargetOffsetY: 0, AnimSpeed: 8.0,
	}
	sd.btnDebugIdx = len(sd.buttons)
	sd.buttons = append(sd.buttons, bDebug)

	// Apply / Back (right-bottom)
	applyW, applyH := 160, 56
	applyX := ctx.ConfigWorker.Config.WindowW - applyW - 60
	applyY := ctx.ConfigWorker.Config.WindowH - applyH - 60
	applyImg := ghelper.RenderRoundedRect(applyW, applyH, 12, ctx.Theme.Accent, ctx.Theme.ButtonStroke, 3)
	bApply := &gbase.Button{
		Label: ctx.AssetsWorker.Lang().T("button.save"),
		X:     applyX, Y: applyY, W: applyW, H: applyH, Image: applyImg,
		Scale: 1.0, TargetScale: 1.0, OffsetY: 0, TargetOffsetY: 0, AnimSpeed: 8.0,
	}
	sd.btnApplyIdx = len(sd.buttons)
	sd.buttons = append(sd.buttons, bApply)

	backX := ctx.ConfigWorker.Config.WindowW - applyW - 240
	backY := applyY
	backImg := ghelper.RenderRoundedRect(applyW, applyH, 12, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)
	bBack := &gbase.Button{
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
		if i == sd.btnThemeLightIdx && sd.themeIndex == 0 {
			fill = ctx.Theme.Accent
		}
		if i == sd.btnThemeDarkIdx && sd.themeIndex == 1 {
			fill = ctx.Theme.Accent
		}
		if i == sd.btnEngineIntIdx && sd.engineMode == 0 {
			fill = ctx.Theme.Accent
		}
		if i == sd.btnEngineUciIdx && sd.engineMode == 1 {
			fill = ctx.Theme.Accent
		}
		if i == sd.btnBrowseIdx && sd.engineMode == 0 {
			fill = ctx.Theme.ButtonFill
		}
		// debug button label
		if i == sd.btnDebugIdx {
			if ctx.ConfigWorker.Config.Debug {
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
