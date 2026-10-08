package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRecordedRoot(t *testing.T) {
	dir := t.TempDir()
	if _, ok := recordedRoot(dir); ok {
		t.Errorf("recordedRoot(no file) ok = true, want false")
	}
	want := filepath.Join(dir, "instance")
	if err := os.WriteFile(filepath.Join(dir, DataDirFile), []byte(want+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, ok := recordedRoot(dir); !ok || got != want {
		t.Errorf("recordedRoot() = %q, %v, want %q, true", got, ok, want)
	}
	if err := os.WriteFile(filepath.Join(dir, DataDirFile), []byte("relative\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := recordedRoot(dir); ok {
		t.Errorf("recordedRoot(relative) ok = true, want false")
	}
}
