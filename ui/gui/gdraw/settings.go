package gdraw

import (
	"evilchess/ui/gui/gbase"
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
	msg     *ghelper.MessageBox
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

	lastTick time.Time
}

func NewGUISettingsDrawer(ctx *ghelper.GUIGameContext) *GUISettingsDrawer {
	sd := &GUISettingsDrawer{lastTick: time.Now()}
	textBrowse := ctx.AssetsWorker.Lang().T("settings.engine.no_file")
	if ctx.Config.UCIPath != "" && ctx.Config.Engine == "external" {
		if err := IsCorrectEngine(ctx); err != nil {
			ctx.Logx.Error("invalid engine config")
			ctx.Config.UCIPath = ""
			ctx.Config.Engine = "internal"
		}
		textBrowse = filepath.Base(ctx.Config.UCIPath)
	}

	// buttons
	sd.buttons = []*ghelper.Button{}
	btnW := 220
	btnH := 56
	spacingX := 20 // horizontal
	spacingY := 18 // vertical
	startX := 260
	startY := 120

	// lang
	sd.btnLangEnIdx, sd.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("settings.lang.en"), startX, startY, btnW, btnH, sd.buttons)
	sd.btnLangEnIdx, sd.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("settings.lang.ru"), startX+btnW+spacingX, startY, btnW, btnH, sd.buttons)
	// theme
	themeY := startY + btnH + spacingY
	sd.btnThemeLightIdx, sd.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("settings.theme.light"), startX, themeY, btnW, btnH, sd.buttons)
	sd.btnThemeDarkIdx, sd.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("settings.theme.dark"), startX+btnW+spacingX, themeY, btnW, btnH, sd.buttons)
	// engine
	engineY := themeY + btnH + spacingY
	sd.btnEngineIntIdx, sd.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("settings.engine.internal"), startX, engineY, btnW, btnH, sd.buttons)
	sd.btnEngineUciIdx, sd.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("settings.engine.uci"), startX+btnW+spacingX, engineY, btnW, btnH, sd.buttons)
	// browse
	browseY := engineY + btnH + spacingY
	browseW := btnW*2 + spacingX
	sd.btnBrowseIdx, sd.buttons = ghelper.AppendButton(ctx, textBrowse, startX, browseY, browseW, btnH, sd.buttons)
	// debug
	debugY := browseY + btnH + spacingY
	sd.btnDebugIdx, sd.buttons = ghelper.AppendButton(ctx, "", startX, debugY, btnW, btnH, sd.buttons)
	// apply
	applyW, applyH := 160, 56
	applyX := ctx.Config.WindowW - applyW - 60
	applyY := ctx.Config.WindowH - applyH - 60
	sd.btnApplyIdx, sd.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("button.save"), applyX, applyY, applyW, applyH, sd.buttons)
	// back
	backX := ctx.Config.WindowW - applyW - 240
	backY := applyY
	sd.btnBackIdx, sd.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("button.back"), backX, backY, applyW, applyH, sd.buttons)

	sd.refreshButtons(ctx)
	sd.msg = &ghelper.MessageBox{}
	return sd
}

func (sd *GUISettingsDrawer) Update(ctx *ghelper.GUIGameContext) (SceneType, error) {
	// mouse handling
	mx, my := ebiten.CursorPosition()
	mouseDown := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	justClicked := mouseDown && !sd.prevMouseDown
	justReleased := !mouseDown && sd.prevMouseDown
	sd.prevMouseDown = mouseDown

	now := time.Now()
	dt := now.Sub(sd.lastTick).Seconds()
	sd.lastTick = now

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
				ctx.Config.Lang = "en"
				ctx.AssetsWorker.Lang().SetLang(glang.EN)
			case sd.btnLangRuIdx:
				ctx.Config.Lang = "ru"
				ctx.AssetsWorker.Lang().SetLang(glang.RU)
			case sd.btnThemeLightIdx:
				ctx.Theme = gbase.LightPalette
			case sd.btnThemeDarkIdx:
				ctx.Theme = gbase.DarkPalette
			case sd.btnEngineIntIdx:
				ctx.Config.Engine = "internal"
				ctx.Config.UCIPath = ""
			case sd.btnEngineUciIdx:
				ctx.Config.Engine = "external"
			case sd.btnBrowseIdx:
				// open dialog if UCI used
				if ctx.Config.Engine == "external" && !sd.browseActive {
					sd.browseActive = true
					b.Label = ctx.AssetsWorker.Lang().T("settings.engine.selecting")

					go func() {
						var err error
						ctx.Config.UCIPath, err = dialog.File().Title("Select UCI engine binary").Load()
						b.Label = ctx.AssetsWorker.Lang().T("settings.engine.no_file")
						if err != nil {
							ctx.Logx.Errorf("error dialog: %v", err)
						}
						if ctx.Config.UCIPath != "" {
							if err = IsCorrectEngine(ctx); err != nil {
								ctx.Config.UCIPath = ""
								ctx.Logx.Error("selected file is not engine")
								sd.msg.ShowMessage(ctx.AssetsWorker.Lang().T("settings.engine.failed"), nil)
							} else {
								b.Label = filepath.Base(ctx.Config.UCIPath)
							}
						}
						sd.browseActive = false
					}()
				}
				break
			case sd.btnDebugIdx:
				ctx.Config.Debug = !ctx.Config.Debug
			case sd.btnApplyIdx:
				// save Config
				ctx.Config.Theme = ctx.Theme.String()
				err := ctx.Config.Save()
				if err != nil {
					sd.msg.ShowMessage(ctx.AssetsWorker.Lang().T("settings.save.failed"), nil)
				} else {
					sd.msg.ShowMessage(ctx.AssetsWorker.Lang().T("settings.save.success"), nil)
				}
				break
			case sd.btnBackIdx:
				if !sd.browseActive {
					return SceneMenu, nil
				} else {
					sd.msg.ShowMessage(ctx.AssetsWorker.Lang().T("settings.engine.selecting.active"), nil)
					return SceneNotChanged, nil
				}
			}
			sd.refreshButtons(ctx)
		}
	}

	// escape -> redo
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		if !sd.browseActive {
			return SceneMenu, nil
		} else {
			sd.msg.ShowMessage(ctx.AssetsWorker.Lang().T("settings.engine.selecting.active"), nil)
			return SceneNotChanged, nil
		}
	}

	return SceneNotChanged, nil
}

func (sd *GUISettingsDrawer) Draw(ctx *ghelper.GUIGameContext, screen *ebiten.Image) {
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
		if i == sd.btnBrowseIdx && ctx.Config.Engine == "internal" {
			continue
		}
		// debug up if browse skiped
		if i == sd.btnDebugIdx && ctx.Config.Engine == "internal" {
			b.Y = sd.buttons[sd.btnBrowseIdx].Y
		} else if i == sd.btnDebugIdx && ctx.Config.Engine == "external" {
			// debug down if browse used
			b.Y = sd.buttons[sd.btnBrowseIdx].Y + b.H + 18
		}
		b.DrawAnimated(screen, ctx.AssetsWorker.Fonts().PixelLow, ctx.Theme)
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

// update accent buttons
func (sd *GUISettingsDrawer) refreshButtons(ctx *ghelper.GUIGameContext) {
	stroke := ctx.Theme.ButtonStroke
	for i, b := range sd.buttons {
		fill := ctx.Theme.ButtonFill
		switch i {
		case sd.btnLangEnIdx:
			b.Label = ctx.AssetsWorker.Lang().T("settings.lang.en")
			if ctx.AssetsWorker.Lang().GetLang() == glang.EN {
				fill = ctx.Theme.Accent
			}
		case sd.btnLangRuIdx:
			b.Label = ctx.AssetsWorker.Lang().T("settings.lang.ru")
			if ctx.AssetsWorker.Lang().GetLang() == glang.RU {
				fill = ctx.Theme.Accent
			}
		case sd.btnThemeLightIdx:
			b.Label = ctx.AssetsWorker.Lang().T("settings.theme.light")
			if ctx.Theme == gbase.LightPalette {
				fill = ctx.Theme.Accent
			}
		case sd.btnThemeDarkIdx:
			b.Label = ctx.AssetsWorker.Lang().T("settings.theme.dark")
			if ctx.Theme == gbase.DarkPalette {
				fill = ctx.Theme.Accent
			}
		case sd.btnEngineIntIdx:
			b.Label = ctx.AssetsWorker.Lang().T("settings.engine.internal")
			if ctx.Config.Engine == "internal" {
				fill = ctx.Theme.Accent
			}
		case sd.btnEngineUciIdx:
			b.Label = ctx.AssetsWorker.Lang().T("settings.engine.uci")
			if ctx.Config.Engine == "external" {
				fill = ctx.Theme.Accent
			}
		case sd.btnBrowseIdx:
			// if ctx.Config.Engine == "external" {
			// 	fill = ctx.Theme.ButtonFill
			// }
		case sd.btnBackIdx:
			b.Label = ctx.AssetsWorker.Lang().T("button.back")
		case sd.btnApplyIdx:
			b.Label = ctx.AssetsWorker.Lang().T("button.save")
		case sd.btnDebugIdx:
			if ctx.Config.Debug {
				b.Label = ctx.AssetsWorker.Lang().T("settings.debug.on")
				fill = ctx.Theme.Accent
			} else {
				b.Label = ctx.AssetsWorker.Lang().T("settings.debug.off")
			}
		}
		b.Image = ghelper.RenderRoundedRect(b.W, b.H, 12, fill, stroke, 3)
	}
}
