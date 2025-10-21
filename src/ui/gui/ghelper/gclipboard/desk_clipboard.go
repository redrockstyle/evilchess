//go:build !js && !wasm
// +build !js,!wasm

package gclipboard

import "github.com/atotto/clipboard"

func ReadAll() (string, error) {
	return clipboard.ReadAll()
}

func WriteAll(text string) error {
	return clipboard.WriteAll(text)
}
