package gos

import (
	"errors"
	"io"
	"time"
)

var ErrNotExist = errors.New("file does not exist (oswrap)")

// FileInfo
type FileInfo struct {
	Name    string
	Size    int64
	ModTime time.Time
	IsDir   bool
}

// OpenFunc returns io.ReadCloser
type ReadCloser = io.ReadCloser

// Stat(name) (FileInfo, error)
// Open(name) (ReadCloser, error)
// ReadFile(name) ([]byte, error)
// WriteFile(name []byte, perm os.FileMode) error
// Remove(name) error
// IsNotExist(err error) bool
