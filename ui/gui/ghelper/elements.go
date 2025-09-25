package ghelper

import (
	"evilchess/ui/gui/gbase"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
)

// ---- UI ELEMENTS ----

// ---- Button ----

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

func (b *Button) DrawAnimated(screen *ebiten.Image, face font.Face, theme gbase.Palette) {
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

// ---- MessageBox ----

type MessageBox struct {
	Label      string
	X, Y, W, H int     // position
	Open       bool    //
	Animating  bool    //
	Scale      float64 // 0..1
	Opening    bool
	Text       string
	OnClose    func()
}

func (mb *MessageBox) AnimateMessage() {
	// basic animation: linear scale and fade (scale 0->1 opening, 1->0 closing)
	const dt = 1.0 / 60.0
	const speed = 6.0 // how fast the tween goes
	if mb.Opening {
		mb.Scale += speed * dt
		if mb.Scale >= 1.0 {
			mb.Scale = 1.0
			mb.Animating = false
		}
	} else {
		mb.Scale -= speed * dt
		if mb.Scale <= 0.0 {
			mb.Scale = 0.0
			mb.Animating = false
			mb.Open = false
			// call OnClose if set
			if mb.OnClose != nil {
				mb.OnClose()
			}
		}
	}
}

func (mb *MessageBox) ShowMessage(msg string, onClose func()) {
	mb.Text = msg
	mb.Open = true
	mb.Opening = true
	mb.Animating = true
	mb.Scale = 0.0
	mb.OnClose = onClose
}

func (mb *MessageBox) ShowMessageInRect(mx, my int) bool {
	if PointInRect(mx, my, mb.X, mb.Y, mb.W, mb.H) {
		mb.ShowMessage(mb.Label, nil)
		return true
	}
	return false
}

func (mb *MessageBox) CollapseMessage() {
	// start closing animation
	mb.Opening = false
	mb.Animating = true
	// call close handler after animation ends
	if mb.OnClose == nil {
		mb.OnClose = func() {}
	}
}

func (mb *MessageBox) CollapseMessageInRect(windW, windH, textW, textH int) {
	mw := textW + 64
	mh := textH + 120
	mx := (windW - mw) / 2
	my := (windH - mh) / 2

	okW, okH := 120, 44
	okX := mx + (mw-okW)/2
	okY := my + mh - 56

	mxPos, myPos := ebiten.CursorPosition()
	if PointInRect(mxPos, myPos, okX, okY, okW, okH) {
		// start closing animation
		mb.Opening = false
		mb.Animating = true
		// call close handler after animation ends
		if mb.OnClose == nil {
			mb.OnClose = func() {}
		}
	}
}
