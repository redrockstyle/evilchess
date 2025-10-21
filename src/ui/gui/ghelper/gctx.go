package ghelper

import (
	"evilchess/src/chesslib"
	"evilchess/src/logx"
	"evilchess/src/ui/gui/gbase"
	"evilchess/src/ui/gui/gbase/gconf"
)

// --- GUI Game Types ---

type GUIGameType struct {
	IsGameWithEngine bool // if false then game with yourself
	ClockType        gbase.GUIChessClock
}

// ---- GUI Context ----

type GUIGameContext struct {
	Builder      *chesslib.GameBuilder
	AssetsWorker *GUIAssetsWorker
	GameType     *GUIGameType
	Config       *gconf.Config
	Theme        gbase.Palette
	Logx         logx.Logger
	IsReady      bool
}

func NewGUIGameContext(b *chesslib.GameBuilder, a *GUIAssetsWorker, c *gconf.Config, l logx.Logger) *GUIGameContext {
	return &GUIGameContext{
		Builder:      b,
		AssetsWorker: a,
		Config:       c,
		Theme:        gbase.PaletteFromString(c.Theme),
		Logx:         l,
	}
}
