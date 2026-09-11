// Package logfile answers how pwikit keeps its own log a bounded size.
package logfile

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
)

const (
	DefaultLimit = 10 << 20
	DefaultKeep  = 5
)

type Writer struct {
	mu    sync.Mutex
	path  string
	limit int64
	keep  int
	file  *os.File
	size  int64
}

func Open(path string, limit int64, keep int) (*Writer, error) {
	w := &Writer{path: path, limit: limit, keep: keep}
	if err := w.open(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *Writer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return 0, os.ErrClosed
	}
	if w.size > 0 && w.size+int64(len(p)) > w.limit {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}
	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

func (w *Writer) open() error {
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	w.file, w.size = f, info.Size()
	return nil
}

// Windows refuses to rename an open file.
func (w *Writer) rotate() error {
	if err := w.file.Close(); err != nil {
		return err
	}
	w.file = nil
	if err := os.Remove(w.numbered(w.keep)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for i := w.keep - 1; i >= 1; i-- {
		if err := os.Rename(w.numbered(i), w.numbered(i+1)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if err := os.Rename(w.path, w.numbered(1)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return w.open()
}

func (w *Writer) numbered(i int) string {
	return w.path + "." + strconv.Itoa(i)
}
