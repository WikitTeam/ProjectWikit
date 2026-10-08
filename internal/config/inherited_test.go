package config

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
)

func TestLoadReadsInheritedDescriptor(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe() err = %v, want nil", err)
	}
	if _, err := w.WriteString("[update]\nmirror = \"https://mirror.test\"\n"); err != nil {
		t.Fatalf("WriteString() err = %v, want nil", err)
	}
	w.Close()
	t.Setenv(FDEnv, strconv.Itoa(int(r.Fd())))
	t.Cleanup(func() { piped.data, piped.err = nil, nil; piped.once = sync.Once{} })

	f, err := Load(filepath.Join(t.TempDir(), "unreadable.toml"))
	if err != nil {
		t.Fatalf("Load() err = %v, want nil", err)
	}
	if f.Update.Mirror != "https://mirror.test" {
		t.Errorf("Load().Update.Mirror = %q, want %q", f.Update.Mirror, "https://mirror.test")
	}
	if got := os.Getenv(FDEnv); got != "" {
		t.Errorf("Getenv(%s) after Load = %q, want empty", FDEnv, got)
	}
	again, err := Load(filepath.Join(t.TempDir(), "unreadable.toml"))
	if err != nil {
		t.Fatalf("Load() second call err = %v, want nil", err)
	}
	if again.Update.Mirror != "https://mirror.test" {
		t.Errorf("Load() second call Update.Mirror = %q, want %q", again.Update.Mirror, "https://mirror.test")
	}
}
