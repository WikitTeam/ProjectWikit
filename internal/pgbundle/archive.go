package pgbundle

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"

	"github.com/WikitTeam/ProjectWikit/internal/paths"
)

const Version = "18.6.0"

const (
	markerFile = ".pwikit-archive"

	// Bump when Unpack lays files out differently, or old unpacked copies are kept.
	unpackRevision = "1"
)

func Embedded() bool {
	return len(archive) > 0
}

func Unpack(dir string) (bool, error) {
	if !Embedded() {
		return false, nil
	}
	return unpack(archive, strings.TrimSpace(archiveSum), dir)
}

func unpack(data []byte, sum, dir string) (bool, error) {
	marker := sum + " " + unpackRevision + "\n"
	if have, err := os.ReadFile(filepath.Join(dir, markerFile)); err == nil && string(have) == marker {
		return false, nil
	}

	staging := dir + ".unpacking"
	if err := os.RemoveAll(staging); err != nil {
		return false, fmt.Errorf("clear %s: %w", staging, err)
	}
	if err := extract(data, staging); err != nil {
		os.RemoveAll(staging)
		return false, err
	}
	if err := os.WriteFile(filepath.Join(staging, markerFile), []byte(marker), 0o644); err != nil {
		os.RemoveAll(staging)
		return false, err
	}

	old := dir + ".old"
	if err := os.RemoveAll(old); err != nil {
		return false, fmt.Errorf("clear %s: %w", old, err)
	}
	if err := os.Rename(dir, old); err != nil && !errors.Is(err, os.ErrNotExist) {
		os.RemoveAll(staging)
		return false, fmt.Errorf("set aside the previous PostgreSQL in %s: %w", dir, err)
	}
	if err := os.Rename(staging, dir); err != nil {
		return false, fmt.Errorf("move PostgreSQL into %s: %w", dir, err)
	}
	os.RemoveAll(old)
	return true, nil
}

func extract(data []byte, dest string) error {
	zr, err := zstd.NewReader(bytes.NewReader(data), zstd.WithDecoderConcurrency(1), zstd.WithDecoderLowmem(true))
	if err != nil {
		return err
	}
	defer zr.Close()

	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	tr := tar.NewReader(zr)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read the bundled PostgreSQL: %w", err)
		}
		name, err := paths.Resolve(dest, filepath.FromSlash(strings.TrimSuffix(h.Name, "/")))
		if err != nil {
			return err
		}
		switch h.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(name, 0o755)
		case tar.TypeReg:
			err = writeFile(name, tr, os.FileMode(h.Mode)&0o777)
		case tar.TypeLink:
			err = link(dest, name, h.Linkname)
		case tar.TypeSymlink:
			err = symlink(dest, name, h.Linkname)
		}
		if err != nil {
			return fmt.Errorf("unpack %s: %w", h.Name, err)
		}
	}
}

func writeFile(name string, r io.Reader, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func link(dest, name, target string) error {
	from, err := paths.Resolve(dest, filepath.FromSlash(target))
	if err != nil {
		return err
	}
	return os.Link(from, name)
}

func symlink(dest, name, target string) error {
	if filepath.IsAbs(target) {
		return fmt.Errorf("%w: link to %q", paths.ErrEscapes, target)
	}
	rel, err := filepath.Rel(dest, filepath.Join(filepath.Dir(name), target))
	if err != nil {
		return err
	}
	if _, err := paths.Resolve(dest, rel); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return err
	}
	return os.Symlink(target, name)
}
