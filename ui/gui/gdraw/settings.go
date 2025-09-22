package gdraw

import (
	"evilchess/ui/gui/gbase"
	"evilchess/ui/gui/gctx"
	"evilchess/ui/gui/ghelper"
	"image/color"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/sqweek/dialog"
	"golang.org/x/image/font/basicfont"
)

type GUISettingsDrawer struct {
	// elem
	themeIndex int // 0 = light, 1 = dark
	engineMode int // 0 = internal, 1 = uci
	debug      bool
	uciPath    string

	// ui layout
	buttons []*gbase.Button // settings and back

	// internal ui state
	prevMouseDown bool
	fileChosenCh  chan string
	browseActive  bool // show "Selecting..."
}

func NewGUISettingsDrawer(ctx *gctx.GUIGameContext) *GUISettingsDrawer {
	sd := &GUISettingsDrawer{fileChosenCh: make(chan string, 1)}
	sd.makeLayout(ctx)
	return sd
}

func (sd *GUISettingsDrawer) Update(ctx *gctx.GUIGameContext) (SceneType, error) {
	// check file dialog result (non-blocking)
	select {
	case p := <-sd.fileChosenCh:
		sd.browseActive = false
		if p != "" {
			sd.uciPath = p
		}
	default:
	}

	// mouse click detection
	mouseDown := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	justClicked := mouseDown && !sd.prevMouseDown
	sd.prevMouseDown = mouseDown

	// handle clicks
	if justClicked {
		mx, my := ebiten.CursorPosition()

		// Theme boxes
		lightRect := sd.themeRect(ctx, 0)
		darkRect := sd.themeRect(ctx, 1)
		if ghelper.PointInRect(mx, my, lightRect[0], lightRect[1], lightRect[2], lightRect[3]) {
			sd.themeIndex = 0
			// apply theme immediately (visual)
			ctx.Theme = gbase.LightPalette
			//sd.refreshButtons(ctx)
			return SceneNotChanged, nil
		}
		if ghelper.PointInRect(mx, my, darkRect[0], darkRect[1], darkRect[2], darkRect[3]) {
			sd.themeIndex = 1
			ctx.Theme = gbase.DarkPalette
			// s.refreshButtons(ctx)
			return SceneNotChanged, nil
		}

		// Engine mode
		internalRect := sd.engineRect(ctx, 0)
		uciRect := sd.engineRect(ctx, 1)
		if ghelper.PointInRect(mx, my, internalRect[0], internalRect[1], internalRect[2], internalRect[3]) {
			sd.engineMode = 0
			return SceneNotChanged, nil
		}
		if ghelper.PointInRect(mx, my, uciRect[0], uciRect[1], uciRect[2], uciRect[3]) {
			sd.engineMode = 1
			return SceneNotChanged, nil
		}

		// Debug toggle
		debugRect := sd.debugRect(ctx)
		if ghelper.PointInRect(mx, my, debugRect[0], debugRect[1], debugRect[2], debugRect[3]) {
			sd.debug = !sd.debug
			return SceneNotChanged, nil
		}

		// Browse button (only if engineMode==1)
		browseRect := sd.browseRect(ctx)
		if sd.engineMode == 1 && ghelper.PointInRect(mx, my, browseRect[0], browseRect[1], browseRect[2], browseRect[3]) {
			// open native dialog in goroutine
			sd.browseActive = true
			go func(ch chan<- string) {
				// file filter for executables (cross-platform)
				// On Windows: filter exe; on Linux/Mac allow any file
				var path string
				var err error
				// Use dialog library - it blocks; we put it in goroutine
				path, err = dialog.File().Title("Select UCI engine binary").Load()
				if err != nil {
					// send empty to indicate canceled
					ch <- ""
					return
				}
				ch <- path
			}(sd.fileChosenCh)
			return SceneNotChanged, nil
		}

		// Apply
		applyRect := sd.applyRect(ctx)
		if ghelper.PointInRect(mx, my, applyRect[0], applyRect[1], applyRect[2], applyRect[3]) {
			// write into ctx.Settings (assume map[string]interface{} or struct — do best-effort)
			// if ctx.Settings == nil {
			// 	ctx.Settings = make(map[string]interface{})
			// }
			// if s.themeIndex == 1 {
			// 	ctx.Settings["theme"] = "dark"
			// } else {
			// 	ctx.Settings["theme"] = "light"
			// }
			// if s.engineMode == 1 {
			// 	ctx.Settings["engine"] = "uci"
			// } else {
			// 	ctx.Settings["engine"] = "internal"
			// }
			// ctx.Settings["debug"] = s.debug
			// ctx.Settings["ucipath"] = s.uciPath

			// // Try to save settings if ctx.SaveSettings exists (best-effort)
			// if ctx.SaveSettingsFunc != nil {
			// 	_ = ctx.SaveSettingsFunc(ctx.Settings)
			// }
			// // Optionally apply theme via ctx.ApplyTheme
			// if ctx.ApplyThemeFunc != nil {
			// 	if s.themeIndex == 1 {
			// 		ctx.ApplyThemeFunc("dark")
			// 	} else {
			// 		ctx.ApplyThemeFunc("light")
			// 	}
			// }
			// keep on settings screen (or set NextScene if you want to leave)
			return SceneNotChanged, nil
		}

		// Back -> request scene change by setting ctx.NextScene string
		backRect := sd.backRect(ctx)
		if ghelper.PointInRect(mx, my, backRect[0], backRect[1], backRect[2], backRect[3]) {
			// Request to switch to menu: GUIProcessing should detect NextScene and perform replacement
			return SceneMenu, nil
		}
	}

	return SceneNotChanged, nil
}

func (sd *GUISettingsDrawer) Draw(ctx *gctx.GUIGameContext, screen *ebiten.Image) {
	// background
	screen.Fill(ctx.Theme.Bg)

	// title
	title := ctx.AssetsWorker.Lang().T("settings.title")
	text.Draw(screen, title, basicfont.Face7x13, 40, 80, ctx.Theme.MenuText)

	// Theme row
	x := 60
	y := 140
	text.Draw(screen, ctx.AssetsWorker.Lang().T("settings.theme"), basicfont.Face7x13, x, y, ctx.Theme.MenuText)
	// Light
	lrx, lry, lrw, lrh := sd.themeRect(ctx, 0)[0], sd.themeRect(ctx, 0)[1], sd.themeRect(ctx, 0)[2], sd.themeRect(ctx, 0)[3]
	lightFill := ctx.Theme.ButtonFill
	if sd.themeIndex == 0 {
		lightFill = ctx.Theme.Accent
	}
	lightImg := ghelper.RenderRoundedRect(lrw, lrh, 12, lightFill, ctx.Theme.ButtonStroke, 3)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(lrx), float64(lry))
	screen.DrawImage(lightImg, op)
	text.Draw(screen, ctx.AssetsWorker.Lang().T("settings.theme.light"), basicfont.Face7x13, lrx+18, lry+30, ctx.Theme.ButtonText)

	// Dark
	drx, dry, drw, drh := sd.themeRect(ctx, 1)[0], sd.themeRect(ctx, 1)[1], sd.themeRect(ctx, 1)[2], sd.themeRect(ctx, 1)[3]
	darkFill := ctx.Theme.ButtonFill
	if sd.themeIndex == 1 {
		darkFill = ctx.Theme.Accent
	}
	darkImg := ghelper.RenderRoundedRect(drw, drh, 12, darkFill, ctx.Theme.ButtonStroke, 3)
	op2 := &ebiten.DrawImageOptions{}
	op2.GeoM.Translate(float64(drx), float64(dry))
	screen.DrawImage(darkImg, op2)
	text.Draw(screen, ctx.AssetsWorker.Lang().T("settings.theme.dark"), basicfont.Face7x13, drx+18, dry+30, ctx.Theme.MenuText)

	// Engine section
	ex := 60
	ey := 240
	text.Draw(screen, ctx.AssetsWorker.Lang().T("settings.engine"), basicfont.Face7x13, ex, ey, ctx.Theme.MenuText)
	// Internal
	irx, iry, irw, irh := sd.engineRect(ctx, 0)[0], sd.engineRect(ctx, 0)[1], sd.engineRect(ctx, 0)[2], sd.engineRect(ctx, 0)[3]
	intFill := ctx.Theme.ButtonFill
	if sd.engineMode == 0 {
		intFill = ctx.Theme.Accent
	}
	intImg := ghelper.RenderRoundedRect(irw, irh, 10, intFill, ctx.Theme.ButtonStroke, 2)
	op3 := &ebiten.DrawImageOptions{}
	op3.GeoM.Translate(float64(irx), float64(iry))
	screen.DrawImage(intImg, op3)
	text.Draw(screen, ctx.AssetsWorker.Lang().T("settings.engine.internal"), basicfont.Face7x13, irx+18, iry+30, ctx.Theme.ButtonText)

	// UCI
	urx, ury, urw, urh := sd.engineRect(ctx, 1)[0], sd.engineRect(ctx, 1)[1], sd.engineRect(ctx, 1)[2], sd.engineRect(ctx, 1)[3]
	uciFill := ctx.Theme.ButtonFill
	if sd.engineMode == 1 {
		uciFill = ctx.Theme.Accent
	}
	uciImg := ghelper.RenderRoundedRect(urw, urh, 10, uciFill, ctx.Theme.ButtonStroke, 2)
	op4 := &ebiten.DrawImageOptions{}
	op4.GeoM.Translate(float64(urx), float64(ury))
	screen.DrawImage(uciImg, op4)
	text.Draw(screen, ctx.AssetsWorker.Lang().T("settings.engine.uci"), basicfont.Face7x13, urx+18, ury+30, ctx.Theme.ButtonText)

	// Browse field + button
	bx, by, bw, bh := sd.browseRect(ctx)[0], sd.browseRect(ctx)[1], sd.browseRect(ctx)[2], sd.browseRect(ctx)[3]
	valImg := ghelper.RenderRoundedRect(bw, bh, 10, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 2)
	op5 := &ebiten.DrawImageOptions{}
	op5.GeoM.Translate(float64(bx), float64(by))
	screen.DrawImage(valImg, op5)
	display := ctx.AssetsWorker.Lang().T("settings.engine.no_file")
	if sd.uciPath != "" {
		display = filepath.Base(sd.uciPath)
	}
	if sd.browseActive {
		display = ctx.AssetsWorker.Lang().T("settings.engine.selecting")
	}
	text.Draw(screen, display, basicfont.Face7x13, bx+10, by+30, ctx.Theme.ButtonText)

	// Debug toggle
	dx, dy, dw, dh := sd.debugRect(ctx)[0], sd.debugRect(ctx)[1], sd.debugRect(ctx)[2], sd.debugRect(ctx)[3]
	debugImg := ghelper.RenderRoundedRect(dw, dh, 8, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 2)
	op6 := &ebiten.DrawImageOptions{}
	op6.GeoM.Translate(float64(dx), float64(dy))
	screen.DrawImage(debugImg, op6)
	debugText := ctx.AssetsWorker.Lang().T("settings.debug.off")
	if sd.debug {
		debugText = ctx.AssetsWorker.Lang().T("settings.debug.on")
	}
	text.Draw(screen, debugText, basicfont.Face7x13, dx+18, dy+30, ctx.Theme.ButtonText)

	// Apply / Back buttons
	applyX, applyY, applyW, applyH := sd.applyRect(ctx)[0], sd.applyRect(ctx)[1], sd.applyRect(ctx)[2], sd.applyRect(ctx)[3]
	applyImg := ghelper.RenderRoundedRect(applyW, applyH, 12, ctx.Theme.Accent, ctx.Theme.ButtonStroke, 2)
	op7 := &ebiten.DrawImageOptions{}
	op7.GeoM.Translate(float64(applyX), float64(applyY))
	screen.DrawImage(applyImg, op7)
	text.Draw(screen, ctx.AssetsWorker.Lang().T("button.apply"), basicfont.Face7x13, applyX+26, applyY+34, color.White)

	backX, backY, backW, backH := sd.backRect(ctx)[0], sd.backRect(ctx)[1], sd.backRect(ctx)[2], sd.backRect(ctx)[3]
	backImg := ghelper.RenderRoundedRect(backW, backH, 12, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 2)
	op8 := &ebiten.DrawImageOptions{}
	op8.GeoM.Translate(float64(backX), float64(backY))
	screen.DrawImage(backImg, op8)
	text.Draw(screen, ctx.AssetsWorker.Lang().T("button.back"), basicfont.Face7x13, backX+26, backY+34, ctx.Theme.ButtonText)
}

func (sd *GUISettingsDrawer) makeLayout(ctx *gctx.GUIGameContext) {
	btnW, btnH := 320, 64
	gap := 18
	n := 4
	totalH := n*btnH + (n-1)*gap
	startY := (ctx.ConfigWorker.Config.WindowH - totalH) / 2
	cx := ctx.ConfigWorker.Config.WindowW / 2
	sd.buttons = []*gbase.Button{}
	labels := []string{
		ctx.AssetsWorker.Lang().T("settings.theme"),
		ctx.AssetsWorker.Lang().T("settings.lang"),
		ctx.AssetsWorker.Lang().T("settings.engine"),
		ctx.AssetsWorker.Lang().T("settings.debug"),
		ctx.AssetsWorker.Lang().T("buttun.back"),
	}
	for i, lab := range labels {
		x := cx - btnW/2
		y := startY + i*(btnH+gap)
		b := &gbase.Button{
			Label: lab,
			X:     x, Y: y, W: btnW, H: btnH,
		}
		// pre-render button image
		b.Image = ghelper.RenderRoundedRect(
			btnW, btnH, 16,
			ctx.Theme.ButtonFill,
			ctx.Theme.ButtonStroke,
			3,
		)
		sd.buttons = append(sd.buttons, b)
	}
}

// themeRect returns rect for theme option index
func (sd *GUISettingsDrawer) themeRect(ctx *gctx.GUIGameContext, index int) [4]int {
	x := 260
	y := 120
	w := 220
	h := 56
	offset := index * (w + 20)
	return [4]int{x + offset, y, w, h}
}

func (sd *GUISettingsDrawer) engineRect(ctx *gctx.GUIGameContext, index int) [4]int {
	x := 260
	y := 220
	w := 360
	h := 56
	offset := index * (w + 20)
	return [4]int{x + offset, y, w, h}
}

func (sd *GUISettingsDrawer) browseRect(ctx *gctx.GUIGameContext) [4]int {
	x := 260
	y := 300
	w := 560
	h := 56
	return [4]int{x, y, w, h}
}

func (sd *GUISettingsDrawer) debugRect(ctx *gctx.GUIGameContext) [4]int {
	x := 260
	y := 380
	w := 220
	h := 56
	return [4]int{x, y, w, h}
}

func (sd *GUISettingsDrawer) applyRect(ctx *gctx.GUIGameContext) [4]int {
	w := 160
	h := 56
	x := ctx.ConfigWorker.Config.WindowW - w - 60
	y := ctx.ConfigWorker.Config.WindowH - h - 60
	return [4]int{x, y, w, h}
}

func (sd *GUISettingsDrawer) backRect(ctx *gctx.GUIGameContext) [4]int {
	w := 160
	h := 56
	x := ctx.ConfigWorker.Config.WindowW - w - 240
	y := ctx.ConfigWorker.Config.WindowH - h - 60
	return [4]int{x, y, w, h}
}
