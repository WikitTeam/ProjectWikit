package backup

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func filesArchive(t *testing.T, files map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "b.pwbak")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := gzip.NewWriter(f)
	tw := tar.NewWriter(zw)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: filesDir + "/" + name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		tw.Write([]byte(body))
	}
	tw.Close()
	zw.Close()
	f.Close()
	return path
}

func TestRestoreFilesWhenTheDataDirectoryIsClosed(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs a directory this account cannot write")
	}
	top := t.TempDir()
	root := filepath.Join(top, "files")
	if err := os.MkdirAll(filepath.Join(root, "main", "old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main", "old", "a.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(top, "backups"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(top, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(top, 0o755) })

	var out RestoreResult
	if err := restoreFiles(filesArchive(t, map[string]string{"main/new/b.txt": "new"}), root, Manifest{}, &out); err != nil {
		t.Fatalf("restoreFiles() err = %v, want nil", err)
	}
	if got, err := os.ReadFile(filepath.Join(root, "main", "new", "b.txt")); err != nil || string(got) != "new" {
		t.Errorf("restored b.txt = %q, %v, want %q", got, err, "new")
	}
	if _, err := os.Stat(filepath.Join(root, "main", "old")); !os.IsNotExist(err) {
		t.Errorf("Stat(old page dir) err = %v, want not exist", err)
	}
	if want := filepath.Join(top, "backups", "files.replaced"); out.ReplacedDir != want {
		t.Errorf("ReplacedDir = %q, want %q", out.ReplacedDir, want)
	}
	if got, err := os.ReadFile(filepath.Join(out.ReplacedDir, "main", "old", "a.txt")); err != nil || string(got) != "old" {
		t.Errorf("kept a.txt = %q, %v, want %q", got, err, "old")
	}
}
