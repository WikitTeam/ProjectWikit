package update

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func CopyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		switch {
		case d.IsDir():
			return os.MkdirAll(target, info.Mode().Perm()|0o700)
		case info.Mode()&fs.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		case !info.Mode().IsRegular():
			return nil
		case rel == "postmaster.pid":
			return nil
		}
		return copyFile(path, target, info.Mode().Perm())
	})
}

func copyFile(from, to string, mode fs.FileMode) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func CopyFile(from, to string) error {
	info, err := os.Stat(from)
	if err != nil {
		return err
	}
	tmp := to + ".part"
	if err := copyFile(from, tmp, info.Mode().Perm()); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, to)
}

func TreeSize(dir string) (int64, error) {
	var total int64
	err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			total += info.Size()
		}
		return nil
	})
	return total, err
}

// Windows refuses to overwrite a running program but lets it be renamed, and
// the updater is itself that program.
func Replace(current, next, keep string) error {
	os.Remove(keep)
	if err := os.Rename(current, keep); err != nil {
		return fmt.Errorf("move %s aside: %w", current, err)
	}
	if err := os.Rename(next, current); err != nil {
		if restoreErr := os.Rename(keep, current); restoreErr != nil {
			return fmt.Errorf("put %s in place: %w; and the old one could not be put back: %v", next, err, restoreErr)
		}
		return fmt.Errorf("put %s in place: %w", next, err)
	}
	return nil
}

const lockName = "update.lock"

var ErrBusy = errors.New("another pwikit update is running")

const staleLock = 6 * time.Hour

func Lock(dir string) (func(), error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, lockName)
	for attempt := 0; attempt < 2; attempt++ {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err == nil {
			fmt.Fprintf(f, "%d\n", os.Getpid())
			f.Close()
			return func() { os.Remove(path) }, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, err
		}
		info, statErr := os.Stat(path)
		if statErr != nil || time.Since(info.ModTime()) < staleLock {
			holder, _ := os.ReadFile(path)
			return nil, fmt.Errorf("%w (process %s)", ErrBusy, strings.TrimSpace(string(holder)))
		}
		os.Remove(path)
	}
	return nil, ErrBusy
}
