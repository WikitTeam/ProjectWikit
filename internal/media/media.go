// Package media serves the uploaded files under /local--files/.
package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/paths"
)

const (
	Prefix       = "/local--files/"
	notFoundBody = "Not found"
	defaultMime  = "application/octet-stream"
	// What Django gives a response built without an explicit type.
	defaultHTMLMime = "text/html; charset=utf-8"
)

// Attachments resolves an article attachment's on-disk names.
type Attachments interface {
	ArticleFile(ctx context.Context, articleRef, fileName string) (*db.ArticleFile, error)
}

type Handler struct {
	root  string
	files Attachments
}

var _ http.Handler = (*Handler)(nil)

// New serves out of root, which is files/ itself rather than its media/
// subdirectory; the request decides which of the two it lands in.
func New(root string, files Attachments) *Handler {
	return &Handler{root: root, files: files}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	rest, ok := strings.CutPrefix(r.URL.Path, Prefix)
	if !ok || rest == "" {
		notFound(w)
		return
	}

	full, mimeType, size, ok := h.locate(r.Context(), rest)
	if !ok {
		notFound(w)
		return
	}

	f, err := os.Open(full)
	if err != nil {
		notFound(w)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		notFound(w)
		return
	}

	// Django never writes the guessed type back onto the response, so anything
	// the database did not name goes out labelled text/html.
	responseMime := mimeType
	if mimeType == "" {
		responseMime = defaultHTMLMime
		mimeType, _ = guessType(filepath.Base(full))
		if mimeType == "" {
			mimeType = defaultMime
		}
	}

	chunk := chunkSizeFor(mimeType)
	if chunk == 0 {
		// Django's FileResponse sets its own headers and ignores
		// If-Modified-Since.
		w.Header().Set("Content-Type", mimeType)
		w.Header().Set("Content-Disposition", disposition(filepath.Base(full)))
		w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
		w.WriteHeader(http.StatusOK)
		if r.Method != http.MethodHead {
			copyRange(w, f, 0, info.Size())
		}
		return
	}

	if size == 0 {
		size = info.Size()
	}
	if !modifiedSince(r.Header.Get("If-Modified-Since"), info.ModTime()) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	begin, end, ok := rangeBounds(r.Header.Get("Range"), chunk, size)
	if !ok {
		w.Header().Set("Content-Type", defaultHTMLMime)
		w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	w.Header().Set("Content-Type", responseMime)
	w.Header().Set("Last-Modified", info.ModTime().UTC().Format(http.TimeFormat))
	w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range")
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("Accept-Ranges", "bytes")

	if begin >= end || end == 0 {
		// Django keeps the full Content-Length here and sends no body, which
		// breaks the framing.
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", size))
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	// end is exclusive here and inclusive in the header, so this is a byte
	// short of what the client asked for. Django does the same.
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", begin, end-1, size))
	w.Header().Set("Content-Length", strconv.FormatInt(end-begin, 10))
	w.WriteHeader(http.StatusPartialContent)
	if r.Method != http.MethodHead {
		copyRange(w, f, begin, end-begin)
	}
}

// locate turns the URL tail into a path on disk; a type and size come back
// only from the attachment table.
func (h *Handler) locate(ctx context.Context, rest string) (full, mimeType string, size int64, ok bool) {
	segments := strings.Split(rest, "/")
	root := h.root

	if !strings.HasPrefix(rest, "-/") {
		root = filepath.Join(root, "media")
		// Uploads live under a pair of UUIDs, so a two-segment path that
		// resolves to nothing is read as article and attachment names.
		if len(segments) == 2 && !exists(filepath.Join(root, rest)) {
			if f, err := h.files.ArticleFile(ctx, segments[0], segments[1]); err == nil {
				segments = []string{f.ArticleMediaName, f.MediaName}
				mimeType, size = f.MimeType, f.Size
			} else if !errors.Is(err, db.ErrNotFound) {
				return "", "", 0, false
			}
		}
	}

	for i, s := range segments {
		segments[i] = partialQuote(s)
	}
	full, err := paths.Resolve(root, filepath.Join(segments...))
	if err != nil {
		return "", "", 0, false
	}
	return full, mimeType, size, true
}

// partialQuote is not urlencoding: the write side escapes only ':' and '/',
// so widening this set would stop finding files already on disk.
func partialQuote(s string) string {
	return strings.NewReplacer(":", "%3A", "/", "%2F", "?", "%3F").Replace(s)
}

func exists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

// rangeBounds returns a half-open [begin, end). A false third value is a 416;
// Django raises on the malformed cases instead.
func rangeBounds(header string, chunk, size int64) (begin, end int64, ok bool) {
	if header == "" {
		return 0, min(chunk, size), true
	}
	unit, spec, found := strings.Cut(header, "=")
	if !found || unit != "bytes" {
		return 0, 0, false
	}
	first, last, found := strings.Cut(spec, "-")
	if !found {
		return 0, 0, false
	}
	// An empty bound reads as 0, and 0 then reads as "not given" a line later,
	// which is why bytes=-50 returns the first 50 bytes rather than the last.
	if begin, ok = parseBound(first); !ok {
		return 0, 0, false
	}
	if end, ok = parseBound(last); !ok {
		return 0, 0, false
	}

	if begin != 0 {
		begin = min(begin, size)
	}
	maxEnd := min(begin+chunk, size)
	if end == 0 {
		end = maxEnd
	} else {
		end = min(end, maxEnd)
	}
	return begin, min(end, size), true
}

func parseBound(s string) (int64, bool) {
	if s == "" {
		return 0, true
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

// modifiedSince counts anything it cannot parse as modified.
func modifiedSince(header string, mtime time.Time) bool {
	if header == "" {
		return true
	}
	since, err := http.ParseTime(strings.TrimSpace(strings.Split(header, ";")[0]))
	if err != nil {
		return true
	}
	return mtime.Unix() > since.Unix()
}

func copyRange(w http.ResponseWriter, f *os.File, offset, length int64) {
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return
	}
	_, _ = io.Copy(w, io.LimitReader(f, length))
}

// disposition moves a name that is not ASCII to the RFC 5987 form instead of
// quoting it.
func disposition(name string) string {
	for i := 0; i < len(name); i++ {
		if name[i] > 127 {
			return "inline; filename*=utf-8''" + escape.URLQuote(name)
		}
	}
	escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(name)
	return `inline; filename="` + escaped + `"`
}

func notFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", defaultHTMLMime)
	w.Header().Set("Content-Length", strconv.Itoa(len(notFoundBody)))
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(notFoundBody))
}
