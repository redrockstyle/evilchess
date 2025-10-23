package gassets

import (
	"embed"
	"evilchess/src/ui/gui/gbase/gos"
)

//go:embed assets/**
var embeddedAssets embed.FS

func ReadAsset(path string) ([]byte, error) {
	if _, err := gos.Stat(path); err == nil {
		return gos.ReadFile(path)
	}
	return embeddedAssets.ReadFile(path)
}

func OpenAsset(path string) (gos.ReadCloser, error) {
	if r, err := gos.Open(path); err == nil {
		return r, nil
	}

	return embeddedAssets.Open(path)
}
