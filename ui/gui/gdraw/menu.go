package gdraw

import (
	"evilchess/ui/gui/gbase"
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
	msg        *ghelper.MessageBox
	buttons    []*ghelper.Button
	btnPlayIdx int
	btnEditIdx int
	btnStgsIdx int
	btnExitIdx int
	btnLangIdx int
	btnInfoIdx int

	// click tracking
	prevMouseDown bool

	// crown
	crownImg         *ebiten.Image
	crownScale       int
	crownElapsed     float64
	crownBaseOffsetY float64
	shadowImg        *ebiten.Image

	// for animation
	lastTick time.Time
}

func NewGUIMenuDrawer(ctx *ghelper.GUIGameContext) *GUIMenuDrawer {
	md := &GUIMenuDrawer{lastTick: time.Now()}
	// md.makeLayout(ctx)

	// buttons
	md.buttons = []*ghelper.Button{}
	btnW, btnH := 320, 64
	gap := 18
	n := 4
	totalH := n*btnH + (n-1)*gap
	startY := (ctx.Config.WindowH - totalH) / 2
	cx := ctx.Config.WindowW / 2
	md.btnPlayIdx, md.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("menu.play"), cx-btnW/2, startY, btnW, btnH, md.buttons)
	md.btnExitIdx, md.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("menu.editor"), cx-btnW/2, startY+(btnH+gap), btnW, btnH, md.buttons)
	md.btnStgsIdx, md.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("menu.settings"), cx-btnW/2, startY+2*(btnH+gap), btnW, btnH, md.buttons)
	md.btnExitIdx, md.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("menu.exit"), cx-btnW/2, startY+3*(btnH+gap), btnW, btnH, md.buttons)
	// left-down buttons
	md.btnLangIdx, md.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("lang.type"), 20, ctx.Config.WindowH-76, 56, 56, md.buttons)
	md.btnInfoIdx, md.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("about.title"), 90, ctx.Config.WindowH-76, 56, 56, md.buttons)

	md.msg = &ghelper.MessageBox{}
	md.initCrown(ctx.AssetsWorker.Icon(60), 2)
	return md
}

func (md *GUIMenuDrawer) Update(ctx *ghelper.GUIGameContext) (SceneType, error) {
	// for animation
	now := time.Now()
	dt := now.Sub(md.lastTick).Seconds()
	md.lastTick = now

	// check mouse just clicked
	mx, my := ebiten.CursorPosition()
	mouseDown := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	justClicked := mouseDown && !md.prevMouseDown
	justReleased := !mouseDown && md.prevMouseDown
	md.prevMouseDown = mouseDown

	// if message box open -> handle clicks on it
	if md.msg.Open {
		if justClicked {
			// check OK button area in modal coords (we place it centered)
			// Modal geometry: centered rectangle
			bounds := text.BoundString(ctx.AssetsWorker.Fonts().Normal, ctx.AssetsWorker.Lang().T("about.body"))
			md.msg.CollapseMessageInRect(ctx.Config.WindowW, ctx.Config.WindowH, bounds.Dx(), bounds.Dy())

		}
		// animate open/close
		md.msg.AnimateMessage()
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
					case md.btnPlayIdx:
						return ScenePlayMenu, nil
					case md.btnExitIdx:
						return SceneEditor, nil
					case md.btnStgsIdx:
						return SceneSettings, nil
					case md.btnExitIdx:
						return SceneNotChanged, gbase.ErrExit
					case md.btnLangIdx:
						if ctx.AssetsWorker.Lang().GetLang() == glang.EN {
							ctx.AssetsWorker.Lang().SetLang(glang.RU)
						} else {
							ctx.AssetsWorker.Lang().SetLang(glang.EN)
						}
						md.refreshButtons(ctx)
						return SceneNotChanged, nil
					case md.btnInfoIdx:
						md.msg.ShowMessage(ctx.AssetsWorker.Lang().T("about.body"), nil)
					}
					return SceneNotChanged, nil
				}
			}

		}
	}

	// update crown
	md.crownElapsed += dt
	// md.crownElapsed += 0.5 / 60.0

	return SceneNotChanged, nil
}

func (md *GUIMenuDrawer) Draw(ctx *ghelper.GUIGameContext, screen *ebiten.Image) {
	// background
	screen.Fill(ctx.Theme.Bg)

	for i, b := range md.buttons {
		face := ctx.AssetsWorker.Fonts().Pixel
		if i == md.btnLangIdx || i == md.btnInfoIdx {
			face = ctx.AssetsWorker.Fonts().PixelLow
		}
		b.DrawAnimated(screen, face, ctx.Theme)
	}

	// if message box open -> draw overlay and modal
	if md.msg.Open || md.msg.Animating {
		DrawModal(ctx, md.msg.Scale, md.msg.Text, screen)
	}

	md.drawCrown(screen)

	text.Draw(screen, ctx.AssetsWorker.Lang().T("version"), ctx.AssetsWorker.Fonts().Normal, ctx.Config.WindowW-160, ctx.Config.WindowH-24, ctx.Theme.MenuText)
	// debug overlay
	if ctx.Config.Debug {
		ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f", ebiten.ActualTPS()))
	}
}

func (md *GUIMenuDrawer) refreshButtons(ctx *ghelper.GUIGameContext) {
	// update labels and re-render button images if needed
	labels := []string{
		ctx.AssetsWorker.Lang().T("menu.play"),
		ctx.AssetsWorker.Lang().T("menu.editor"),
		ctx.AssetsWorker.Lang().T("menu.settings"),
		ctx.AssetsWorker.Lang().T("menu.exit"),
		ctx.AssetsWorker.Lang().T("lang.type"),
		ctx.AssetsWorker.Lang().T("about.title"),
	}
	for i := range md.buttons {
		md.buttons[i].Label = labels[i]
		md.buttons[i].Image = ghelper.RenderRoundedRect(md.buttons[i].W, md.buttons[i].H, 16, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)
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
