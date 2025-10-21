// main.go
package main

import (
	"fmt"
	"image/color"
	"log"

	"github.com/fogleman/gg"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"
)

// ---- Simple i18n (very small) ----
var locales = map[string]map[string]string{
	"en": {
		"menu.play":      "Play",
		"menu.editor":    "Board Editor",
		"menu.settings":  "Settings",
		"menu.exit":      "Exit",
		"lang":           "EN",
		"message.sample": "This is a MessageBox.",
		"button.ok":      "OK",
		"version":        "EvilChess 0.1.0",
	},
	"ru": {
		"menu.play":      "Играть",
		"menu.editor":    "Редактор",
		"menu.settings":  "Настройки",
		"menu.exit":      "Выход",
		"lang":           "RU",
		"message.sample": "Это MessageBox.",
		"button.ok":      "OK",
		"version":        "EvilChess 0.1.0",
	},
}

func t(lang, key string) string {
	if m, ok := locales[lang]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	return key
}

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

// App holds UI state
type App struct {
	lang string
	pal  Palette

	// buttons centered vertically
	buttons []*Button

	// language selector square bottom-left
	langBoxX, langBoxY, langBoxS int

	// messagebox
	msg MessageBox

	// small cached images for controls
	btnImageCache *ebiten.Image

	// window size
	w, h int

	// click tracking
	prevMouseDown bool
}

func NewApp() *App {
	a := &App{
		lang: "en",
		pal:  LightPalette,
		w:    1000,
		h:    700,
	}
	a.makeLayout()
	return a
}

func (a *App) makeLayout() {
	// center buttons vertically
	btnW, btnH := 320, 64
	gap := 18
	n := 4
	totalH := n*btnH + (n-1)*gap
	startY := (a.h - totalH) / 2
	cx := a.w / 2
	a.buttons = []*Button{}
	labels := []string{t(a.lang, "menu.play"), t(a.lang, "menu.editor"), t(a.lang, "menu.settings"), t(a.lang, "menu.exit")}
	for i, lab := range labels {
		x := cx - btnW/2
		y := startY + i*(btnH+gap)
		b := &Button{
			Label: lab,
			X:     x, Y: y, W: btnW, H: btnH,
		}
		// pre-render button image
		b.Image = renderRoundedRect(btnW, btnH, 12, a.pal.ButtonFill, a.pal.ButtonStroke, 2)
		a.buttons = append(a.buttons, b)
	}

	// language box bottom-left
	a.langBoxS = 56
	a.langBoxX = 20
	a.langBoxY = a.h - a.langBoxS - 20
}

func (a *App) Update() error {
	// keyboard: toggle palette for demo
	if ebiten.IsKeyPressed(ebiten.KeyTab) {
		a.pal = DarkPalette
		a.refreshButtons()
	}
	// check mouse just clicked
	mouseDown := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	justClicked := mouseDown && !a.prevMouseDown
	a.prevMouseDown = mouseDown

	// if message box open -> handle clicks on it
	if a.msg.Open {
		if justClicked {
			// check OK button area in modal coords (we place it centered)
			// Modal geometry: centered rectangle
			mw, mh := 520, 220
			mx := (a.w - mw) / 2
			my := (a.h - mh) / 2
			okW, okH := 120, 44
			okX := mx + (mw-okW)/2
			okY := my + mh - 56
			mxPos, myPos := ebiten.CursorPosition()
			if pointInRect(mxPos, myPos, okX, okY, okW, okH) {
				// start closing animation
				a.msg.Opening = false
				a.msg.Animating = true
				// call close handler after animation ends
				if a.msg.OnClose == nil {
					a.msg.OnClose = func() {}
				}
			}
		}
		// animate open/close
		a.animateMessage()
		return nil
	}

	// handle clicks on menu buttons
	if justClicked {
		mx, my := ebiten.CursorPosition()
		for i, b := range a.buttons {
			if pointInRect(mx, my, b.X, b.Y, b.W, b.H) {
				// Demo: open messagebox with text of clicked button
				a.ShowMessage(fmt.Sprintf("%s clicked", b.Label), func() { log.Printf("closed message for #%d", i) })
				return nil
			}
		}
		// language box click
		if pointInRect(mx, my, a.langBoxX, a.langBoxY, a.langBoxS, a.langBoxS) {
			if a.lang == "en" {
				a.lang = "ru"
			} else {
				a.lang = "en"
			}
			a.refreshButtons()
			return nil
		}
	}

	return nil
}

func (a *App) animateMessage() {
	// basic animation: linear scale and fade (scale 0->1 opening, 1->0 closing)
	const dt = 1.0 / 60.0
	const speed = 6.0 // how fast the tween goes
	if a.msg.Opening {
		a.msg.Scale += speed * dt
		if a.msg.Scale >= 1.0 {
			a.msg.Scale = 1.0
			a.msg.Animating = false
		}
	} else {
		a.msg.Scale -= speed * dt
		if a.msg.Scale <= 0.0 {
			a.msg.Scale = 0.0
			a.msg.Animating = false
			a.msg.Open = false
			// call OnClose if set
			if a.msg.OnClose != nil {
				a.msg.OnClose()
			}
		}
	}
}

func (a *App) ShowMessage(msg string, onClose func()) {
	a.msg.Text = msg
	a.msg.Open = true
	a.msg.Opening = true
	a.msg.Animating = true
	a.msg.Scale = 0.0
	a.msg.OnClose = onClose
}

func (a *App) refreshButtons() {
	// update labels and re-render button images if needed
	labels := []string{t(a.lang, "menu.play"), t(a.lang, "menu.editor"), t(a.lang, "menu.settings"), t(a.lang, "menu.exit")}
	for i := range a.buttons {
		a.buttons[i].Label = labels[i]
		// we don't need to regenerate image for color change, but if palette changes do it here
	}
}

func (a *App) Draw(screen *ebiten.Image) {
	// clear background
	screen.Fill(a.pal.Bg)

	// draw centered menu buttons
	for _, b := range a.buttons {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(b.X), float64(b.Y))
		screen.DrawImage(b.Image, op)

		// button label (text) centered
		textX := b.X + b.W/2 - len(b.Label)*4
		textY := b.Y + b.H/2 + 6
		text.Draw(screen, b.Label, basicfont.Face7x13, textX, textY, a.pal.ButtonText)

		// outline (for strong contour) — draw thin stroke rectangle slightly larger
		// ebitenutilDrawRectStroke(screen, float64(b.X), float64(b.Y), float64(b.W), float64(b.H), 2, a.pal.ButtonStroke)
	}

	// language box bottom-left (square)
	// square background
	langImg := renderRoundedRect(a.langBoxS, a.langBoxS, 8, a.pal.ButtonFill, a.pal.ButtonStroke, 2)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(a.langBoxX), float64(a.langBoxY))
	screen.DrawImage(langImg, op)
	text.Draw(screen, t(a.lang, "lang"), basicfont.Face7x13, a.langBoxX+16, a.langBoxY+a.langBoxS/2+4, a.pal.ButtonText)
	// small label
	text.Draw(screen, "Lang", basicfont.Face7x13, a.langBoxX+6, a.langBoxY-6, a.pal.MenuText)

	// version on bottom-right
	ver := t(a.lang, "version")
	text.Draw(screen, ver, basicfont.Face7x13, a.w-160, a.h-24, a.pal.MenuText)

	// if message box open -> draw overlay and modal
	if a.msg.Open || a.msg.Animating {
		// dim background
		// draw full-screen translucent rectangle
		overlay := ebiten.NewImage(a.w, a.h)
		overlay.Fill(a.pal.ModalBg)
		screen.DrawImage(overlay, nil)

		// modal rectangle centered with scale
		mw, mh := 520, 220
		scale := a.msg.Scale
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
		mx := (a.w - currW) / 2
		my := (a.h - currH) / 2

		// render a rounded rect for modal
		modalImg := renderRoundedRect(currW, currH, 12, color.RGBA{0xff, 0xff, 0xff, 0xff}, color.RGBA{0xcc, 0xcc, 0xcc, 0xff}, 2)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(mx), float64(my))
		screen.DrawImage(modalImg, op)

		// draw message text and OK button (only if fully opened)
		if scale > 0.85 {
			// text centered
			text.Draw(screen, a.msg.Text, basicfont.Face7x13, mx+32, my+60, color.Black)
			// OK button
			okW, okH := 120, 44
			okX := mx + (currW-okW)/2
			okY := my + currH - 56
			okImg := renderRoundedRect(okW, okH, 10, a.pal.Accent, a.pal.ButtonStroke, 2)
			op2 := &ebiten.DrawImageOptions{}
			op2.GeoM.Translate(float64(okX), float64(okY))
			screen.DrawImage(okImg, op2)
			text.Draw(screen, t(a.lang, "button.ok"), basicfont.Face7x13, okX+36, okY+28, color.White)
		}
	}
}

// ---- helpers ----

func renderRoundedRect(w, h, radius int, fill color.RGBA, stroke color.RGBA, strokeW float64) *ebiten.Image {
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

func ebitenutilDrawRectStroke(screen *ebiten.Image, x, y, w, h float64, sw float64, c color.RGBA) {
	// draw four thin rects to simulate stroke slightly outside
	ebitenutilDrawRect(screen, x-sw/2, y-sw/2, w+sw, sw, c)   // top
	ebitenutilDrawRect(screen, x-sw/2, y+h-sw/2, w+sw, sw, c) // bottom
	ebitenutilDrawRect(screen, x-sw/2, y, sw, h, c)           // left
	ebitenutilDrawRect(screen, x+w-sw/2, y, sw, h, c)         // right
}

func ebitenutilDrawRect(screen *ebiten.Image, x, y, w, h float64, c color.RGBA) {
	img := ebiten.NewImage(int(w), int(h))
	img.Fill(c)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y)
	screen.DrawImage(img, op)
}

func pointInRect(px, py, rx, ry, rw, rh int) bool {
	return px >= rx && px < rx+rw && py >= ry && py < ry+rh
}

func (a *App) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return a.w, a.h
}

// ---- main ----
func main() {
	app := NewApp()
	ebiten.SetWindowSize(app.w, app.h)
	ebiten.SetWindowTitle("EvilChess — Menu Demo")
	if err := ebiten.RunGame(app); err != nil {
		log.Fatal(err)
	}
}
