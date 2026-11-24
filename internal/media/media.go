// Package media serves the uploaded files under /local--files/.
package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/mediaurl"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/storage"
)

const (
	Prefix          = "/local--files/"
	notFoundBody    = "Not found"
	defaultMime     = "application/octet-stream"
	defaultHTMLMime = "text/html; charset=utf-8"
)

// Attachments resolves an article attachment's on-disk names.
type Attachments interface {
	ArticleFile(ctx context.Context, siteID int64, articleRef, fileName string) (*db.ArticleFile, error)
	ArticleFileByMedia(ctx context.Context, siteID int64, articleMedia, fileMedia string) (*db.ArticleFile, error)
}

type Handler struct {
	set    *storage.Set
	files  Attachments
	signer mediaurl.Signer
	public mediaurl.Public
}

var _ http.Handler = (*Handler)(nil)

// The store holds files/ itself rather than its media/ subdirectory, and the
// request decides which of the two it lands in.
func New(set *storage.Set, files Attachments) *Handler {
	return &Handler{set: set, files: files}
}

// Only a direct backend needs these. Without them every page counts as one
// some readers may not see, which keeps its files behind pwikit.
func (h *Handler) WithAccess(signer mediaurl.Signer, public mediaurl.Public) *Handler {
	h.signer, h.public = signer, public
	return h
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

	key, mimeType, size, named, ok := h.locate(r.Context(), rest)
	if !ok {
		notFound(w)
		return
	}
	// A file reached by its stored name never changes, since a new upload gets
	// a new name. One reached by page and file name can be replaced.
	if named {
		w.Header().Set("Cache-Control", namedCache)
	} else {
		w.Header().Set("Cache-Control", storedCache)
	}
	if !h.direct(w, r, key) {
		return
	}

	f, err := h.backend(r.Context(), key).Store.Open(r.Context(), key)
	if err != nil {
		notFound(w)
		return
	}
	defer f.Close()
	info := f.Info()

	responseMime := mimeType
	if mimeType == "" {
		responseMime = defaultHTMLMime
		mimeType, _ = guessType(path.Base(key))
		if mimeType == "" {
			mimeType = defaultMime
		}
	}

	site.Sandbox(w, site.FromContext(r.Context()), mimeType)

	chunk := chunkSizeFor(mimeType)
	if chunk == 0 {
		w.Header().Set("Content-Type", mimeType)
		w.Header().Set("Content-Disposition", disposition(path.Base(key)))
		w.Header().Set("Content-Length", strconv.FormatInt(info.Size, 10))
		w.WriteHeader(http.StatusOK)
		if r.Method != http.MethodHead {
			copyRange(w, f, 0, info.Size)
		}
		return
	}

	if size == 0 {
		size = info.Size
	}
	if !modifiedSince(r.Header.Get("If-Modified-Since"), info.ModTime) {
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
	w.Header().Set("Last-Modified", info.ModTime.UTC().Format(http.TimeFormat))
	w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range")
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("Accept-Ranges", "bytes")

	if begin >= end || end == 0 {
		// The length has to be zero here, since no body follows and a full one would
		// break the framing.
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", size))
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	// end is exclusive here and inclusive in the header, so this is a byte
	// short of what the client asked for.
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", begin, end-1, size))
	w.Header().Set("Content-Length", strconv.FormatInt(end-begin, 10))
	w.WriteHeader(http.StatusPartialContent)
	if r.Method != http.MethodHead {
		copyRange(w, f, begin, end-begin)
	}
}

// locate turns the URL tail into a path on disk; a type and size come back
// only from the attachment table.
const (
	storedCache = "public, max-age=31536000, immutable"
	namedCache  = "public, max-age=600"
)

func (h *Handler) locate(ctx context.Context, rest string) (key, mimeType string, size int64, named, ok bool) {
	segments := strings.Split(rest, "/")
	prefix := ""

	if !strings.HasPrefix(rest, "-/") {
		prefix = "media"
		// Uploads live under a pair of UUIDs, so a two-segment path that
		// resolves to nothing is read as article and attachment names.
		if len(segments) == 2 && !storage.Exists(ctx, h.set.For(siteSlug(ctx)).Store, storage.Join(prefix, QuoteName(segments[0]), QuoteName(segments[1]))) {
			if f, err := h.files.ArticleFile(ctx, siteID(ctx), segments[0], segments[1]); err == nil {
				segments = []string{f.ArticleMediaName, f.MediaName}
				mimeType, size, named = f.MimeType, f.Size, true
			} else if !errors.Is(err, db.ErrNotFound) {
				return "", "", 0, false, false
			}
		}
	}

	for i, s := range segments {
		segments[i] = QuoteName(s)
	}
	key = strings.Join(segments, "/")
	if prefix != "" {
		key = prefix + "/" + key
	}
	if storage.CheckKey(key) != nil {
		return "", "", 0, false, false
	}
	return key, mimeType, size, named, true
}

// Not urlencoding. Widening this set would put new uploads in directories where
// the files already on disk are not.
func QuoteName(s string) string {
	return strings.NewReplacer(":", "%3A", "/", "%2F", "?", "%3F").Replace(s)
}

// rangeBounds returns a half-open [begin, end), and a false third value is a
// 416 rather than a raised error.
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

func copyRange(w http.ResponseWriter, f io.ReadSeeker, offset, length int64) {
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

// A file on a page some readers may not see is served only to a request carrying
// the token its page was rendered with. False means the response is written.
func (h *Handler) direct(w http.ResponseWriter, r *http.Request, key string) bool {
	ctx := r.Context()
	b := h.backend(ctx, key)
	if b.Mode != storage.ModeDirect {
		return true
	}
	if storage.IsShared(key) {
		return redirect(w, b, key)
	}
	current := site.FromContext(ctx)
	parts := strings.Split(key, "/")
	if current == nil || len(parts) != 3 {
		notFound(w)
		return false
	}
	f, err := h.files.ArticleFileByMedia(ctx, current.ID, unquote(parts[1]), unquote(parts[2]))
	if err != nil {
		notFound(w)
		return false
	}
	if h.public != nil && h.public(ctx, current, f.ArticleName) {
		return redirect(w, b, key)
	}
	if h.signer.Valid(current.ID, key, r.URL.Query().Get(mediaurl.TokenParam), time.Now()) {
		w.Header().Set("Cache-Control", "private, max-age=600")
		return true
	}
	notFound(w)
	return false
}

func redirect(w http.ResponseWriter, b *storage.Backend, key string) bool {
	to, ok := b.URL(key)
	if !ok {
		return true
	}
	w.Header().Set("Cache-Control", namedCache)
	w.Header().Set("Location", to)
	w.WriteHeader(http.StatusFound)
	return false
}

func unquote(s string) string {
	return strings.NewReplacer("%3A", ":", "%2F", "/", "%3F", "?").Replace(s)
}

func (h *Handler) backend(ctx context.Context, key string) *storage.Backend {
	return h.set.ForKey(siteSlug(ctx), key)
}

func siteSlug(ctx context.Context) string {
	if current := site.FromContext(ctx); current != nil {
		return current.Slug
	}
	return ""
}

func siteID(ctx context.Context) int64 {
	if current := site.FromContext(ctx); current != nil {
		return current.ID
	}
	return 0
}
