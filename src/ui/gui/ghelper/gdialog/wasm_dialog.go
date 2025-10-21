//go:build js && wasm
// +build js,wasm

package gdialog

import (
	"errors"
	"syscall/js"
)

type Result struct {
	Path string // empty
	Name string
	Data []byte
}

// <input type="file">, wait and read to []byte.
func OpenFile(title string) (Result, error) {
	doc := js.Global().Get("document")
	if !doc.Truthy() {
		return Result{}, errors.New("document not available")
	}

	ch := make(chan struct {
		res Result
		err error
	}, 1)

	input := doc.Call("createElement", "input")
	input.Set("type", "file")
	// input.Set("accept", ".wasm,.bin")

	// onchange dispath
	onchange := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		files := input.Get("files")
		if files.Length() == 0 {
			ch <- struct {
				res Result
				err error
			}{Result{}, errors.New("no file selected")}
			return nil
		}

		file := files.Index(0)
		name := file.Get("name").String()
		reader := js.Global().Get("FileReader").New()

		onload := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			arrayBuf := reader.Get("result")
			uint8Arr := js.Global().Get("Uint8Array").New(arrayBuf)
			n := uint8Arr.Get("length").Int()
			data := make([]byte, n)
			read := js.CopyBytesToGo(data, uint8Arr)
			_ = read // read == n
			ch <- struct {
				res Result
				err error
			}{Result{Path: "", Name: name, Data: data}, nil}

			// free
			// onload.Release()
			// onerror.Release()
			// onchange.Release()
			return nil
		})

		onerror := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			ch <- struct {
				res Result
				err error
			}{Result{}, errors.New("failed to read file")}
			// onload.Release()
			// onerror.Release()
			// onchange.Release()
			return nil
		})

		reader.Set("onload", onload)
		reader.Set("onerror", onerror)
		reader.Call("readAsArrayBuffer", file)
		return nil
	})

	input.Set("onchange", onchange)

	// add to DOM, then click()
	body := doc.Get("body")
	if !body.Truthy() {
		return Result{}, errors.New("document.body not available")
	}
	body.Call("appendChild", input)
	input.Call("click")

	// wait
	r := <-ch

	// cleanup: delete input from DOM (is exists)
	body.Call("removeChild", input)

	return r.res, r.err
}
