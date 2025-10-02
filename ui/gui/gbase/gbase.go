package gbase

import (
	"errors"
	"image/color"
)

// ---- Exit Call ----

var ErrExit = errors.New("exit request")

// --- UI constants ---

const (
	// SquareSize  = 64
	// BoardMargin = 12
	// RightPanelW = 320
	// HeaderH     = 60
	// BottomH     = 120
	// WindowPadW  = 40
	// WindowPadH  = 40
	AppBgR      = 90  //0xf0
	AppBgG      = 120 //0xf0
	AppBgB      = 130 //0xf0
	WindowW int = 1000
	WindowH int = 700
)

// -- GUI Chess Clock --

type GUIChessClock int

const (
	ClockNull GUIChessClock = iota
	Clock1Min
	Clock3Min
	Clock5Min
	Clock10Min
	Clock15Min
)

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

func (p Palette) String() string {
	switch p {
	case LightPalette:
		return "light"
	case DarkPalette:
		return "dark"
	default:
	}
	return ""
}

func PaletteFromString(p string) Palette {
	switch p {
	case "light":
		return LightPalette
	case "dark":
		return DarkPalette
	default:
	}
	return Palette{}
}

var LightPalette = Palette{
	Bg:         color.RGBA{0xf7, 0xf7, 0xf7, 0xff},
	ButtonFill: color.RGBA{0xff, 0xff, 0xff, 0xff},
	// ButtonStroke: color.RGBA{0xd0, 0xd6, 0xdb, 0xff},
	// ButtonStroke: color.RGBA{0x44, 0x44, 0x44, 0xff},
	ButtonStroke: color.RGBA{0x88, 0x88, 0x88, 0xff},
	ButtonText:   color.RGBA{0x22, 0x22, 0x22, 0xff},
	MenuText:     color.RGBA{0x22, 0x22, 0x22, 0xff},
	Accent:       color.RGBA{0x22, 0x88, 0xcc, 0xff},
	ModalBg:      color.RGBA{0x00, 0x00, 0x00, 0x88},
}

var DarkPalette = Palette{
	Bg:         color.RGBA{0x12, 0x12, 0x12, 0xff},
	ButtonFill: color.RGBA{0x20, 0x20, 0x20, 0xff},
	// ButtonStroke: color.RGBA{0x40, 0x40, 0x40, 0xff},
	ButtonStroke: color.RGBA{0xdd, 0xdd, 0xdd, 0xff},
	ButtonText:   color.RGBA{0xee, 0xee, 0xee, 0xff},
	MenuText:     color.RGBA{0xee, 0xee, 0xee, 0xff},
	Accent:       color.RGBA{0x2a, 0xa1, 0xd1, 0xff},
	ModalBg:      color.RGBA{0x00, 0x00, 0x00, 0x99},
}
