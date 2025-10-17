package gimages

import (
	"evilchess/src/base"
	"evilchess/ui/gui/gbase/gos"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

func loadIcon(path string) (image.Image, error) {
	f, err := gos.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func LoadIconAssets(workdir string) (map[int]image.Image, error) {
	files := []string{
		// pieces
		workdir + "/crown16.png", // 0
		workdir + "/crown32.png", // 1
		workdir + "/crown48.png", // 2
		workdir + "/crown60.png", // 3
	}
	keys := []int{
		16,
		32,
		48,
		60,
	}
	icons := make(map[int]image.Image)
	for i := 0; i < 4; i++ {
		img, err := loadIcon(files[i])
		if err != nil {
			return nil, err
		}
		icons[keys[i]] = img
	}
	return icons, nil
}

func LoadImageIconAssets(workdir string) (map[int]*ebiten.Image, error) {
	files := []string{
		// pieces
		workdir + "/crown16.png", // 0
		workdir + "/crown32.png", // 1
		workdir + "/crown48.png", // 2
		workdir + "/crown60.png", // 3
	}
	keys := []int{
		16,
		32,
		48,
		60,
	}
	icons := make(map[int]*ebiten.Image)
	for i := 0; i < 4; i++ {
		img, _, err := ebitenutil.NewImageFromFile(files[i])
		if err != nil {
			return nil, err
		}
		icons[keys[i]] = img
	}
	return icons, nil
}

func LoadImagePieceAssets(workdir string) (map[base.Piece]*ebiten.Image, error) {
	files := []string{
		// pieces
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
