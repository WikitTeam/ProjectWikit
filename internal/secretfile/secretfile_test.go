package secretfile

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestEnsureMakesOneAndKeepsIt(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "secrets")
	first, err := Ensure(dir, "key")
	if err != nil {
		t.Fatalf("Ensure() err = %v, want nil", err)
	}
	if len(first) < 32 {
		t.Errorf("len(Ensure()) = %d, want at least 32", len(first))
	}
	second, err := Ensure(dir, "key")
	if err != nil {
		t.Fatalf("Ensure() second call err = %v, want nil", err)
	}
	if second != first {
		t.Errorf("Ensure() second call = %q, want the first %q", second, first)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(dir, "key"))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Errorf("secret file mode = %o, want 600", info.Mode().Perm())
		}
	}
}

func TestEnsureKeepsAValueSomebodyWrote(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "key"), []byte("carried-over\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Ensure(dir, "key")
	if err != nil {
		t.Fatalf("Ensure() err = %v, want nil", err)
	}
	if got != "carried-over" {
		t.Errorf("Ensure() = %q, want %q", got, "carried-over")
	}
}

func TestEnsureReplacesAnEmptyFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "key"), []byte("\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Ensure(dir, "key")
	if err != nil {
		t.Fatalf("Ensure() err = %v, want nil", err)
	}
	if got == "" {
		t.Errorf("Ensure() over an empty file = %q, want a new value", got)
	}
}

func TestEnsureFailsWhereTheNameIsADirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "key"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := Ensure(dir, "key"); err == nil {
		t.Errorf("Ensure() where the name is a directory err = nil, want an error")
	}
}
