package ghelper

import (
	"image/color"
	"math"

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

func EbitenutilDrawRectStroke(screen *ebiten.Image, x, y, w, h, thickness float64, col color.Color) {
	if screen == nil || w <= 0 || h <= 0 || thickness <= 0 {
		return
	}

	maxTh := math.Min(w, h) / 2.0
	if thickness > maxTh {
		thickness = maxTh
	}

	px := ebiten.NewImage(1, 1)
	px.Fill(col)

	// up
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(w, thickness)
	op.GeoM.Translate(x, y)
	screen.DrawImage(px, op)

	// down
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(w, thickness)
	op.GeoM.Translate(x, y+h-thickness)
	screen.DrawImage(px, op)

	// left
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(thickness, h-thickness*2)
	op.GeoM.Translate(x, y+thickness)
	screen.DrawImage(px, op)

	// right
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(thickness, h-thickness*2)
	op.GeoM.Translate(x+w-thickness, y+thickness)
	screen.DrawImage(px, op)
}
