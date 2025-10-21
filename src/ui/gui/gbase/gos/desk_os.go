//go:build !js && !wasm
// +build !js,!wasm

package gos

import (
	"os"
)

func Stat(name string) (FileInfo, error) {
	fi, err := os.Stat(name)
	if err != nil {
		return FileInfo{}, err
	}
	return FileInfo{
		Name:    fi.Name(),
		Size:    fi.Size(),
		ModTime: fi.ModTime(),
		IsDir:   fi.IsDir(),
	}, nil
}

func Open(name string) (ReadCloser, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func Remove(name string) error {
	return os.Remove(name)
}

func IsNotExist(err error) bool {
	return os.IsNotExist(err)
}
