package paths

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Jail struct {
	base string
}

func In(base string) Jail { return Jail{base: base} }

func (j Jail) enter(full string) (*os.Root, string, error) {
	rel, err := filepath.Rel(j.base, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return nil, "", fmt.Errorf("%w: %q", ErrEscapes, full)
	}
	before, err := os.Lstat(j.base)
	if err != nil {
		return nil, "", err
	}
	if before.Mode()&os.ModeSymlink != 0 {
		if !trustedLink(before) {
			return nil, "", fmt.Errorf("%w: %s is a link another account made", ErrEscapes, j.base)
		}
		if before, err = os.Stat(j.base); err != nil {
			return nil, "", err
		}
	}
	if !before.IsDir() {
		return nil, "", fmt.Errorf("%w: %s is not a directory", ErrEscapes, j.base)
	}
	root, err := os.OpenRoot(j.base)
	if err != nil {
		return nil, "", err
	}
	after, err := root.Stat(".")
	if err != nil || !os.SameFile(before, after) {
		root.Close()
		return nil, "", fmt.Errorf("%w: %s changed while it was opened", ErrEscapes, j.base)
	}
	return root, rel, nil
}

func (j Jail) Open(full string) (*os.File, error) {
	return j.OpenFile(full, os.O_RDONLY, 0)
}

func (j Jail) OpenFile(full string, flag int, perm fs.FileMode) (*os.File, error) {
	root, rel, err := j.enter(full)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return root.OpenFile(rel, flag, perm)
}

func (j Jail) Create(full string) (*os.File, error) {
	return j.OpenFile(full, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o666)
}

func (j Jail) ReadFile(full string) ([]byte, error) {
	root, rel, err := j.enter(full)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return root.ReadFile(rel)
}

func (j Jail) WriteFile(full string, data []byte, perm fs.FileMode) error {
	root, rel, err := j.enter(full)
	if err != nil {
		return err
	}
	defer root.Close()
	return root.WriteFile(rel, data, perm)
}

func (j Jail) MkdirAll(full string, perm fs.FileMode) error {
	root, rel, err := j.enter(full)
	if err != nil {
		return err
	}
	defer root.Close()
	if rel == "." {
		return nil
	}
	return root.MkdirAll(rel, perm)
}

func (j Jail) Remove(full string) error {
	root, rel, err := j.enter(full)
	if err != nil {
		return err
	}
	defer root.Close()
	return root.Remove(rel)
}

func (j Jail) RemoveAll(full string) error {
	root, rel, err := j.enter(full)
	if err != nil {
		return err
	}
	defer root.Close()
	return root.RemoveAll(rel)
}

func (j Jail) Rename(from, to string) error {
	root, relFrom, err := j.enter(from)
	if err != nil {
		return err
	}
	defer root.Close()
	relTo, err := filepath.Rel(j.base, to)
	if err != nil || relTo == ".." || strings.HasPrefix(relTo, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%w: %q", ErrEscapes, to)
	}
	return root.Rename(relFrom, relTo)
}
