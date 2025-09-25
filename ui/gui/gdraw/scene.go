package gdraw

import (
	"evilchess/src/engine/uci"
	"evilchess/ui/gui/gctx"
	"evilchess/ui/gui/ghelper"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
)

// ---- Scene ----

type Scene interface {
	Update(ctx *gctx.GUIGameContext) (SceneType, error)
	Draw(ctx *gctx.GUIGameContext, screen *ebiten.Image)
}

type SceneType int

const (
	SceneMenu SceneType = iota
	ScenePlay
	SceneEditor
	SceneAnalyzer
	SceneSettings
	SceneNotChanged
)

func (t SceneType) ToScene(s Scene, ctx *gctx.GUIGameContext) Scene {
	switch t {
	case SceneMenu:
		s = NewGUIMenuDrawer(ctx)
	case ScenePlay:
		s = NewGUIPlayDrawer(ctx)
	case SceneEditor:
		s = NewGUIEditDrawer(ctx)
	case SceneAnalyzer:
		s = NewGUIAnalyzeDrawer(ctx)
	case SceneSettings:
		s = NewGUISettingsDrawer(ctx)
	case SceneNotChanged:
	default:
	}
	return s
}

func DrawModal(ctx *gctx.GUIGameContext, scale float64, message string, screen *ebiten.Image) {

	// dim background
	// draw full-screen translucent rectangle
	overlay := ebiten.NewImage(ctx.Config.WindowW, ctx.Config.WindowH)
	overlay.Fill(ctx.Theme.ModalBg)
	screen.DrawImage(overlay, nil)

	bounds := text.BoundString(ctx.AssetsWorker.Fonts().Normal, message)
	textW := bounds.Dx()
	textH := bounds.Dy()

	paddingX := 64
	paddingY := 120

	mw := textW + paddingX
	mh := textH + paddingY
	// modal rectangle centered with scale
	// mw, mh := 520, 220
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
	mx := (ctx.Config.WindowW - currW) / 2
	my := (ctx.Config.WindowH - currH) / 2

	// render a rounded rect for modal
	modalImg := ghelper.RenderRoundedRect(currW, currH, 16, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(mx), float64(my))
	screen.DrawImage(modalImg, op)

	// draw message text and OK button (only if fully opened)
	if scale > 0.85 {
		// text centered
		text.Draw(screen, message, ctx.AssetsWorker.Fonts().Normal, mx+32, my+60, ctx.Theme.MenuText)
		// OK button
		okW, okH := 120, 44
		okX := mx + (currW-okW)/2
		okY := my + currH - 56
		okImg := ghelper.RenderRoundedRect(okW, okH, 16, ctx.Theme.Accent, ctx.Theme.ButtonStroke, 3)
		op2 := &ebiten.DrawImageOptions{}
		op2.GeoM.Translate(float64(okX), float64(okY))
		screen.DrawImage(okImg, op2)
		text.Draw(screen, ctx.AssetsWorker.Lang().T("button.ok"), ctx.AssetsWorker.Fonts().PixelLow, okX+36, okY+28, color.White)
	}
}

func IsCorrectEngine(ctx *gctx.GUIGameContext) error {
	e := uci.NewUCIExec(ctx.Logx, ctx.Config.UCIPath)
	if err := e.Init(); err != nil {
		return err
	}
	defer e.Close()
	return nil
}

// ---- SceneManager ----

type TransitionType int

const (
	TransNone TransitionType = iota
	TransCrossfade
	TransFade
	TransSlideLeft
)

// transition state
type Transition struct {
	typ        TransitionType
	dur        time.Duration
	elapsed    time.Duration
	from       Scene
	to         Scene
	fromImg    *ebiten.Image
	toImg      *ebiten.Image
	updateBoth bool
}

type SceneManager struct {
	ctx    *gctx.GUIGameContext
	cur    Scene
	trans  *Transition
	last   time.Time
	bg     color.Color
	width  int
	height int
}

// init and create menu scene
func NewSceneManager(ctx *gctx.GUIGameContext) *SceneManager {
	sm := &SceneManager{
		ctx:    ctx,
		last:   time.Now(),
		bg:     color.RGBA{0x10, 0x12, 0x14, 0xff},
		width:  ctx.Config.WindowW,
		height: ctx.Config.WindowH,
	}
	sm.cur = SceneMenu.ToScene(nil, ctx) // create menu scene
	return sm
}

func (m *SceneManager) Update() error {
	now := time.Now()
	dt := now.Sub(m.last)
	m.last = now

	// check transform
	if m.trans != nil {
		t := m.trans
		// update scene
		if t.updateBoth {
			if t.from != nil {
				_, _ = t.from.Update(m.ctx)
			}
			_, _ = t.to.Update(m.ctx)
		} else {
			_, _ = t.to.Update(m.ctx)
		}

		// redraw offscreen's
		if t.from != nil {
			t.fromImg.Fill(m.bg)
			t.from.Draw(m.ctx, t.fromImg)
		} else {
			t.fromImg.Fill(m.bg)
		}
		t.toImg.Fill(m.bg)
		t.to.Draw(m.ctx, t.toImg)

		t.elapsed += dt
		if t.elapsed >= t.dur {
			// transform done
			m.cur = t.to
			m.trans = nil
		}
		return nil
	}

	// default mode (no tranform)
	if m.cur == nil {
		return nil
	}
	req, err := m.cur.Update(m.ctx)
	if err != nil {
		return err
	}
	if req == SceneNotChanged { // check need to change scene
		return nil
	}

	next := req.ToScene(nil, m.ctx)
	// create offscreen
	fromImg := ebiten.NewImage(m.width, m.height)
	toImg := ebiten.NewImage(m.width, m.height)
	if m.cur != nil {
		fromImg.Fill(m.bg)
		m.cur.Draw(m.ctx, fromImg)
	} else {
		fromImg.Fill(m.bg)
	}
	toImg.Fill(m.bg)
	next.Draw(m.ctx, toImg)

	// init transform
	m.trans = &Transition{
		// typ:        TransSlideLeft,
		typ:        TransCrossfade,
		dur:        300 * time.Millisecond,
		elapsed:    0,
		from:       m.cur,
		to:         next,
		fromImg:    fromImg,
		toImg:      toImg,
		updateBoth: true, // if TRUE then Update() is called for both scene
	}
	return nil
}

// manager render
func (m *SceneManager) Draw(screen *ebiten.Image) {
	if m.trans == nil {
		if m.cur != nil {
			m.cur.Draw(m.ctx, screen)
			return
		}
		screen.Fill(m.bg)
		return
	}

	t := m.trans
	// compute progress 0..1
	prog := float64(t.elapsed) / float64(t.dur)
	if prog < 0 {
		prog = 0
	}
	if prog > 1 {
		prog = 1
	}

	switch t.typ {
	case TransCrossfade:
		// draw from with alpha (1-prog)
		if t.fromImg != nil {
			opFrom := &ebiten.DrawImageOptions{}
			var cm ebiten.ColorM
			cm.Scale(1, 1, 1, 1.0-prog)
			opFrom.ColorM = cm
			screen.DrawImage(t.fromImg, opFrom)
		}
		// draw to with alpha prog
		if t.toImg != nil {
			opTo := &ebiten.DrawImageOptions{}
			var cm2 ebiten.ColorM
			cm2.Scale(1, 1, 1, prog)
			opTo.ColorM = cm2
			screen.DrawImage(t.toImg, opTo)
		}
	case TransFade:
		// draw target and fade from black -> visible
		if t.toImg != nil {
			screen.DrawImage(t.toImg, nil)
		}
		overlay := ebiten.NewImage(m.width, m.height)
		a := uint8((1.0 - prog) * 255.0)
		overlay.Fill(color.RGBA{0, 0, 0, a})
		screen.DrawImage(overlay, nil)
	case TransSlideLeft:
		// slide from left to right
		w := float64(m.width)
		ofFrom := -prog * w
		ofTo := (1.0 - prog) * w
		if t.fromImg != nil {
			opFrom := &ebiten.DrawImageOptions{}
			opFrom.GeoM.Translate(ofFrom, 0)
			screen.DrawImage(t.fromImg, opFrom)
		}
		if t.toImg != nil {
			opTo := &ebiten.DrawImageOptions{}
			opTo.GeoM.Translate(ofTo, 0)
			screen.DrawImage(t.toImg, opTo)
		}
	default:
		// fallback
		if t.toImg != nil {
			screen.DrawImage(t.toImg, nil)
		}
	}
}
