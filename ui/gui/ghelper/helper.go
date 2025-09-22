package ghelper

import (
	"evilchess/ui/gui/gbase"
	"image/color"

	"github.com/fogleman/gg"
	"github.com/hajimehoshi/ebiten/v2"
)

func RenderRoundedRect(w, h, radius int, fill color.RGBA, stroke color.RGBA, strokeW float64) *ebiten.Image {
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

func EbitenutilDrawRect(screen *ebiten.Image, x, y, w, h float64, c color.RGBA) {
	img := ebiten.NewImage(int(w), int(h))
	img.Fill(c)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y)
	screen.DrawImage(img, op)
}

func PointInRect(px, py, rx, ry, rw, rh int) bool {
	return px >= rx && px < rx+rw && py >= ry && py < ry+rh
}

func AnimateMessage(box *gbase.MessageBox) {
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

func ShowMessage(box *gbase.MessageBox, msg string, onClose func()) {
	box.Text = msg
	box.Open = true
	box.Opening = true
	box.Animating = true
	box.Scale = 0.0
	box.OnClose = onClose
}
