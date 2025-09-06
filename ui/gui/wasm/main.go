package main

import (
	"evilchess/src/base"
	"evilchess/ui/gui"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	g := gui.NewGameFromFEN(base.FEN_START_GAME)
	app := gui.NewApp(g)

	// ebiten.SetWindowSize(800, 800)
	ebiten.SetWindowTitle("EvilChess")

	ebiten.RunGame(app)
}
