// Package static serves the frontend asset bundle.
package static

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"
)

const (
	Prefix       = "/-/static/"
	cacheControl = "max-age=60, public"
)

type Handler struct {
	fsys fs.FS
	next http.Handler
}

var _ http.Handler = (*Handler)(nil)

// New serves what fsys carries and hands everything else to next. A nil fsys
// hands over every request, which is the state before the bundle is built into
// the binary.
func New(fsys fs.FS, next http.Handler) *Handler {
	return &Handler{fsys: fsys, next: next}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name, ok := h.lookup(r)
	if !ok {
		h.next.ServeHTTP(w, r)
		return
	}
	data, err := fs.ReadFile(h.fsys, name)
	if err != nil {
		h.next.ServeHTTP(w, r)
		return
	}

	w.Header().Set("Content-Type", contentType(name))
	w.Header().Set("Cache-Control", cacheControl)
	// Pages on media_domain load these from the other origin, so the bundle has
	// to be readable cross-origin.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("ETag", etag(data))

	// The zero time keeps Last-Modified out: an embedded file has no mtime,
	// and a made-up one would change on every build.
	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(data))
}

func (h *Handler) lookup(r *http.Request) (string, bool) {
	if h.fsys == nil {
		return "", false
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return "", false
	}
	rest, ok := strings.CutPrefix(r.URL.Path, Prefix)
	if !ok {
		return "", false
	}
	name := path.Clean(rest)
	// fstest.MapFS and os.DirFS both reject these on their own, so this guard
	// is here for whatever fs.FS the bundle ends up being served from.
	if !fs.ValidPath(name) || name == "." {
		return "", false
	}
	info, err := fs.Stat(h.fsys, name)
	if err != nil || info.IsDir() {
		return "", false
	}
	return name, true
}

// contentType leans on the extension alone. Sniffing would let an uploaded
// file decide its own type, and everything under this prefix is a build
// artifact whose extension is trustworthy.
func contentType(name string) string {
	if t := mime.TypeByExtension(path.Ext(name)); t != "" {
		return t
	}
	return "application/octet-stream"
}

// etag hashes the content rather than reading a timestamp: the bundle ships
// inside the binary, where every file has the same zero mtime.
func etag(data []byte) string {
	sum := sha256.Sum256(data)
	return `"` + hex.EncodeToString(sum[:16]) + `"`
}
