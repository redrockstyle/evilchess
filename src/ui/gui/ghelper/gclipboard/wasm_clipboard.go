//go:build js && wasm
// +build js,wasm

package gclipboard

import (
	"errors"
	"syscall/js"
)

func ReadAll() (string, error) {
	global := js.Global()
	nav := global.Get("navigator")
	if !nav.Truthy() {
		return "", errors.New("navigator not available")
	}

	clipboard := nav.Get("clipboard")
	if clipboard.Truthy() && clipboard.Get("readText").Truthy() {
		promise := clipboard.Call("readText")

		type res struct {
			text string
			err  error
		}
		ch := make(chan res, 1)

		then := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			// args[0] - результат readText()
			ch <- res{args[0].String(), nil}
			return nil
		})
		catch := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			ch <- res{"", errors.New(args[0].String())}
			return nil
		})

		promise.Call("then", then).Call("catch", catch)
		r := <-ch
		then.Release()
		catch.Release()
		return r.text, r.err
	}

	return "", errors.New("navigator.clipboard.readText not available")
}

func WriteAll(text string) error {
	global := js.Global()
	nav := global.Get("navigator")
	doc := global.Get("document")

	// 1) try navigator.clipboard.writeText
	if nav.Truthy() && nav.Get("clipboard").Truthy() && nav.Get("clipboard").Get("writeText").Truthy() {
		promise := nav.Get("clipboard").Call("writeText", text)

		ch := make(chan error, 1)
		then := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			ch <- nil
			return nil
		})
		catch := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			// args[0] ???
			var msg string
			if args != nil && len(args) > 0 {
				msg = args[0].String()
			}
			ch <- errors.New(msg)
			return nil
		})

		promise.Call("then", then).Call("catch", catch)
		err := <-ch
		then.Release()
		catch.Release()
		return err
	}

	// 2) fallback: create textarea, select, execCommand("copy")
	if doc.Truthy() && doc.Get("createElement").Truthy() && doc.Get("execCommand").Truthy() {
		ta := doc.Call("createElement", "textarea")
		ta.Get("style").Set("position", "fixed")
		ta.Get("style").Set("left", "-10000px")
		ta.Set("value", text)

		// append -> select -> execCommand -> remove
		body := doc.Get("body")
		if !body.Truthy() {
			return errors.New("document.body not available for fallback copy")
		}
		body.Call("appendChild", ta)
		ta.Call("select")

		success := doc.Call("execCommand", "copy").Bool()

		body.Call("removeChild", ta)

		if success {
			return nil
		}
		return errors.New("fallback copy failed (execCommand returned false)")
	}

	return errors.New("clipboard write not available")
}
