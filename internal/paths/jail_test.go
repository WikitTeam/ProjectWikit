package paths

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestJailStaysInside(t *testing.T) {
	base := t.TempDir()
	j := In(base)
	full := filepath.Join(base, "a", "b.txt")
	if err := j.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll() err = %v, want nil", err)
	}
	if err := j.WriteFile(full, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile() err = %v, want nil", err)
	}
	if got, err := j.ReadFile(full); err != nil || string(got) != "x" {
		t.Errorf("ReadFile() = %q, %v, want %q, nil", got, err, "x")
	}
	if _, err := j.ReadFile(filepath.Join(base, "..", "outside")); !errors.Is(err, ErrEscapes) {
		t.Errorf("ReadFile(outside) err = %v, want ErrEscapes", err)
	}
}

func TestJailRefusesALinkOut(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	outside := t.TempDir()
	target := filepath.Join(outside, "secret")
	if err := os.WriteFile(target, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	base := t.TempDir()
	if err := os.Symlink(target, filepath.Join(base, "link")); err != nil {
		t.Fatal(err)
	}
	j := In(base)
	if _, err := j.ReadFile(filepath.Join(base, "link")); err == nil {
		t.Errorf("ReadFile(link out) err = nil, want an error")
	}
	if err := j.WriteFile(filepath.Join(base, "link"), []byte("x"), 0o644); err == nil {
		t.Errorf("WriteFile(link out) err = nil, want an error")
	}
	if got, _ := os.ReadFile(target); string(got) != "secret" {
		t.Errorf("target = %q, want %q", got, "secret")
	}
}

func TestJailFollowsItsOwnLinkedBase(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	real := t.TempDir()
	base := filepath.Join(t.TempDir(), "files")
	if err := os.Symlink(real, base); err != nil {
		t.Fatal(err)
	}
	if err := In(base).WriteFile(filepath.Join(base, "x"), []byte("x"), 0o644); err != nil {
		t.Errorf("WriteFile(own linked base) err = %v, want nil", err)
	}
	if got, _ := os.ReadFile(filepath.Join(real, "x")); string(got) != "x" {
		t.Errorf("ReadFile(real/x) = %q, want %q", got, "x")
	}
}
