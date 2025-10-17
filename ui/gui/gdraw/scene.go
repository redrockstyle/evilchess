package gdraw

import (
	"evilchess/src/engine/uci"
	"evilchess/ui/gui/ghelper"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---- Scene ----

type Scene interface {
	Update(ctx *ghelper.GUIGameContext) (SceneType, error)
	Draw(ctx *ghelper.GUIGameContext, screen *ebiten.Image)
}

type SceneType int

const (
	SceneMenu SceneType = iota
	ScenePlay
	ScenePlayMenu
	SceneEditor
	SceneAnalyzer
	SceneSettings
	SceneNotChanged
)

func (t SceneType) ToScene(s Scene, ctx *ghelper.GUIGameContext) Scene {
	switch t {
	case SceneMenu:
		s = NewGUIMenuDrawer(ctx)
	case ScenePlay:
		s = NewGUIPlayDrawer(ctx)
	case ScenePlayMenu:
		s = NewGUIPlayMenuDrawer(ctx)
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

func IsCorrectEngine(ctx *ghelper.GUIGameContext) error {
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
	ctx    *ghelper.GUIGameContext
	cur    Scene
	trans  *Transition
	last   time.Time
	bg     color.Color
	width  int
	height int
}

// init and create menu scene
func NewSceneManager(ctx *ghelper.GUIGameContext) *SceneManager {
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

// indexToFileRank: index 0..63 -> file(0..7), rank(0..7) where rank 0 == bottom (a1..h1).
func indexToFileRank(idx int) (int, int) {
	f := idx % 8
	r := idx / 8
	return f, r
}

func inBoard(px, py, bx, by, sqSize int) bool {
	return px >= bx && py >= by && px < bx+sqSize*8 && py < by+sqSize*8
}

// return: 0..63
// px,py — screen cords; flipped — chessboard flipped
func pixelToSquare(px, py, bx, by, sqSize int, flipped bool) int {
	fx := (px - bx) / sqSize
	fy := (py - by) / sqSize
	if fx < 0 {
		fx = 0
	}
	if fx > 7 {
		fx = 7
	}
	if fy < 0 {
		fy = 0
	}
	if fy > 7 {
		fy = 7
	}

	var file, rank int
	if !flipped {
		file = fx
		// fy: 0 = top row on screen -> that's rank 7, so rank = 7 - fy
		rank = 7 - fy
	} else {
		// flipped: top-left on screen corresponds to a1 (rank 0)
		file = 7 - fx
		rank = fy
	}
	return rank*8 + file
}
