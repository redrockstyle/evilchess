package gdraw

import (
	"evilchess/ui/gui/gbase"
	"evilchess/ui/gui/gctx"
	"evilchess/ui/gui/ghelper"
	"evilchess/ui/gui/ghelper/glang"
	"fmt"
	"math"
	"time"

	"github.com/fogleman/gg"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
)

type GUIMenuDrawer struct {
	buttons []*gbase.Button

	// messagebox
	msg gbase.MessageBox

	// language selector square bottom-left
	langBoxX, langBoxY, langBoxS int

	// about selector square bottom-left
	aboutBoxX, aboutBoxY, aboutBoxS int

	// click tracking
	prevMouseDown bool

	// crown
	crownImg         *ebiten.Image
	crownScale       int
	crownElapsed     float64
	crownBaseOffsetY float64
	shadowImg        *ebiten.Image

	// fro animation
	prevTime time.Time
}

func NewGUIMenuDrawer(ctx *gctx.GUIGameContext) *GUIMenuDrawer {
	md := &GUIMenuDrawer{}
	md.prevTime = time.Now()
	md.makeLayout(ctx)
	md.initCrown(ctx.AssetsWorker.Icon(60), 2)
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
	mx, my := ebiten.CursorPosition()
	mouseDown := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	justClicked := mouseDown && !md.prevMouseDown
	justReleased := !mouseDown && md.prevMouseDown
	md.prevMouseDown = mouseDown

	now := time.Now()
	dt := now.Sub(md.prevTime).Seconds()
	md.prevTime = now

	// if message box open -> handle clicks on it
	if md.msg.Open {
		if justClicked {
			// check OK button area in modal coords (we place it centered)
			// Modal geometry: centered rectangle
			bounds := text.BoundString(ctx.AssetsWorker.Fonts().Normal, ctx.AssetsWorker.Lang().T("about.body"))
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
	for _, b := range md.buttons {
		clicked := b.HandleInput(mx, my, justClicked, justReleased)
		b.UpdateAnim(dt)
		if clicked {
			mx, my := ebiten.CursorPosition()
			for i, b := range md.buttons {
				if ghelper.PointInRect(mx, my, b.X, b.Y, b.W, b.H) {
					ctx.Logx.Infof("%s (%d) clicked", b.Label, i)
					switch i {
					case 0: // menu.play
						return ScenePlay, nil
					case 1: // menu.editor
						return SceneEditor, nil
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

		}
	}

	if justClicked {
		// language box clickа
		if ghelper.PointInRect(mx, my, md.langBoxX, md.langBoxY, md.langBoxS, md.langBoxS) {
			if ctx.AssetsWorker.Lang().GetLang() == glang.EN {
				ctx.AssetsWorker.Lang().SetLang(glang.RU)
			} else {
				ctx.AssetsWorker.Lang().SetLang(glang.EN)
			}
			md.refreshButtons(ctx)
			return SceneNotChanged, nil
		}
		if ghelper.PointInRect(mx, my, md.aboutBoxX, md.aboutBoxY, md.aboutBoxS, md.aboutBoxS) {
			ghelper.ShowMessage(&md.msg, ctx.AssetsWorker.Lang().T("about.body"), nil)
			return SceneNotChanged, nil
		}
	}

	// update crown
	// now := time.Now()
	// dt := now.Sub(md.prevTime).Seconds()
	// md.prevTime = now
	md.crownElapsed += dt
	// md.crownElapsed += 0.5 / 60.0

	return SceneNotChanged, nil
}

func (md *GUIMenuDrawer) Draw(ctx *gctx.GUIGameContext, screen *ebiten.Image) {
	// clear background
	screen.Fill(ctx.Theme.Bg)

	md.drawButtuns(ctx, screen)
	md.drawBoxes(ctx, screen)

	// if message box open -> draw overlay and modal
	if md.msg.Open || md.msg.Animating {
		DrawModal(ctx, md.msg.Scale, md.msg.Text, screen)
	}

	md.drawCrown(screen)

	// debug overlay
	if ctx.ConfigWorker.Config.Debug {
		ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f", ebiten.ActualTPS()))
	}
}

func (md *GUIMenuDrawer) makeLayout(ctx *gctx.GUIGameContext) {
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

		b.Scale = 1.0
		b.TargetScale = 1.0
		b.OffsetY = 0
		b.TargetOffsetY = 0
		b.AnimSpeed = 10.0

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

	// about box bottom-left
	md.aboutBoxS = md.langBoxS
	md.aboutBoxX = md.langBoxX + 70
	md.aboutBoxY = ctx.ConfigWorker.Config.WindowH - md.aboutBoxS - 20
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
			ctx.Theme.ButtonFill,   // <- new fill
			ctx.Theme.ButtonStroke, // <- new stroke
			3,                      // strokeWidth
		)
	}
}

func (md *GUIMenuDrawer) initCrown(img *ebiten.Image, scale int) {
	md.crownImg = img
	md.crownScale = scale

	// level crown
	md.crownBaseOffsetY = -100.0 // -36..-80
	md.shadowImg = nil           // first render Draw
	md.crownElapsed = 0
}

func (md *GUIMenuDrawer) drawButtuns(ctx *gctx.GUIGameContext, screen *ebiten.Image) {
	// draw centered menu buttons
	for _, b := range md.buttons {
		b.DrawAnimated(screen, ctx.AssetsWorker.Fonts().Pixel, ctx.Theme)
	}
	// for _, b := range md.buttons {
	// 	b.DrawAnimated(screen, ctx.AssetsWorker.Fonts().Bold, ctx.Theme)
	// 	op := &ebiten.DrawImageOptions{}
	// 	op.GeoM.Translate(float64(b.X), float64(b.Y))
	// 	screen.DrawImage(b.Image, op)

	// 	// button label (text) centered
	// 	textX := b.X + b.W/2 - len(b.Label)*4
	// 	textY := b.Y + b.H/2 + 6
	// 	text.Draw(screen, b.Label, ctx.AssetsWorker.Fonts().Normal, textX, textY, ctx.Theme.ButtonText)

	// 	// outline (for strong contour) — draw thin stroke rectangle slightly larger
	// 	// ctx.Helper.EbitenutilDrawRectStroke(screen, float64(b.X), float64(b.Y), float64(b.W), float64(b.H), 2, md.pal.ButtonStroke)
	// }
}

func (md *GUIMenuDrawer) drawBoxes(ctx *gctx.GUIGameContext, screen *ebiten.Image) {
	// language box bottom-left (square)
	// square background
	langImg := ghelper.RenderRoundedRect(md.langBoxS, md.langBoxS, 8, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 2)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(md.langBoxX), float64(md.langBoxY))
	op.Filter = ebiten.FilterNearest
	screen.DrawImage(langImg, op)
	text.Draw(screen, ctx.AssetsWorker.Lang().T("lang.type"), ctx.AssetsWorker.Fonts().Normal, md.langBoxX+16, md.langBoxY+md.langBoxS/2+4, ctx.Theme.ButtonText)
	// small label
	// text.Draw(screen, ctx.AssetsWorker.Lang().T("lang.title"), basicfont.Face7x13, md.langBoxX+6, md.langBoxY-6, ctx.Theme.MenuText)

	// language box bottom-left (square)
	// square background
	aboutImg := ghelper.RenderRoundedRect(md.aboutBoxS, md.aboutBoxS, 8, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 2)
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(md.aboutBoxX), float64(md.aboutBoxY))
	op.Filter = ebiten.FilterNearest
	screen.DrawImage(aboutImg, op)
	text.Draw(screen, ctx.AssetsWorker.Lang().T("about.title"), ctx.AssetsWorker.Fonts().Normal, md.aboutBoxX+16, md.aboutBoxY+md.aboutBoxS/2+4, ctx.Theme.ButtonText)
	// small label
	// text.Draw(screen, ctx.AssetsWorker.Lang().T("lang.title"), basicfont.Face7x13, md.aboutBoxX+6, md.aboutBoxY-6, ctx.Theme.MenuText)

	// version on bottom-right
	ver := ctx.AssetsWorker.Lang().T("version")
	text.Draw(screen, ver, ctx.AssetsWorker.Fonts().Normal, ctx.ConfigWorker.Config.WindowW-160, ctx.ConfigWorker.Config.WindowH-24, ctx.Theme.MenuText)
}

func (md *GUIMenuDrawer) drawCrown(screen *ebiten.Image) {
	// --- draw crown ---
	play := md.buttons[0]
	centerX := float64(play.X + play.W/2)
	topY := float64(play.Y)

	// bobbing params
	freq := 1.0
	amp := 10.0
	slowAmp := 2.0
	rotFreq := 0.8
	rotAmpDeg := 6.0

	dy := math.Sin(2*math.Pi*freq*md.crownElapsed)*amp + math.Sin(2*math.Pi*0.15*md.crownElapsed)*slowAmp
	rot := math.Sin(2*math.Pi*rotFreq*md.crownElapsed) * (rotAmpDeg * math.Pi / 180.0)

	// compute final position
	w, h := md.crownImg.Size()
	finalX := centerX
	// offset to Y: topY*scale + baseOffset + bob
	finalY := topY - (float64(h)*float64(md.crownScale))/2.0 + md.crownBaseOffsetY + dy

	// shadow
	if md.shadowImg == nil {
		sw := int(float64(w*md.crownScale) * 1.6)
		sh := int(float64(h*md.crownScale) * 0.5)
		if sw < 4 {
			sw = 4
		}
		if sh < 2 {
			sh = 2
		}
		dc := gg.NewContext(sw, sh)
		for i := 0; i < 8; i++ {
			alpha := 0.18 * (1.0 - float64(i)/8.0) // 0.18 .. small
			dc.SetRGBA(0, 0, 0, alpha)
			pad := float64(i)
			dc.DrawEllipse(float64(sw)/2, float64(sh)/2+pad*0.2, float64(sw)/2-pad, float64(sh)/2-pad*0.6)
			dc.Fill()
		}
		md.shadowImg = ebiten.NewImageFromImage(dc.Image())
	}

	// draw shadow (scale & alpha vary with height)
	maxRange := amp + slowAmp
	heightFactor := (dy + maxRange) / (2 * maxRange) // 0..1
	shadowScale := 0.7 + (1.0-heightFactor)*0.25
	sW := float64(md.shadowImg.Bounds().Dx()) * shadowScale
	sH := float64(md.shadowImg.Bounds().Dy()) * shadowScale
	sop := &ebiten.DrawImageOptions{}
	sop.GeoM.Scale(sW/float64(md.shadowImg.Bounds().Dx()), sH/float64(md.shadowImg.Bounds().Dy()))
	// place shadow a bit below the top of the button (so appears on button surface)
	// shadowY := topY + float64(h*md.crownScale)/2.0 + 8
	shadowY := (topY + float64(play.H)) - (sH * 0.6) - 120 // level shadow
	sop.GeoM.Translate(finalX-sW/2.0, shadowY)
	sop.Filter = ebiten.FilterLinear
	screen.DrawImage(md.shadowImg, sop)

	// draw crown
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterNearest
	// transform: center -> scale -> rotate -> translate(finalX, finalY)
	op.GeoM.Translate(-float64(w)/2.0, -float64(h)/2.0)
	op.GeoM.Scale(float64(md.crownScale), float64(md.crownScale))
	op.GeoM.Rotate(rot)
	op.GeoM.Translate(finalX, finalY)
	screen.DrawImage(md.crownImg, op)
}
