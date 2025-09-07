package main

import (
	"image/color"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var figureImages []*ebiten.Image

func loadFigures() {
	files := []string{"../assets/bking60.png", "../assets/wqueen60.png", "../assets/bbishop60.png", "../assets/wknight60.png", "../assets/brook60.png", "../assets/wpawn60.png"}
	for _, f := range files {
		img, _, err := ebitenutil.NewImageFromFile(f)
		if err != nil {
			log.Fatal(err)
		}
		figureImages = append(figureImages, img)
	}
}

type Bubble struct {
	x, y   float64
	vx, vy float64
	radius float64
	figure *ebiten.Image
}

type Game struct {
	bubbles []*Bubble
}

func NewGame() *Game {
	rand.Seed(time.Now().UnixNano())
	loadFigures()
	bubbles := make([]*Bubble, 20)
	for i := range bubbles {
		bubbles[i] = &Bubble{
			x:      rand.Float64() * 640,
			y:      rand.Float64() * 480,
			vx:     (rand.Float64() - 0.5) * 2,
			vy:     (rand.Float64() - 0.5) * 2,
			radius: 20 + rand.Float64()*20,
			figure: figureImages[rand.Intn(len(figureImages))],
		}
	}
	return &Game{bubbles: bubbles}
}

func (g *Game) Update() error {
	for i, b := range g.bubbles {
		b.x += b.vx
		b.y += b.vy

		// Отражение от стен
		if b.x < b.radius {
			b.x = b.radius
			b.vx = -b.vx
		}
		if b.x > 640-b.radius {
			b.x = 640 - b.radius
			b.vx = -b.vx
		}
		if b.y < b.radius {
			b.y = b.radius
			b.vy = -b.vy
		}
		if b.y > 480-b.radius {
			b.y = 480 - b.radius
			b.vy = -b.vy
		}

		// Столкновения
		for j := i + 1; j < len(g.bubbles); j++ {
			b2 := g.bubbles[j]
			dx := b2.x - b.x
			dy := b2.y - b.y
			dist := math.Hypot(dx, dy)
			minDist := b.radius + b2.radius
			if dist < minDist && dist > 0 {
				overlap := 0.5 * (minDist - dist)
				nx := dx / dist
				ny := dy / dist

				b.x -= overlap * nx
				b.y -= overlap * ny
				b2.x += overlap * nx
				b2.y += overlap * ny

				dot1 := b.vx*nx + b.vy*ny
				dot2 := b2.vx*nx + b2.vy*ny

				b.vx += (dot2 - dot1) * nx
				b.vy += (dot2 - dot1) * ny
				b2.vx += (dot1 - dot2) * nx
				b2.vy += (dot1 - dot2) * ny
			}
		}
	}
	return nil
}

func drawBubble(screen *ebiten.Image, x, y, radius float64) {
	// Центр - светло-голубой почти белый с высокой прозрачностью
	centerColor := color.RGBA{200, 230, 255, 150}
	// Край - более прозрачный синий
	edgeColor := color.RGBA{100, 150, 255, 20}

	// Рисуем концентрические круги с уменьшением альфа и изменением цвета
	steps := 8
	for i := 0; i < steps; i++ {
		r := radius * (1 - float64(i)/float64(steps))
		alpha := uint8(150 - i*15)
		c := color.RGBA{
			R: uint8(int(centerColor.R) + int(edgeColor.R-centerColor.R)*i/steps),
			G: uint8(int(centerColor.G) + int(edgeColor.G-centerColor.G)*i/steps),
			B: uint8(int(centerColor.B) + int(edgeColor.B-centerColor.B)*i/steps),
			A: alpha,
		}
		ebitenutil.DrawCircle(screen, x, y, r, c)
	}

	// Добавляем блик у верхнего левого края пузыря
	highlightColor := color.RGBA{255, 255, 255, 200}
	ebitenutil.DrawCircle(screen, x-radius/3, y-radius/3, radius/4, highlightColor)
}

// func (g *Game) Draw(screen *ebiten.Image) {
// 	screen.Fill(color.RGBA{135, 206, 250, 255}) // голубой фон

// 	for _, b := range g.bubbles {
// 		// Тень (смещенный темный полупрозрачный круг)
// 		shadowColor := color.RGBA{0, 0, 0, 50}
// 		ebitenutil.DrawCircle(screen, b.x+3, b.y+3, b.radius, shadowColor)

// 		// Пузырь (полупрозрачный белый)
// 		// bubbleColor := color.RGBA{255, 255, 255, 80}
// 		// ebitenutil.DrawCircle(screen, b.x, b.y, b.radius, bubbleColor)
// 		drawBubble(screen, b.x, b.y, b.radius)

//			// Блик (яркий маленький круг)
//			highlightColor := color.RGBA{255, 255, 255, 150}
//			ebitenutil.DrawCircle(screen, b.x-b.radius/3, b.y-b.radius/3, b.radius/4, highlightColor)
//		}
//	}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{135, 206, 250, 255}) // фон

	for _, b := range g.bubbles {
		// Тень пузыря (можно оставить или убрать)
		shadowColor := color.RGBA{0, 0, 0, 50}
		ebitenutil.DrawCircle(screen, b.x+3, b.y+3, b.radius, shadowColor)

		// Рисуем фигуру с масштабированием под размер пузыря
		op := &ebiten.DrawImageOptions{}
		scale := (2 * b.radius) / float64(b.figure.Bounds().Dx()) // масштабируем по радиусу
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(b.x-b.radius, b.y-b.radius)
		op.ColorM.Scale(1, 1, 1, 0.8) // прозрачность

		screen.DrawImage(b.figure, op)

		// Можно добавить блик поверх, если нужно
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 640, 480
}

func main() {
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowTitle("Soap Bubbles with Shadows")

	game := NewGame()
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
