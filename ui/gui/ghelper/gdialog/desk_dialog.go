//go:build !js && !wasm
// +build !js,!wasm

package gdialog

import (
	"os"
	"path/filepath"

	"github.com/sqweek/dialog"
)

type Result struct {
	Path string
	Name string // empty
	Data []byte // empty
}

func OpenFile(title string) (Result, error) {
	path, err := dialog.File().Title(title).Load()
	if err != nil {
		return Result{}, err
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return Result{}, err
	}

	return Result{
		Path: path,
		Name: filepath.Base(path),
		Data: b,
	}, nil
}
