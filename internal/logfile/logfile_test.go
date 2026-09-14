package logfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) err = %v, want nil", filepath.Base(path), err)
	}
	return string(raw)
}

func mustWrite(t *testing.T, w *Writer, line string) {
	t.Helper()
	if _, err := w.Write([]byte(line)); err != nil {
		t.Fatalf("Write(%q) err = %v, want nil", line, err)
	}
}

func TestWriteAppendsToWhatIsThere(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pwikit.log")
	if err := os.WriteFile(path, []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	w, err := Open(path, 1<<20, 3)
	if err != nil {
		t.Fatalf("Open() err = %v, want nil", err)
	}
	mustWrite(t, w, "new\n")
	w.Close()
	if got := readFile(t, path); got != "old\nnew\n" {
		t.Errorf("log = %q, want %q", got, "old\nnew\n")
	}
}

func TestWriteRotatesPastTheLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pwikit.log")
	w, err := Open(path, 10, 3)
	if err != nil {
		t.Fatalf("Open() err = %v, want nil", err)
	}
	mustWrite(t, w, "aaaaaaaa\n")
	mustWrite(t, w, "bbbbbbbb\n")
	w.Close()
	if got := readFile(t, path); got != "bbbbbbbb\n" {
		t.Errorf("pwikit.log = %q, want %q", got, "bbbbbbbb\n")
	}
	if got := readFile(t, path+".1"); got != "aaaaaaaa\n" {
		t.Errorf("pwikit.log.1 = %q, want %q", got, "aaaaaaaa\n")
	}
}

func TestWriteKeepsOnlySoManyOldFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pwikit.log")
	w, err := Open(path, 4, 2)
	if err != nil {
		t.Fatalf("Open() err = %v, want nil", err)
	}
	for _, line := range []string{"1111", "2222", "3333", "4444"} {
		mustWrite(t, w, line)
	}
	w.Close()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if got, want := strings.Join(names, ","), "pwikit.log,pwikit.log.1,pwikit.log.2"; got != want {
		t.Errorf("files = %s, want %s", got, want)
	}
	if got := readFile(t, path+".2"); got != "2222" {
		t.Errorf("pwikit.log.2 = %q, want %q", got, "2222")
	}
}

func TestWriteLetsOneOversizedLineThrough(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pwikit.log")
	w, err := Open(path, 4, 2)
	if err != nil {
		t.Fatalf("Open() err = %v, want nil", err)
	}
	mustWrite(t, w, "a line longer than the limit\n")
	w.Close()
	if _, err := os.Stat(path + ".1"); !os.IsNotExist(err) {
		t.Errorf("Stat(pwikit.log.1) err = %v, want not exist", err)
	}
}

func TestWriteAfterCloseFails(t *testing.T) {
	w, err := Open(filepath.Join(t.TempDir(), "pwikit.log"), 1<<20, 2)
	if err != nil {
		t.Fatalf("Open() err = %v, want nil", err)
	}
	w.Close()
	if _, err := w.Write([]byte("late")); err == nil {
		t.Errorf("Write() after Close err = nil, want an error")
	}
}
