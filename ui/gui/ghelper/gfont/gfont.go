package gfont

import (
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

type Fonts struct {
	Pixel    font.Face
	PixelLow font.Face
	Normal   font.Face
	Bold     font.Face
}

func LoadFonts(workdir string) (*Fonts, error) {
	var err error

	// read ttf
	ps2p, err := os.ReadFile(workdir + "/PressStart2P-Regular.ttf")
	if err != nil {
		return nil, err
	}
	f, err := opentype.Parse(ps2p)
	if err != nil {
		return nil, err
	}

	fonts := &Fonts{}
	// create face (size 18px)
	fonts.Pixel, err = opentype.NewFace(f, &opentype.FaceOptions{
		Size:    12,
		DPI:     72,
		Hinting: font.HintingNone,
	})
	if err != nil {
		return nil, err
	}

	fonts.PixelLow, err = opentype.NewFace(f, &opentype.FaceOptions{
		Size:    10,
		DPI:     72,
		Hinting: font.HintingNone,
	})
	if err != nil {
		return nil, err
	}

	// for titles
	fonts.Bold, err = opentype.NewFace(f, &opentype.FaceOptions{
		Size:    16,
		DPI:     72,
		Hinting: font.HintingNone,
	})
	if err != nil {
		return nil, err
	}

	// read ttf
	nsd, err := os.ReadFile(workdir + "/NotoSansDisplay-Regular.ttf")
	if err != nil {
		return nil, err
	}
	f, err = opentype.Parse(nsd)
	if err != nil {
		return nil, err
	}

	fonts.Normal, err = opentype.NewFace(f, &opentype.FaceOptions{
		Size:    13,
		DPI:     72,
		Hinting: font.HintingNone,
	})
	if err != nil {
		return nil, err
	}

	return fonts, nil
}
