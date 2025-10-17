//go:build js && wasm
// +build js,wasm

package gos

import (
	"bytes"
	"errors"
	"io"
	"syscall/js"
	"time"
)

// fetchBytes выполняет JS fetch() и возвращает []byte.
func fetchBytes(path string) ([]byte, error) {
	// Создаём промис: fetch(path).then(r => r.arrayBuffer())
	global := js.Global()
	fetch := global.Get("fetch")
	if !fetch.Truthy() {
		return nil, errors.New("fetch() not supported")
	}

	promise := fetch.Invoke(path)
	ch := make(chan struct {
		data []byte
		err  error
	}, 1)

	thenFn := js.FuncOf(func(this js.Value, args []js.Value) any {
		resp := args[0]
		if !resp.Get("ok").Bool() {
			ch <- struct {
				data []byte
				err  error
			}{nil, ErrNotExist}
			return nil
		}
		// Преобразуем в ArrayBuffer
		arrayBufferPromise := resp.Call("arrayBuffer")
		arrayBufferPromise.Call("then",
			js.FuncOf(func(this js.Value, args []js.Value) any {
				arrayBuffer := args[0]
				uint8Array := js.Global().Get("Uint8Array").New(arrayBuffer)
				length := uint8Array.Get("length").Int()
				data := make([]byte, length)
				js.CopyBytesToGo(data, uint8Array)
				ch <- struct {
					data []byte
					err  error
				}{data, nil}
				return nil
			}),
			js.FuncOf(func(this js.Value, args []js.Value) any {
				ch <- struct {
					data []byte
					err  error
				}{nil, errors.New("failed to read arrayBuffer")}
				return nil
			}),
		)
		return nil
	})

	catchFn := js.FuncOf(func(this js.Value, args []js.Value) any {
		ch <- struct {
			data []byte
			err  error
		}{nil, errors.New("fetch() failed")}
		return nil
	})

	promise.Call("then", thenFn).Call("catch", catchFn)
	result := <-ch
	thenFn.Release()
	catchFn.Release()

	return result.data, result.err
}

// Stat имитирует метаданные (размер, время не определено).
func Stat(name string) (FileInfo, error) {
	data, err := fetchBytes(name)
	if err != nil {
		return FileInfo{}, err
	}
	return FileInfo{
		Name:    name,
		Size:    int64(len(data)),
		ModTime: time.Time{},
		IsDir:   false,
	}, nil
}

// Open возвращает io.ReadCloser для данных, загруженных через fetch().
type wasmFile struct {
	r *bytes.Reader
}

func (f *wasmFile) Read(p []byte) (int, error) { return f.r.Read(p) }
func (f *wasmFile) Close() error               { return nil }

func Open(name string) (io.ReadCloser, error) {
	data, err := fetchBytes(name)
	if err != nil {
		return nil, err
	}
	return &wasmFile{r: bytes.NewReader(data)}, nil
}

func ReadFile(name string) ([]byte, error) {
	return fetchBytes(name)
}

// Поскольку браузер не может писать в локальную ФС — функции-заглушки:
func WriteFile(name string, data []byte, _ any) error {
	return errors.New("WriteFile not supported in wasm")
}

func Remove(name string) error {
	return errors.New("Remove not supported in wasm")
}

func IsNotExist(err error) bool {
	return errors.Is(err, ErrNotExist)
}
