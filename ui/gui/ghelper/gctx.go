package ghelper

import (
	"evilchess/src"
	"evilchess/src/logx"
	"evilchess/ui/gui/gbase"
	"evilchess/ui/gui/gbase/gconf"
)

// --- GUI Game Types ---

type GUIGameType struct {
	IsGameWithEngine bool // if false then game with yourself
	ClockType        gbase.GUIChessClock
}

// ---- GUI Context ----

type GUIGameContext struct {
	Builder      *src.GameBuilder
	AssetsWorker *GUIAssetsWorker
	GameType     *GUIGameType
	Config       *gconf.Config
	Theme        gbase.Palette
	Logx         logx.Logger
	IsReady      bool
}

func NewGUIGameContext(b *src.GameBuilder, a *GUIAssetsWorker, c *gconf.Config, l logx.Logger) *GUIGameContext {
	return &GUIGameContext{
		Builder:      b,
		AssetsWorker: a,
		Config:       c,
		Theme:        gbase.PaletteFromString(c.Theme),
		Logx:         l,
	}
}
