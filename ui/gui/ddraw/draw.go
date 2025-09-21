package ddraw

import (
	"errors"
	"evilchess/src"
	"evilchess/src/base"
	"evilchess/src/logx"
	"evilchess/ui/gui/tools/lang"
	"image/color"

	"github.com/fogleman/gg"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var ErrExit = errors.New("exit request")

// --- UI constants ---

const (
	// SquareSize  = 64
	// BoardMargin = 12
	// RightPanelW = 320
	// HeaderH     = 60
	// BottomH     = 120
	// WindowPadW  = 40
	// WindowPadH  = 40
	AppBgR      = 90  //0xf0
	AppBgG      = 120 //0xf0
	AppBgB      = 130 //0xf0
	WindowW int = 1000
	WindowH int = 700
)

// ---- Styles (palettes) ----

type Palette struct {
	Bg           color.RGBA
	ButtonFill   color.RGBA
	ButtonStroke color.RGBA
	ButtonText   color.RGBA
	MenuText     color.RGBA
	Accent       color.RGBA
	ModalBg      color.RGBA
}

var LightPalette = Palette{
	Bg:           color.RGBA{0xf7, 0xf7, 0xf7, 0xff},
	ButtonFill:   color.RGBA{0xff, 0xff, 0xff, 0xff},
	ButtonStroke: color.RGBA{0xd0, 0xd6, 0xdb, 0xff},
	ButtonText:   color.RGBA{0x22, 0x22, 0x22, 0xff},
	MenuText:     color.RGBA{0x22, 0x22, 0x22, 0xff},
	Accent:       color.RGBA{0x22, 0x88, 0xcc, 0xff},
	ModalBg:      color.RGBA{0x00, 0x00, 0x00, 0x88},
}

var DarkPalette = Palette{
	Bg:           color.RGBA{0x12, 0x12, 0x12, 0xff},
	ButtonFill:   color.RGBA{0x20, 0x20, 0x20, 0xff},
	ButtonStroke: color.RGBA{0x40, 0x40, 0x40, 0xff},
	ButtonText:   color.RGBA{0xee, 0xee, 0xee, 0xff},
	MenuText:     color.RGBA{0xee, 0xee, 0xee, 0xff},
	Accent:       color.RGBA{0x2a, 0xa1, 0xd1, 0xff},
	ModalBg:      color.RGBA{0x00, 0x00, 0x00, 0x99},
}

// ---- UI elements ----

type Button struct {
	Label      string
	X, Y, W, H int
	Image      *ebiten.Image // pre-rendered rounded rect with stroke
}

type MessageBox struct {
	Open      bool
	Animating bool
	Scale     float64 // 0..1
	Opening   bool
	Text      string
	OnClose   func()
}

// ---- GUI Context ----

type GameContext struct {
	Builder *src.GameBuilder
	Helper  *GUIHelperDraw
	Lang    *lang.GUILangWorker
	Theme   Palette
	Window  struct{ W, H int }
	Logx    logx.Logger
}

type Scene interface {
	Update(ctx *GameContext) (Scene, error)
	Draw(ctx *GameContext, screen *ebiten.Image)
}

type GUIHelperDraw struct {
	background  color.Color
	pieceImages map[base.Piece]*ebiten.Image
}

func NewGUIHelperDraw() (*GUIHelperDraw, error) {
	files := []string{
		"assets/images/wking60.png",   // 0
		"assets/images/bking60.png",   // 1
		"assets/images/wqueen60.png",  // 2
		"assets/images/bqueen60.png",  // 3
		"assets/images/wbishop60.png", // 4
		"assets/images/bbishop60.png", // 5
		"assets/images/wknight60.png", // 6
		"assets/images/bknight60.png", // 7
		"assets/images/wrook60.png",   // 8
		"assets/images/brook60.png",   // 9
		"assets/images/wpawn60.png",   // 10
		"assets/images/bpawn60.png",   // 11
	}
	keys := []base.Piece{
		base.WKing,
		base.BKing,
		base.WQueen,
		base.BQueen,
		base.WBishop,
		base.BBishop,
		base.WKnight,
		base.BKnight,
		base.WRook,
		base.BRook,
		base.WPawn,
		base.BPawn,
		base.InvalidPiece,
	}
	figureImages := make(map[base.Piece]*ebiten.Image)
	for i := 0; i < 12; i++ {
		img, _, err := ebitenutil.NewImageFromFile(files[i])
		if err != nil {
			return nil, err
		}
		figureImages[keys[i]] = img
	}
	return &GUIHelperDraw{pieceImages: figureImages}, nil
}

// ---- helpers ----

func (hd *GUIHelperDraw) RenderRoundedRect(w, h, radius int, fill color.RGBA, stroke color.RGBA, strokeW float64) *ebiten.Image {
	// create a context with alpha and draw rounded rectangle using gg (anti-aliased)
	dc := gg.NewContext(w, h)
	dc.SetRGBA255(int(fill.R), int(fill.G), int(fill.B), int(fill.A))
	dc.DrawRoundedRectangle(0, 0, float64(w), float64(h), float64(radius))
	dc.FillPreserve()
	dc.SetRGBA255(int(stroke.R), int(stroke.G), int(stroke.B), int(stroke.A))
	dc.SetLineWidth(strokeW)
	dc.Stroke()
	img := dc.Image()
	return ebiten.NewImageFromImage(img)
}

func (hd *GUIHelperDraw) EbitenutilDrawRectStroke(screen *ebiten.Image, x, y, w, h float64, sw float64, c color.RGBA) {
	// draw four thin rects to simulate stroke slightly outside
	hd.EbitenutilDrawRect(screen, x-sw/2, y-sw/2, w+sw, sw, c)   // top
	hd.EbitenutilDrawRect(screen, x-sw/2, y+h-sw/2, w+sw, sw, c) // bottom
	hd.EbitenutilDrawRect(screen, x-sw/2, y, sw, h, c)           // left
	hd.EbitenutilDrawRect(screen, x+w-sw/2, y, sw, h, c)         // right
}

func (hd *GUIHelperDraw) EbitenutilDrawRect(screen *ebiten.Image, x, y, w, h float64, c color.RGBA) {
	img := ebiten.NewImage(int(w), int(h))
	img.Fill(c)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y)
	screen.DrawImage(img, op)
}

func (hd *GUIHelperDraw) PointInRect(px, py, rx, ry, rw, rh int) bool {
	return px >= rx && px < rx+rw && py >= ry && py < ry+rh
}

// ---- MessageBox ----

func (hd *GUIHelperDraw) AnimateMessage(box *MessageBox) {
	// basic animation: linear scale and fade (scale 0->1 opening, 1->0 closing)
	const dt = 1.0 / 60.0
	const speed = 6.0 // how fast the tween goes
	if box.Opening {
		box.Scale += speed * dt
		if box.Scale >= 1.0 {
			box.Scale = 1.0
			box.Animating = false
		}
	} else {
		box.Scale -= speed * dt
		if box.Scale <= 0.0 {
			box.Scale = 0.0
			box.Animating = false
			box.Open = false
			// call OnClose if set
			if box.OnClose != nil {
				box.OnClose()
			}
		}
	}
}

func (hd *GUIHelperDraw) ShowMessage(box *MessageBox, msg string, onClose func()) {
	box.Text = msg
	box.Open = true
	box.Opening = true
	box.Animating = true
	box.Scale = 0.0
	box.OnClose = onClose
}
