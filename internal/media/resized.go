package media

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/mediaurl"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/storage"
	"github.com/WikitTeam/ProjectWikit/internal/thumb"
)

const (
	ResizedPrefix = "/local--resized-images/"
	resizedDir    = "resized"
	jpegMime      = "image/jpeg"
)

type ResizedHandler struct {
	set    *storage.Set
	files  Attachments
	signer mediaurl.Signer
	public mediaurl.Public
}

var _ http.Handler = (*ResizedHandler)(nil)

func NewResized(set *storage.Set, files Attachments) *ResizedHandler {
	return &ResizedHandler{set: set, files: files}
}

func (h *ResizedHandler) WithAccess(signer mediaurl.Signer, public mediaurl.Public) *ResizedHandler {
	h.signer, h.public = signer, public
	return h
}

// A scaled copy shows what the original shows, so on a direct backend it asks
// the same question the original would.
func (h *ResizedHandler) allowed(r *http.Request, file *db.ArticleFile) bool {
	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil || h.set.For(current.Slug).Mode != storage.ModeDirect {
		return true
	}
	if h.public != nil && h.public(ctx, current, file.ArticleName) {
		return true
	}
	key := mediaurl.MediaKey(file.ArticleMediaName, file.MediaName)
	return h.signer.Valid(current.ID, key, r.URL.Query().Get(mediaurl.TokenParam), time.Now())
}

func (h *ResizedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	rest, ok := strings.CutPrefix(r.URL.Path, ResizedPrefix)
	if !ok {
		notFound(w)
		return
	}
	// The file name keeps its own extension and the size carries another, so
	// the path has one more segment than an attachment's does.
	segments := strings.Split(rest, "/")
	if len(segments) != 3 {
		notFound(w)
		return
	}
	size, ok := thumb.Lookup(strings.TrimSuffix(segments[2], filepath.Ext(segments[2])))
	if !ok {
		notFound(w)
		return
	}

	body, err := h.image(r, segments[0], segments[1], size)
	if err != nil {
		notFound(w)
		return
	}

	w.Header().Set("Content-Type", jpegMime)
	w.Header().Set("Cache-Control", namedCache)
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.Header().Set("Content-Disposition", "inline")
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write(body)
	}
}

func (h *ResizedHandler) image(r *http.Request, articleRef, fileName string, size thumb.Size) ([]byte, error) {
	ctx := r.Context()
	file, err := h.files.ArticleFile(ctx, siteID(ctx), articleRef, fileName)
	if err != nil {
		return nil, err
	}
	if !h.allowed(r, file) {
		return nil, os.ErrNotExist
	}
	if !strings.HasPrefix(file.MimeType, "image/") {
		return nil, os.ErrNotExist
	}

	store := h.set.For(siteSlug(ctx)).Store
	stored := storage.Join(QuoteName(file.ArticleMediaName), QuoteName(file.MediaName))
	cache := storage.Join(resizedDir, stored, size.Name+".jpg")
	if body, err := storage.ReadAll(ctx, store, cache); err == nil {
		return body, nil
	}

	src, err := store.Open(ctx, storage.Join("media", stored))
	if err != nil {
		return nil, err
	}
	defer src.Close()

	body, err := thumb.Generate(src, size)
	if err != nil {
		return nil, err
	}

	// A cache that cannot be written still answers the request, because the
	// scaled copy is derived and losing it costs only the work to redo it.
	_ = store.Put(ctx, cache, bytes.NewReader(body), int64(len(body)))
	return body, nil
}
