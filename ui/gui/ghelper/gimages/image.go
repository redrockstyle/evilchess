package gimages

import (
	"evilchess/src/base"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

func LoadImageAssets(workdir string) (map[base.Piece]*ebiten.Image, error) {
	files := []string{
		workdir + "/wking60.png",   // 0
		workdir + "/bking60.png",   // 1
		workdir + "/wqueen60.png",  // 2
		workdir + "/bqueen60.png",  // 3
		workdir + "/wbishop60.png", // 4
		workdir + "/bbishop60.png", // 5
		workdir + "/wknight60.png", // 6
		workdir + "/bknight60.png", // 7
		workdir + "/wrook60.png",   // 8
		workdir + "/brook60.png",   // 9
		workdir + "/wpawn60.png",   // 10
		workdir + "/bpawn60.png",   // 11
	}
	keys := []base.Piece{
		base.WKing,
		base.BKing,
		base.WQueen,
		base.BQueen,
		base.WBishop,
		base.BBishop,
		base.WKnight,
		base.BKnight,
		base.WRook,
		base.BRook,
		base.WPawn,
		base.BPawn,
		base.InvalidPiece,
	}
	figureImages := make(map[base.Piece]*ebiten.Image)
	for i := 0; i < 12; i++ {
		img, _, err := ebitenutil.NewImageFromFile(files[i])
		if err != nil {
			return nil, err
		}
		figureImages[keys[i]] = img
	}
	return figureImages, nil
}
