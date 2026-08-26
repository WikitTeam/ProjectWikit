package static

import (
	"crypto/md5"
	"encoding/hex"
	"io/fs"
	"sync"
)

// Assets builds the versioned URL of a file in the bundle.
type Assets struct {
	fsys  fs.FS
	mu    sync.Mutex
	cache map[string]string
}

func NewAssets(fsys fs.FS) *Assets {
	return &Assets{fsys: fsys, cache: make(map[string]string)}
}

// URL appends a cache buster taken from the content. The digest is md5 to keep
// the shape these URLs already shipped with, not for any strength it has.
func (a *Assets) URL(name string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if url, ok := a.cache[name]; ok {
		return url
	}
	url := Prefix + name
	if a.fsys != nil {
		if data, err := fs.ReadFile(a.fsys, name); err == nil {
			sum := md5.Sum(data)
			url += "?v=" + hex.EncodeToString(sum[:])[:8]
		}
	}
	a.cache[name] = url
	return url
}
