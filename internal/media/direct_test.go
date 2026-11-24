package media

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/mediaurl"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/storage"
)

var directSite = &db.Site{ID: 7, Slug: "main"}

func directHandler(t *testing.T, public bool) *Handler {
	t.Helper()
	files := fakeFiles{"start/a.bin": {ArticleName: "start", ArticleMediaName: "article-uuid", MediaName: "file-uuid", Size: 300}}
	h, root := newTestHandler(t, files)
	h.set = storage.Single(&storage.Backend{Store: storage.NewLocal(root), Mode: storage.ModeDirect, PublicURL: "https://cdn.example.org"})
	return h.WithAccess(mediaurl.Signer{Secret: []byte("k")}, func(context.Context, *db.Site, string) bool { return public })
}

func directGet(h *Handler, path string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, path, nil)
	r = r.WithContext(site.WithSite(r.Context(), directSite))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestDirectPublicRedirects(t *testing.T) {
	w := directGet(directHandler(t, true), Prefix+"start/a.bin")
	if w.Code != http.StatusFound {
		t.Fatalf("GET public file status = %d, want %d", w.Code, http.StatusFound)
	}
	if got, want := w.Header().Get("Location"), "https://cdn.example.org/media/article-uuid/file-uuid"; got != want {
		t.Errorf("Location = %q, want %q", got, want)
	}
}

func TestDirectRestrictedNeedsToken(t *testing.T) {
	h := directHandler(t, false)
	if w := directGet(h, Prefix+"start/a.bin"); w.Code != http.StatusNotFound {
		t.Errorf("GET restricted file without token status = %d, want %d", w.Code, http.StatusNotFound)
	}
	token := h.signer.Sign(directSite.ID, "media/article-uuid/file-uuid", time.Now())
	w := directGet(h, Prefix+"start/a.bin?"+mediaurl.TokenParam+"="+token)
	if w.Code != http.StatusOK && w.Code != http.StatusPartialContent {
		t.Errorf("GET restricted file with token status = %d, want 200 or 206", w.Code)
	}
	if got := w.Header().Get("Cache-Control"); got != "private, max-age=600" {
		t.Errorf("Cache-Control = %q, want private", got)
	}
}
