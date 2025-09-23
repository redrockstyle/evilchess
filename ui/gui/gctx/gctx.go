package gctx

import (
	"evilchess/src"
	"evilchess/src/logx"
	"evilchess/ui/gui/gbase"
	"evilchess/ui/gui/gbase/gconf"
	"evilchess/ui/gui/ghelper"
)

// ---- GUI Context ----

type GUIGameContext struct {
	Builder      *src.GameBuilder
	AssetsWorker *ghelper.GUIAssetsWorker
	Config       *gconf.Config
	Theme        gbase.Palette
	Logx         logx.Logger
}

func NewGUIGameContext(b *src.GameBuilder, a *ghelper.GUIAssetsWorker, c *gconf.Config, l logx.Logger) *GUIGameContext {
	return &GUIGameContext{
		Builder:      b,
		AssetsWorker: a,
		Config:       c,
		Theme:        gbase.PaletteFromString(c.Theme),
		Logx:         l,
	}
}
