package main

import (
	"evilchess/src/base"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	g := NewGameFromFEN(base.FEN_START_GAME)
	app := NewApp(g)

	// ebiten.SetWindowSize(800, 800)
	ebiten.SetWindowTitle("EvilChess")

	ebiten.RunGame(app)
}
