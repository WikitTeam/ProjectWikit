package media

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/paths"
	"github.com/WikitTeam/ProjectWikit/internal/thumb"
)

const (
	ResizedPrefix = "/local--resized-images/"
	resizedDir    = "resized"
	jpegMime      = "image/jpeg"
)

type ResizedHandler struct {
	root  string
	files Attachments
}

var _ http.Handler = (*ResizedHandler)(nil)

func NewResized(root string, files Attachments) *ResizedHandler {
	return &ResizedHandler{root: root, files: files}
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

	body, err := h.image(r.Context(), segments[0], segments[1], size)
	if err != nil {
		notFound(w)
		return
	}

	w.Header().Set("Content-Type", jpegMime)
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.Header().Set("Content-Disposition", "inline")
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write(body)
	}
}

func (h *ResizedHandler) image(ctx context.Context, articleRef, fileName string, size thumb.Size) ([]byte, error) {
	file, err := h.files.ArticleFile(ctx, articleRef, fileName)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(file.MimeType, "image/") {
		return nil, os.ErrNotExist
	}

	stored := filepath.Join(QuoteName(file.ArticleMediaName), QuoteName(file.MediaName))
	cache, err := paths.Resolve(filepath.Join(h.root, resizedDir), filepath.Join(stored, size.Name+".jpg"))
	if err != nil {
		return nil, err
	}
	if body, err := os.ReadFile(cache); err == nil {
		return body, nil
	}

	original, err := paths.Resolve(filepath.Join(h.root, "media"), stored)
	if err != nil {
		return nil, err
	}
	src, err := os.Open(original)
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
	if err := os.MkdirAll(filepath.Dir(cache), 0o755); err == nil {
		_ = os.WriteFile(cache, body, 0o644)
	}
	return body, nil
}
