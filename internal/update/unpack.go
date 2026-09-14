package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

func ExtractExecutable(archive, dest, goos string) error {
	want := ExecutableName(goos)
	if strings.HasSuffix(archive, ".zip") {
		return extractZip(archive, dest, want)
	}
	return extractTarGz(archive, dest, want)
}

func extractTarGz(archive, dest, want string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("read %s: %w", archive, err)
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("%s holds no %s", archive, want)
		}
		if err != nil {
			return fmt.Errorf("read %s: %w", archive, err)
		}
		if h.Typeflag == tar.TypeReg && path.Base(h.Name) == want && strings.Count(strings.Trim(h.Name, "/"), "/") == 1 {
			return writeExecutable(dest, tr)
		}
	}
}

func extractZip(archive, dest, want string) error {
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return fmt.Errorf("read %s: %w", archive, err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || path.Base(f.Name) != want || strings.Count(strings.Trim(f.Name, "/"), "/") != 1 {
			continue
		}
		in, err := f.Open()
		if err != nil {
			return err
		}
		defer in.Close()
		return writeExecutable(dest, in)
	}
	return fmt.Errorf("%s holds no %s", archive, want)
}

func writeExecutable(dest string, from io.Reader) error {
	tmp := dest + ".part"
	out, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, from); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dest)
}
