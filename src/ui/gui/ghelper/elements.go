package ghelper

import (
	"evilchess/src/ui/gui/gbase"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"time"

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
type MessageChoice struct {
	Label string
	Image *ebiten.Image
	Value interface{}
}

type imageRect struct{ X, Y, W, H int }

type MessageBox struct {
	// content
	Label string

	// state
	Open      bool
	Animating bool
	Scale     float64 // 0..1
	Opening   bool
	OnClose   func()

	// choices
	Choices    []MessageChoice
	HoverIndex int
	OnSelect   func(idx int, v interface{})

	// internal layout cache
	lastModalRect imageRect
}

func NewMessageBox() *MessageBox {
	return &MessageBox{
		Scale:      0,
		HoverIndex: -1,
	}
}

func (mb *MessageBox) AnimateMessage() {
	const dt = 1.0 / 60.0
	const speed = 6.0
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
			if mb.OnClose != nil {
				mb.OnClose()
			}
		}
	}
}

func (mb *MessageBox) ShowMessage(msg string, onClose func()) {
	mb.Label = msg
	mb.Choices = nil
	mb.OnSelect = nil
	mb.HoverIndex = -1

	mb.Open = true
	mb.Opening = true
	mb.Animating = true
	mb.Scale = 0.0
	mb.OnClose = onClose
}

func (mb *MessageBox) ShowMessageWithChoices(msg string, choices []MessageChoice, onSelect func(idx int, v interface{})) {
	mb.Label = msg
	mb.Choices = choices
	mb.OnSelect = onSelect
	mb.HoverIndex = -1

	mb.Open = true
	mb.Opening = true
	mb.Animating = true
	mb.Scale = 0.0
	mb.OnClose = nil
}

func (mb *MessageBox) Update(ctx *GUIGameContext, mx, my int, justReleased bool) {
	if !mb.Open && !mb.Animating {
		return
	}
	if !mb.Opening && mb.Animating {
		return
	}

	// layout constants
	fontNormal := ctx.AssetsWorker.Fonts().Normal
	txtBounds := text.BoundString(fontNormal, mb.Label)
	textW := txtBounds.Dx()
	textH := txtBounds.Dy()
	if textW < 200 {
		textW = 200
	}
	paddingX := 64
	paddingY := 40
	choiceW := 64
	choiceH := 64
	choiceGap := 14

	choicesCount := len(mb.Choices)
	mw := textW + paddingX
	mh := textH + paddingY
	if choicesCount > 0 {
		totalChoicesW := choicesCount*choiceW + (choicesCount-1)*choiceGap
		if float64(totalChoicesW+paddingX) > float64(mw) {
			mw = totalChoicesW + paddingX
		}
		// h: text + padding + choiceH + offset
		mh = textH + paddingY + choiceH + 24
	} else {
		// for OK button
		mh += 64
	}

	// scale-aware current size
	scale := mb.Scale
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

	// center
	mx0 := (ctx.Config.WindowW - currW) / 2
	my0 := (ctx.Config.WindowH - currH) / 2

	mb.lastModalRect = imageRect{X: mx0, Y: my0, W: currW, H: currH}
	mb.HoverIndex = -1
	if choicesCount > 0 {
		totalChoicesW := choicesCount*choiceW + (choicesCount-1)*choiceGap
		startX := mx0 + (currW-totalChoicesW)/2
		textY := my0 + 40 + textH
		choicesY := textY + 12
		if InsideRect(mx, my, startX, choicesY, totalChoicesW, choiceH) {
			relX := mx - startX
			idx := relX / (choiceW + choiceGap)
			if idx >= 0 && idx < choicesCount {
				cellX := startX + idx*(choiceW+choiceGap)
				if InsideRect(mx, my, cellX, choicesY, choiceW, choiceH) {
					mb.HoverIndex = idx
				}
			}
		}
		if justReleased && mb.HoverIndex >= 0 {
			idx := mb.HoverIndex
			val := mb.Choices[idx].Value
			if mb.OnSelect != nil {
				mb.OnSelect(idx, val)
			}
			mb.CollapseMessage()
		}
	} else {
		// OK button area
		okW, okH := 120, 44
		okX := mx0 + (currW-okW)/2
		okY := my0 + currH - okH - 20
		if justReleased && PointInRect(mx, my, okX, okY, okW, okH) {
			mb.CollapseMessage()
		}
	}
}

func (mb *MessageBox) IsOverlayed() bool {
	return mb.Open || mb.Animating
}

func (mb *MessageBox) Draw(ctx *GUIGameContext, screen *ebiten.Image) {
	if !mb.Open && !mb.Animating {
		return
	}
	// overlay
	overlay := ebiten.NewImage(ctx.Config.WindowW, ctx.Config.WindowH)
	overlay.Fill(ctx.Theme.ModalBg)
	screen.DrawImage(overlay, nil)
	// text
	fontNormal := ctx.AssetsWorker.Fonts().Normal
	bounds := text.BoundString(fontNormal, mb.Label)
	textW := bounds.Dx()
	textH := bounds.Dy()
	if textW < 200 {
		textW = 200
	}

	// layout params
	paddingX := 64
	paddingY := 40
	choiceW := 64
	choiceH := 64
	choiceGap := 14
	choicesCount := len(mb.Choices)

	mw := textW + paddingX
	mh := textH + paddingY
	if choicesCount > 0 {
		totalChoicesW := choicesCount*choiceW + (choicesCount-1)*choiceGap
		if float64(totalChoicesW+paddingX) > float64(mw) {
			mw = totalChoicesW + paddingX
		}
		mh = textH + paddingY + choiceH + 24
	} else {
		mh += 64
	}

	scale := mb.Scale
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

	modalImg := RenderRoundedRect(currW, currH, 16, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(mx), float64(my))
	screen.DrawImage(modalImg, op)
	if scale > 0.85 {
		textX := mx + 32
		textY := my + 20
		if textH < 40 {
			textY += textH
		} else {
			textY += 22
		}
		text.Draw(screen, mb.Label, fontNormal, textX, textY, ctx.Theme.MenuText)
		if choicesCount > 0 {
			totalChoicesW := choicesCount*choiceW + (choicesCount-1)*choiceGap
			startX := mx + (currW-totalChoicesW)/2
			choicesY := textY + 12
			cx, cy := ebiten.CursorPosition()
			hover := -1
			if InsideRect(cx, cy, startX, choicesY, totalChoicesW, choiceH) {
				relX := cx - startX
				idx := relX / (choiceW + choiceGap)
				if idx >= 0 && idx < choicesCount {
					cellX := startX + idx*(choiceW+choiceGap)
					if InsideRect(cx, cy, cellX, choicesY, choiceW, choiceH) {
						hover = idx
					}
				}
			}
			for i, ch := range mb.Choices {
				cxPos := startX + i*(choiceW+choiceGap)
				bg := RenderRoundedRect(choiceW, choiceH, 13, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)
				opc := &ebiten.DrawImageOptions{}
				opc.GeoM.Translate(float64(cxPos), float64(choicesY))
				screen.DrawImage(bg, opc)

				if ch.Image != nil {
					iw, ih := ch.Image.Size()
					if iw > 0 && ih > 0 {
						sx := float64(choiceW) / float64(iw)
						sy := float64(choiceH) / float64(ih)
						s := math.Min(sx, sy) * 0.9
						opImg := &ebiten.DrawImageOptions{}
						opImg.GeoM.Scale(s, s)
						tw := float64(iw) * s
						th := float64(ih) * s
						tx := float64(cxPos) + (float64(choiceW)-tw)/2.0
						ty := float64(choicesY) + (float64(choiceH)-th)/2.0
						opImg.GeoM.Translate(tx, ty)
						opImg.Filter = ebiten.FilterLinear
						screen.DrawImage(ch.Image, opImg)
					}
				} else if ch.Label != "" {
					text.Draw(screen, ch.Label, ctx.AssetsWorker.Fonts().Pixel, cxPos+8, choicesY+40, ctx.Theme.MenuText)
				}

				if i == hover {
					EbitenutilDrawRectStroke(screen, float64(cxPos)+2, float64(choicesY)+2, float64(choiceW)-4, float64(choiceH)-4, 3, ctx.Theme.Accent)
				}
			}
		} else {
			// OK button
			okW, okH := 120, 44
			okX := mx + (currW-okW)/2
			okY := my + currH - okH - 20
			okImg := RenderRoundedRect(okW, okH, 12, ctx.Theme.Accent, ctx.Theme.ButtonStroke, 3)
			op2 := &ebiten.DrawImageOptions{}
			op2.GeoM.Translate(float64(okX), float64(okY))
			screen.DrawImage(okImg, op2)
			text.Draw(screen, ctx.AssetsWorker.Lang().T("button.ok"), ctx.AssetsWorker.Fonts().PixelLow, okX+36, okY+30, color.White)
		}
	}
}

func (mb *MessageBox) CollapseMessage() {
	mb.Opening = false
	mb.Animating = true
	if mb.OnClose == nil {
		mb.OnClose = func() {}
	}
}

// Check click OK
func (mb *MessageBox) CollapseMessageInRect(mx, my int) {
	r := mb.lastModalRect
	if r.W == 0 || r.H == 0 {
		return
	}
	okW, okH := 120, 44
	okX := r.X + (r.W-okW)/2
	okY := r.Y + r.H - okH - 20
	if PointInRect(mx, my, okX, okY, okW, okH) {
		mb.CollapseMessage()
	}
}

// --- Circular Loader ---

// - x, y: центр в пикселях
// - radius: радиус окружности, по которой бегают точки
// - dotSize: базовый диаметр точки (в пикселях)
// - speedRPS: обороты в секунду; положительное -> по часовой стрелке, отрицательное -> обратное
// - segments: количество точек/сегментов
// - colors: слайс color.RGBA (nil или пустой -> используется дефолтная палитра)
type CircularLoader struct {
	X, Y     int
	Radius   float64
	DotSize  float64
	SpeedRPS float64
	Segments int
	Colors   []color.RGBA

	Active bool

	phase float64
	bg    color.Color
	lastT time.Time

	dotImg *ebiten.Image
}

func NewCircularLoader(x, y int, radius, dotSize, speedRPS float64, segments int, ctx *GUIGameContext) *CircularLoader {
	if dotSize <= 0 {
		dotSize = math.Max(6, radius*0.12)
	}
	if segments <= 0 {
		segments = 6
	}

	c := &CircularLoader{
		X:        x,
		Y:        y,
		Radius:   radius,
		DotSize:  dotSize,
		SpeedRPS: speedRPS,
		Segments: segments,
		Colors: []color.RGBA{
			ctx.Theme.ButtonStroke,
			// {R: 120, G: 200, B: 255, A: 255},
			// {R: 100, G: 180, B: 230, A: 255},
			// {R: 80, G: 150, B: 210, A: 255},
		},
		Active: false,
		phase:  0,
	}
	c.makeDotImage()
	return c
}

func (c *CircularLoader) makeDotImage() {
	size := int(math.Ceil(c.DotSize))
	if size < 1 {
		size = 1
	}
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{0, 0, 0, 0}}, image.Point{}, draw.Src)

	cx := float64(size-1) / 2.0
	cy := cx
	r := c.DotSize / 2.0
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			if dx*dx+dy*dy <= r*r {
				// soft edge ~1px
				dist := math.Sqrt(dx*dx + dy*dy)
				var a uint8 = 255
				if dist > r-1.0 {
					alpha := 1.0 - (dist - (r - 1.0))
					if alpha < 0 {
						alpha = 0
					}
					a = uint8(255 * alpha)
				}
				img.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: a})
			}
		}
	}
	c.dotImg = ebiten.NewImageFromImage(img)
}

func (c *CircularLoader) Update(dt float64) {
	if !c.Active {
		return
	}
	// dφ = -2π * speed * dt
	angular := -2 * math.Pi * c.SpeedRPS
	c.phase += dt * angular
	// normalize 0..2π
	c.phase = math.Mod(c.phase, 2*math.Pi)
	if c.phase < 0 {
		c.phase += 2 * math.Pi
	}
}

func (c *CircularLoader) Draw(screen *ebiten.Image) {
	if !c.Active {
		return
	}
	if c.Segments <= 0 {
		c.Segments = 1
	}
	if c.dotImg == nil {
		c.makeDotImage()
	}

	segAngle := 2 * math.Pi / float64(c.Segments)

	for i := c.Segments - 1; i >= 0; i-- {
		n := float64(i)
		angle := c.phase + n*segAngle
		x := float64(c.X) + math.Cos(angle)*c.Radius
		y := float64(c.Y) + math.Sin(angle)*c.Radius

		t := n / float64(c.Segments)    // 0..1 (head ~0, tail ~1)
		headFactor := 1.0 - t           // headFactor 1..0
		scale := 0.6 + 0.9*headFactor   // 0.6..1.5
		alpha := 0.25 + 0.75*headFactor // 0.25..1.0

		col := c.Colors[i%len(c.Colors)]
		cr := float64(col.R) / 255.0
		cg := float64(col.G) / 255.0
		cb := float64(col.B) / 255.0
		colA := (float64(col.A) / 255.0) * alpha

		sw, sh := c.dotImg.Size()
		if sw == 0 || sh == 0 {
			continue
		}
		target := c.DotSize * scale
		sx := target / float64(sw)
		sy := target / float64(sh)

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(sx, sy)
		op.GeoM.Translate(x-float64(sw)*sx/2.0, y-float64(sh)*sy/2.0)

		var cm ebiten.ColorM
		cm.Scale(float64(cr), float64(cg), float64(cb), float64(colA))
		op.ColorM = cm

		screen.DrawImage(c.dotImg, op)
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

func InsideRect(px, py, x, y, w, h int) bool {
	return px >= x && py >= y && px < x+w && py < y+h
}
