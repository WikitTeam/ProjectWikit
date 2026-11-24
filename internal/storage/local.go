package storage

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/paths"
)

type Local struct {
	root string
}

var _ Store = (*Local)(nil)

func NewLocal(root string) *Local {
	return &Local{root: root}
}

func (l *Local) Root() string { return l.root }

func (l *Local) full(key string) (string, error) {
	if err := CheckKey(key); err != nil {
		return "", err
	}
	return paths.Resolve(l.root, filepath.FromSlash(key))
}

type localFile struct {
	*os.File
	info Info
}

func (f localFile) Info() Info { return f.info }

func (l *Local) Open(_ context.Context, key string) (File, error) {
	full, err := l.full(key)
	if err != nil {
		return nil, err
	}
	f, err := paths.In(l.root).Open(full)
	if err != nil {
		return nil, err
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if st.IsDir() {
		f.Close()
		return nil, ErrNotExist
	}
	return localFile{File: f, info: Info{Size: st.Size(), ModTime: st.ModTime()}}, nil
}

func (l *Local) Stat(ctx context.Context, key string) (Info, error) {
	f, err := l.Open(ctx, key)
	if err != nil {
		return Info{}, err
	}
	defer f.Close()
	return f.Info(), nil
}

// The bytes land in a temporary file beside the target first, so a reader
// never sees half a file and a failed write leaves the old one in place.
func (l *Local) Put(_ context.Context, key string, r io.Reader, _ int64) error {
	full, err := l.full(key)
	if err != nil {
		return err
	}
	jail := paths.In(l.root)
	if err := jail.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	temp := full + ".part"
	out, err := jail.Create(temp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, r); err != nil {
		out.Close()
		jail.Remove(temp)
		return err
	}
	if err := out.Close(); err != nil {
		jail.Remove(temp)
		return err
	}
	if err := jail.Rename(temp, full); err != nil {
		jail.Remove(temp)
		return err
	}
	return nil
}

func (l *Local) Remove(_ context.Context, key string) error {
	full, err := l.full(key)
	if err != nil {
		return err
	}
	err = paths.In(l.root).Remove(full)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

func (l *Local) RemoveAll(_ context.Context, prefix string) error {
	full, err := l.full(strings.TrimSuffix(prefix, "/"))
	if err != nil {
		return err
	}
	return paths.In(l.root).RemoveAll(full)
}

func (l *Local) Walk(ctx context.Context, prefix string, fn func(key string, info Info) error) error {
	start := l.root
	if prefix = strings.TrimSuffix(prefix, "/"); prefix != "" {
		full, err := l.full(prefix)
		if err != nil {
			return err
		}
		start = full
	}
	err := filepath.WalkDir(start, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if !entry.Type().IsRegular() || strings.HasSuffix(name, ".part") {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(l.root, name)
		if err != nil {
			return err
		}
		return fn(filepath.ToSlash(rel), Info{Size: info.Size(), ModTime: info.ModTime()})
	})
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}
