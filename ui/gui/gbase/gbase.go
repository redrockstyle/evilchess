package gbase

import (
	"errors"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
)

// ---- Exit Call ----

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

func (p Palette) String() string {
	switch p {
	case LightPalette:
		return "light"
	case DarkPalette:
		return "dark"
	default:
	}
	return ""
}

func PaletteFromString(p string) Palette {
	switch p {
	case "light":
		return LightPalette
	case "dark":
		return DarkPalette
	default:
	}
	return Palette{}
}

var LightPalette = Palette{
	Bg:         color.RGBA{0xf7, 0xf7, 0xf7, 0xff},
	ButtonFill: color.RGBA{0xff, 0xff, 0xff, 0xff},
	// ButtonStroke: color.RGBA{0xd0, 0xd6, 0xdb, 0xff},
	// ButtonStroke: color.RGBA{0x44, 0x44, 0x44, 0xff},
	ButtonStroke: color.RGBA{0x88, 0x88, 0x88, 0xff},
	ButtonText:   color.RGBA{0x22, 0x22, 0x22, 0xff},
	MenuText:     color.RGBA{0x22, 0x22, 0x22, 0xff},
	Accent:       color.RGBA{0x22, 0x88, 0xcc, 0xff},
	ModalBg:      color.RGBA{0x00, 0x00, 0x00, 0x88},
}

var DarkPalette = Palette{
	Bg:         color.RGBA{0x12, 0x12, 0x12, 0xff},
	ButtonFill: color.RGBA{0x20, 0x20, 0x20, 0xff},
	// ButtonStroke: color.RGBA{0x40, 0x40, 0x40, 0xff},
	ButtonStroke: color.RGBA{0xdd, 0xdd, 0xdd, 0xff},
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

	// animation state
	Hover   bool // mouse over
	Pressed bool // mouse currently pressed on this button
	// animation variables
	Scale         float64 // current scale (1.0 default)
	TargetScale   float64
	OffsetY       float64 // current vertical offset for pressed effect
	TargetOffsetY float64
	AnimSpeed     float64 // how fast to approach target (per second)
}

func (b *Button) Contains(px, py int) bool {
	return px >= b.X && px < b.X+b.W && py >= b.Y && py < b.Y+b.H
}

// Call every Update: pass mouse info, returns true if click finished on this button
func (b *Button) HandleInput(px, py int, justClicked, justReleased bool) bool {
	inside := b.Contains(px, py)
	b.Hover = inside

	// pressed start only if mouse went down while cursor inside the button
	if justClicked && inside {
		b.Pressed = true
		b.TargetScale = 0.96
		b.TargetOffsetY = 3.0 // push down 3px
	}
	// release: if we released and the press started on this button and cursor still inside => click
	if justReleased {
		if b.Pressed && inside {
			// click confirmed
			b.Pressed = false
			b.TargetScale = 1.03 // small click bounce out
			b.TargetOffsetY = 0
			return true
		}
		// released outside: cancel press
		b.Pressed = false
		b.TargetScale = 1.0
		b.TargetOffsetY = 0
	}
	// hover enter/leave subtle effect
	if inside && !b.Pressed {
		b.TargetScale = 1.02
		b.TargetOffsetY = 0
	} else if !b.Pressed {
		b.TargetScale = 1.0
		b.TargetOffsetY = 0
	}
	return false
}

// Call every Update with dt seconds to approach the target values
func (b *Button) UpdateAnim(dt float64) {
	if b.AnimSpeed <= 0 {
		b.AnimSpeed = 8.0
	}
	// simple exponential approach (smooth)
	approach := func(cur *float64, target float64, speed float64) {
		// dt-based lerp toward target
		t := 1.0 - math.Exp(-speed*dt)
		*cur = *cur*(1.0-t) + target*t
	}

	approach(&b.Scale, b.TargetScale, b.AnimSpeed)
	approach(&b.OffsetY, b.TargetOffsetY, b.AnimSpeed)

	// subtle damping: if scale exceeded >1.01 after click, gently bring back to 1.0
	if !b.Pressed && math.Abs(b.Scale-1.03) < 0.005 {
		b.TargetScale = 1.0
	}
}

func (b *Button) DrawAnimated(screen *ebiten.Image, face font.Face, theme Palette) {
	if b.Image == nil {
		return
	}
	cx := float64(b.X + b.W/2)
	cy := float64(b.Y+b.H/2) + b.OffsetY

	// draw button image scaled around center
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(b.Image.Bounds().Dx())/2, -float64(b.Image.Bounds().Dy())/2)
	op.GeoM.Scale(b.Scale, b.Scale)
	op.GeoM.Translate(cx, cy)
	op.Filter = ebiten.FilterLinear // UI filter
	screen.DrawImage(b.Image, op)

	// draw label centered using font metrics
	bounds := text.BoundString(face, b.Label)
	tw := bounds.Dx()
	th := bounds.Dy()
	tx := int(cx) - tw/2
	ty := int(cy) + th/2
	text.Draw(screen, b.Label, face, tx, ty, theme.ButtonText)
}

type MessageBox struct {
	Open      bool
	Animating bool
	Scale     float64 // 0..1
	Opening   bool
	Text      string
	OnClose   func()
}
