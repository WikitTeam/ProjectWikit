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

var skipDirs = map[string]bool{
	".git":         true,
	".claude":      true,
	"node_modules": true,
	"target":       true,
	"postgresql":   true,
	"venv":         true,
	"__pycache__":  true,
}

func hasHan(line string) bool {
	for _, r := range line {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func TestNoHanTextInGoSource(t *testing.T) {
	root := filepath.Join("..", "..")
	var found []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
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

	for _, line := range found {
		t.Errorf("Han text in shipping Go source: %s", line)
	}
}
