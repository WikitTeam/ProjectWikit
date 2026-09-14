package i18n

import (
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode"
)

var scannedDirs = []string{"cmd", "internal"}

func hasHan(line string) bool {
	for _, r := range line {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func TestNoHanTextInGoSource(t *testing.T) {
	var found []string
	for _, dir := range scannedDirs {
		root := filepath.Join("..", "..", dir)
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "testdata" {
					return fs.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for i, line := range strings.Split(string(data), "\n") {
				if hasHan(line) {
					found = append(found, filepath.ToSlash(path)+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(line))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("WalkDir(%s) err = %v, want nil", root, err)
		}
	}

	for _, line := range found {
		t.Errorf("Han text in shipping Go source: %s", line)
	}
}
