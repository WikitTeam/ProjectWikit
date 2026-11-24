// Package storage keeps the uploaded files, on disk or in an S3 bucket.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"
	"time"
)

var (
	ErrNotExist = fs.ErrNotExist
	ErrBadKey   = errors.New("storage: bad key")
)

type Info struct {
	Size    int64
	ModTime time.Time
}

type File interface {
	io.ReadSeekCloser
	Info() Info
}

// Keys are slash-separated paths relative to the store's root, the same ones the
// files directory has always used, so moving between backends keeps every name.
type Store interface {
	Open(ctx context.Context, key string) (File, error)
	Stat(ctx context.Context, key string) (Info, error)
	Put(ctx context.Context, key string, r io.Reader, size int64) error
	Remove(ctx context.Context, key string) error
	RemoveAll(ctx context.Context, prefix string) error
	Walk(ctx context.Context, prefix string, fn func(key string, info Info) error) error
}

func CheckKey(key string) error {
	if key == "" || strings.HasPrefix(key, "/") || strings.Contains(key, "\\") || path.Clean(key) != key {
		return fmt.Errorf("%w: %q", ErrBadKey, key)
	}
	for _, part := range strings.Split(key, "/") {
		if part == ".." || part == "." {
			return fmt.Errorf("%w: %q", ErrBadKey, key)
		}
	}
	return nil
}

func Join(parts ...string) string {
	return strings.Join(parts, "/")
}

func ReadAll(ctx context.Context, s Store, key string) ([]byte, error) {
	f, err := s.Open(ctx, key)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func Exists(ctx context.Context, s Store, key string) bool {
	_, err := s.Stat(ctx, key)
	return err == nil
}
