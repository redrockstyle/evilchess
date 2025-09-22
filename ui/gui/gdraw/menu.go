package gdraw

import (
	"evilchess/ui/gui/gbase"
	"evilchess/ui/gui/gctx"
	"evilchess/ui/gui/ghelper"
	"evilchess/ui/gui/ghelper/glang"
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"
)

type GUIMenuDrawer struct {
	buttons []*gbase.Button

	// messagebox
	msg gbase.MessageBox

	// language selector square bottom-left
	langBoxX, langBoxY, langBoxS int

	// click tracking
	prevMouseDown bool
}

func NewGUIMenuDrawer(ctx *gctx.GUIGameContext) *GUIMenuDrawer {
	md := &GUIMenuDrawer{}
	md.makeLayout(ctx)
	return md
}

func (md *GUIMenuDrawer) Update(ctx *gctx.GUIGameContext) (SceneType, error) {
	// keyboard: toggle palette for demo
	if ebiten.IsKeyPressed(ebiten.KeyTab) {
		if ctx.Theme == gbase.LightPalette {
			ctx.Theme = gbase.DarkPalette
		} else {
			ctx.Theme = gbase.LightPalette
		}
		md.refreshButtons(ctx)
	}
	// check mouse just clicked
	mouseDown := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	justClicked := mouseDown && !md.prevMouseDown
	md.prevMouseDown = mouseDown

	// if message box open -> handle clicks on it
	if md.msg.Open {
		if justClicked {
			// check OK button area in modal coords (we place it centered)
			// Modal geometry: centered rectangle
			mw, mh := 520, 220
			mx := (ctx.ConfigWorker.Config.WindowW - mw) / 2
			my := (ctx.ConfigWorker.Config.WindowH - mh) / 2
			okW, okH := 120, 44
			okX := mx + (mw-okW)/2
			okY := my + mh - 56
			mxPos, myPos := ebiten.CursorPosition()
			if ghelper.PointInRect(mxPos, myPos, okX, okY, okW, okH) {
				// start closing animation
				md.msg.Opening = false
				md.msg.Animating = true
				// call close handler after animation ends
				if md.msg.OnClose == nil {
					md.msg.OnClose = func() {}
				}
			}
		}
		// animate open/close
		ghelper.AnimateMessage(&md.msg)
		return SceneNotChanged, nil
	}

	// handle clicks on menu buttons
	if justClicked {
		mx, my := ebiten.CursorPosition()
		for i, b := range md.buttons {
			if ghelper.PointInRect(mx, my, b.X, b.Y, b.W, b.H) {
				ctx.Logx.Infof("%s (%d) clicked", b.Label, i)
				switch i {
				case 0: // menu.play
				case 1: // menu.editor
				case 2: // menu.settings

					return SceneSettings, nil
				case 3: // menu.exit
					return SceneNotChanged, gbase.ErrExit
				}

				// Demo: open messagebox with text of clicked button
				// ctx.Helper.ShowMessage(&md.msg, fmt.Sprintf("%s clicked", b.Label), func() { log.Printf("closed message for #%d", i) })
				return SceneNotChanged, nil
			}
		}
		// language box click
		if ghelper.PointInRect(mx, my, md.langBoxX, md.langBoxY, md.langBoxS, md.langBoxS) {
			if ctx.AssetsWorker.Lang().GetLang() == glang.EN {
				ctx.AssetsWorker.Lang().SetLang(glang.RU)
			} else {
				ctx.AssetsWorker.Lang().SetLang(glang.EN)
			}
			md.refreshButtons(ctx)
			return SceneNotChanged, nil
		}
	}

	return SceneNotChanged, nil
}

func (md *GUIMenuDrawer) Draw(ctx *gctx.GUIGameContext, screen *ebiten.Image) {
	// clear background
	screen.Fill(ctx.Theme.Bg)

	// draw centered menu buttons
	for _, b := range md.buttons {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(b.X), float64(b.Y))
		screen.DrawImage(b.Image, op)

		// button label (text) centered
		textX := b.X + b.W/2 - len(b.Label)*4
		textY := b.Y + b.H/2 + 6
		text.Draw(screen, b.Label, basicfont.Face7x13, textX, textY, ctx.Theme.ButtonText)

		// outline (for strong contour) — draw thin stroke rectangle slightly larger
		// ctx.Helper.EbitenutilDrawRectStroke(screen, float64(b.X), float64(b.Y), float64(b.W), float64(b.H), 2, md.pal.ButtonStroke)
	}

	// language box bottom-left (square)
	// square background
	langImg := ghelper.RenderRoundedRect(md.langBoxS, md.langBoxS, 8, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 2)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(md.langBoxX), float64(md.langBoxY))
	screen.DrawImage(langImg, op)
	text.Draw(screen, ctx.AssetsWorker.Lang().T("lang.type"), basicfont.Face7x13, md.langBoxX+16, md.langBoxY+md.langBoxS/2+4, ctx.Theme.ButtonText)
	// small label
	text.Draw(screen, ctx.AssetsWorker.Lang().T("lang.title"), basicfont.Face7x13, md.langBoxX+6, md.langBoxY-6, ctx.Theme.MenuText)

	// version on bottom-right
	ver := ctx.AssetsWorker.Lang().T("version")
	text.Draw(screen, ver, basicfont.Face7x13, ctx.ConfigWorker.Config.WindowW-160, ctx.ConfigWorker.Config.WindowH-24, ctx.Theme.MenuText)

	// if message box open -> draw overlay and modal
	if md.msg.Open || md.msg.Animating {
		// dim background
		// draw full-screen translucent rectangle
		overlay := ebiten.NewImage(ctx.ConfigWorker.Config.WindowW, ctx.ConfigWorker.Config.WindowH)
		overlay.Fill(ctx.Theme.ModalBg)
		screen.DrawImage(overlay, nil)

		// modal rectangle centered with scale
		mw, mh := 520, 220
		scale := md.msg.Scale
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
		mx := (ctx.ConfigWorker.Config.WindowW - currW) / 2
		my := (ctx.ConfigWorker.Config.WindowH - currH) / 2

		// render a rounded rect for modal
		modalImg := ghelper.RenderRoundedRect(currW, currH, 16, color.RGBA{0xff, 0xff, 0xff, 0xff}, color.RGBA{0xcc, 0xcc, 0xcc, 0xff}, 3)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(mx), float64(my))
		screen.DrawImage(modalImg, op)

		// draw message text and OK button (only if fully opened)
		if scale > 0.85 {
			// text centered
			text.Draw(screen, md.msg.Text, basicfont.Face7x13, mx+32, my+60, color.Black)
			// OK button
			okW, okH := 120, 44
			okX := mx + (currW-okW)/2
			okY := my + currH - 56
			okImg := ghelper.RenderRoundedRect(okW, okH, 16, ctx.Theme.Accent, ctx.Theme.ButtonStroke, 3)
			op2 := &ebiten.DrawImageOptions{}
			op2.GeoM.Translate(float64(okX), float64(okY))
			screen.DrawImage(okImg, op2)
			text.Draw(screen, ctx.AssetsWorker.Lang().T("button.ok"), basicfont.Face7x13, okX+36, okY+28, color.White)
		}
	}

	// debug overlay
	ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f", ebiten.ActualTPS()))
}

func (md *GUIMenuDrawer) makeLayout(ctx *gctx.GUIGameContext) {
	ctx.Theme = gbase.LightPalette

	// center buttons vertically
	btnW, btnH := 320, 64
	gap := 18
	n := 4
	totalH := n*btnH + (n-1)*gap
	startY := (ctx.ConfigWorker.Config.WindowH - totalH) / 2
	cx := ctx.ConfigWorker.Config.WindowW / 2
	md.buttons = []*gbase.Button{}
	labels := []string{
		ctx.AssetsWorker.Lang().T("menu.play"),
		ctx.AssetsWorker.Lang().T("menu.editor"),
		ctx.AssetsWorker.Lang().T("menu.settings"),
		ctx.AssetsWorker.Lang().T("menu.exit"),
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
		md.buttons = append(md.buttons, b)
	}

	// language box bottom-left
	md.langBoxS = 56
	md.langBoxX = 20
	md.langBoxY = ctx.ConfigWorker.Config.WindowH - md.langBoxS - 20
}

func (md *GUIMenuDrawer) refreshButtons(ctx *gctx.GUIGameContext) {
	// update labels and re-render button images if needed
	labels := []string{
		ctx.AssetsWorker.Lang().T("menu.play"),
		ctx.AssetsWorker.Lang().T("menu.editor"),
		ctx.AssetsWorker.Lang().T("menu.settings"),
		ctx.AssetsWorker.Lang().T("menu.exit"),
	}
	for i := range md.buttons {
		md.buttons[i].Label = labels[i]

		md.buttons[i].Image = ghelper.RenderRoundedRect(
			md.buttons[i].W, md.buttons[i].H,
			16,                     // radius
			ctx.Theme.ButtonFill,   // <- новый fill
			ctx.Theme.ButtonStroke, // <- новый stroke
			3,                      // strokeWidth
		)
	}
}
