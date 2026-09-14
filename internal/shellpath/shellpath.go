// Package shellpath puts the pwikit command where a shell looks for it.
package shellpath

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const command = "pwikit"

type Place struct {
	// Path is the link on Linux and macOS, and the directory added to Path on
	// Windows.
	Path    string
	Target  string
	Working bool
	Ours    bool
	OnPath  bool
}

type Outcome string

const (
	Created   Outcome = "created"
	Replaced  Outcome = "replaced"
	Unchanged Outcome = "unchanged"
	Removed   Outcome = "removed"
	Absent    Outcome = "absent"
)

func onPath(dir string) bool {
	want := normalise(dir)
	for _, entry := range filepath.SplitList(os.Getenv("PATH")) {
		if entry != "" && normalise(entry) == want {
			return true
		}
	}
	return false
}

func normalise(dir string) string {
	clean := filepath.Clean(dir)
	if runtime.GOOS == "windows" {
		return strings.ToLower(clean)
	}
	return clean
}

func sameFile(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if ra, err := filepath.EvalSymlinks(a); err == nil {
		a = ra
	}
	if rb, err := filepath.EvalSymlinks(b); err == nil {
		b = rb
	}
	return normalise(a) == normalise(b)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
