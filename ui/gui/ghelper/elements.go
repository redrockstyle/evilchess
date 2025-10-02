package ghelper

import (
	"evilchess/ui/gui/gbase"
	"fmt"
	"image/color"
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

// --- Number Wheel ---

type NumberWheel struct {
	X, Y     int // top-left of widget
	W, H     int // size of widget box
	Min, Max int // inclusive range
	Step     int // шаг (обычно 1)

	// визуальные настройки
	ItemH        int     // высота одного значения (px)
	VisibleCount int     // сколько элементов показываем (нечетное: 3,5,7...)
	CenterScale  float64 // масштаб центрального текста
	SideScale    float64 // масштаб соседних
	SideAlpha    float64 // альфа соседних
	Title        string  // optional title drawn above

	// internal state
	value     int     // текущий выбранный (целое значение)
	offset    float64 // плавный оффсет в px (0 = идеальное центрирование на value)
	velocity  float64 // скорость для инерции (px/sec)
	hover     bool
	onChange  func(int)
	allowWrap bool        // если true — после max идёт min
	fontFace  interface{} // expect font.Face but keep generic type to avoid import cycle; pass actual face as interface{}
}

// use: ctx.AssetsWorker.Fonts().Pixel
// use: visibleCount % 2 == 0
func NewNumberWheel(x, y, w, h, min, max, step int, initial int, visibleCount int, fontFace interface{}, title string) *NumberWheel {
	if visibleCount%2 == 0 {
		visibleCount = 3
	}
	itemH := h / visibleCount
	if itemH < 28 {
		itemH = 36
	}
	nw := &NumberWheel{
		X: x, Y: y, W: w, H: h,
		Min: min, Max: max, Step: step,
		ItemH: itemH, VisibleCount: visibleCount,
		CenterScale: 1.6, SideScale: 0.9, SideAlpha: 0.55,
		value:     clamp(initial, min, max),
		offset:    0,
		velocity:  0,
		allowWrap: false,
		fontFace:  fontFace,
		Title:     title,
	}
	return nw
}

// SetOnChange registers callback
func (nw *NumberWheel) SetOnChange(fn func(int)) {
	nw.onChange = fn
}

func (nw *NumberWheel) AllowWrap(v bool) {
	nw.allowWrap = v
}

func (nw *NumberWheel) Value() int { return nw.value }

func (nw *NumberWheel) SetValue(v int) {
	v = clamp(v, nw.Min, nw.Max)
	if nw.value != v {
		nw.value = v
		if nw.onChange != nil {
			nw.onChange(nw.value)
		}
	}
	nw.offset = 0
	nw.velocity = 0
}

func (nw *NumberWheel) Update(ctx *GUIGameContext) {
	// mouse hover detection
	mx, my := ebiten.CursorPosition()
	nw.hover = pointInRect(mx, my, nw.X, nw.Y, nw.W, nw.H)

	// wheel delta (ebiten.Wheel())
	_, dy := ebiten.Wheel()
	if nw.hover && dy != 0 {
		step := -int(math.Copysign(1, dy)) // normalize
		nw.velocity = float64(step) * float64(nw.ItemH) * 8.0
	}

	// keyboard support when hovered: up/down
	if nw.hover {
		if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
			nw.applySteps(-1)
		}
		if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
			nw.applySteps(1)
		}
	}

	// apply physics: velocity -> offset
	// dt approximated by 1/TPS
	dt := 1.0 / math.Max(1.0, ebiten.ActualTPS())
	// integrate velocity
	nw.offset += nw.velocity * dt
	// damping
	nw.velocity *= math.Pow(0.001, dt)

	// if offset passed threshold of ItemH/2 -> commit step(s)
	for nw.offset >= float64(nw.ItemH)/2.0 {
		nw.commitStep(-1) // negative because offset positive means user moved wheel down
		nw.offset -= float64(nw.ItemH)
	}
	for nw.offset <= -float64(nw.ItemH)/2.0 {
		nw.commitStep(1)
		nw.offset += float64(nw.ItemH)
	}

	// gentle snap to zero when tiny
	if math.Abs(nw.velocity) < 0.1 {
		target := 0.0
		// offset → 0
		nw.offset += (target - nw.offset) * 0.2
		if math.Abs(nw.offset) < 0.5 {
			// stoping
			nearest := math.Round(nw.offset / float64(nw.ItemH))
			nw.applySteps(int(nearest))
			nw.offset = 0
		}
	}
}

func (nw *NumberWheel) Draw(ctx *GUIGameContext, screen *ebiten.Image) {
	// background box
	bg := RenderRoundedRect(nw.W, nw.H, 10, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(nw.X), float64(nw.Y))
	screen.DrawImage(bg, op)

	// title if any
	if nw.Title != "" {
		// draw title above
		text.Draw(screen, ctx.AssetsWorker.Lang().T(nw.Title), ctx.AssetsWorker.Fonts().Pixel, nw.X+8, nw.Y-2, ctx.Theme.MenuText)
	}

	// central coords
	centerX := nw.X + nw.W/2
	centerY := nw.Y + nw.H/2

	// how many items to draw above/below
	half := nw.VisibleCount / 2

	// draw items: from -half .. +half
	for i := -half; i <= half; i++ {
		// compute index relative to current value considering offset
		// offset shifts items by fractional number of items (offset / ItemH)
		pos := float64(i)*float64(nw.ItemH) + nw.offset
		// screen position
		y := float64(centerY) + pos

		// index value for this row
		val := nw.value + i
		if nw.allowWrap {
			val = wrapInt(val, nw.Min, nw.Max)
		}

		// if val out of range and not wrap -> skip draw
		if val < nw.Min || val > nw.Max {
			continue
		}

		// visual scale/alpha depending on distance from center
		dist := math.Abs(float64(i) + nw.offset/float64(nw.ItemH))
		// scale := nw.SideScale + (nw.CenterScale-nw.SideScale)*math.Max(0.0, 1.0-math.Min(dist, 1.0))
		alpha := 1.0 - (1.0-nw.SideAlpha)*math.Min(dist, 1.0)

		// choose face and color
		face := ctx.AssetsWorker.Fonts().Pixel
		col := ctx.Theme.MenuText
		// modulate alpha into color (text.Draw doesn't accept alpha directly), so use ColorM with small image
		// Simpler: choose darker color for side elements by linear interpolation
		col = lerpColor(col, ctx.Theme.Bg, 1.0-alpha) // blend with background to reduce visibility

		// draw label as centered
		lbl := fmt.Sprintf("%02d", val)
		// compute text metrics
		// use text.BoundString to measure width for centering
		b := text.BoundString(face, lbl)
		tw := b.Dx()
		th := b.Dy()

		// prepare options to scale text by scale: draw into offscreen? simpler: change font size externally.
		// Here we approximate scaling by offsetting a bit and drawing normally (Pixel font looks OK).
		tx := int(float64(centerX) - float64(tw)/2.0)
		ty := int(y + float64(th)/2.0)

		// draw with color
		text.Draw(screen, lbl, face, tx, ty, col)
	}

	// border highlight when hover
	if nw.hover {
		EbitenutilDrawRectStroke(screen, float64(nw.X)+1, float64(nw.Y)+1, float64(nw.W)-2, float64(nw.H)-2, 2, ctx.Theme.Accent)
	}
}

func (nw *NumberWheel) applySteps(steps int) {
	for s := 0; s < absInt(steps); s++ {
		if steps > 0 {
			nw.commitStep(1)
		} else if steps < 0 {
			nw.commitStep(-1)
		}
	}
}

func (nw *NumberWheel) commitStep(dir int) {
	newVal := nw.value + dir*nw.Step
	if nw.allowWrap {
		newVal = wrapInt(newVal, nw.Min, nw.Max)
	} else {
		if newVal < nw.Min {
			newVal = nw.Min
		}
		if newVal > nw.Max {
			newVal = nw.Max
		}
	}
	if newVal != nw.value {
		nw.value = newVal
		if nw.onChange != nil {
			nw.onChange(nw.value)
		}
	}
}

func clamp(v, a, b int) int {
	if v < a {
		return a
	}
	if v > b {
		return b
	}
	return v
}

func absInt(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func wrapInt(v, a, b int) int {
	r := b - a + 1
	if r <= 0 {
		return a
	}
	off := (v - a) % r
	if off < 0 {
		off += r
	}
	return a + off
}

func pointInRect(px, py, x, y, w, h int) bool {
	return px >= x && py >= y && px < x+w && py < y+h
}

func lerpColor(c1, c2 color.RGBA, t float64) color.RGBA {
	if t <= 0 {
		return c1
	}
	if t >= 1 {
		return c2
	}
	out := color.RGBA{
		R: uint8(float64(c1.R)*(1.0-t) + float64(c2.R)*t),
		G: uint8(float64(c1.G)*(1.0-t) + float64(c2.G)*t),
		B: uint8(float64(c1.B)*(1.0-t) + float64(c2.B)*t),
		A: uint8(float64(c1.A)*(1.0-t) + float64(c2.A)*t),
	}
	return out
}
