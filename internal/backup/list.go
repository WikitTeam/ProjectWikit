package backup

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Entry struct {
	Path     string
	Bytes    int64
	Manifest Manifest
	Problem  string
}

// A file this cannot read is reported rather than skipped, since a backup
// nobody can open is the one worth hearing about.
func List(dir string) ([]Entry, error) {
	found, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []Entry
	for _, one := range found {
		if one.IsDir() || !strings.HasSuffix(one.Name(), Extension) {
			continue
		}
		full := filepath.Join(dir, one.Name())
		entry := Entry{Path: full}
		if info, err := one.Info(); err == nil {
			entry.Bytes = info.Size()
		}
		m, err := ReadManifestOf(full)
		if err != nil {
			entry.Problem = err.Error()
		} else {
			entry.Manifest = m
		}
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Manifest.CreatedAt.After(out[j].Manifest.CreatedAt)
	})
	return out, nil
}
