//go:build unix

package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstanceCheck(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "pwikit")
	private := filepath.Join(base, "private")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(private, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := RecordInstance(private, root, filepath.Join(root, "pwikit")); err != nil {
		t.Fatalf("RecordInstance() err = %v, want nil", err)
	}
	in, ok := ReadInstance(private)
	if !ok {
		t.Fatalf("ReadInstance() ok = false, want true")
	}
	if err := in.Check(); err != nil {
		t.Errorf("Check(untouched) err = %v, want nil", err)
	}
	if got, _ := os.ReadFile(filepath.Join(private, PrivateBin, "pwikit.data-dir")); strings.TrimSpace(string(got)) != root {
		t.Errorf("data-dir record = %q, want %q", got, root)
	}

	moved := filepath.Join(base, "elsewhere")
	if err := os.Rename(root, moved); err != nil {
		t.Fatal(err)
	}
	if err := in.Check(); err == nil || !strings.Contains(err.Error(), "no longer exists") {
		t.Errorf("Check(moved) err = %v, want one saying the directory no longer exists", err)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := in.Check(); err == nil || !strings.Contains(err.Error(), "no longer the directory") {
		t.Errorf("Check(replaced) err = %v, want one saying it is no longer the directory", err)
	}
	if _, err := os.Stat(filepath.Join(root, "pwikit.toml")); !os.IsNotExist(err) {
		t.Errorf("Stat(pwikit.toml) err = %v, want not exist", err)
	}
}
