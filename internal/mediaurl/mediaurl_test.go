package mediaurl

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/storage"
)

func TestSignerRoundTrip(t *testing.T) {
	s := Signer{Secret: []byte("k")}
	now := time.Unix(1_700_000_000, 0)
	token := s.Sign(1, "media/a/b", now)
	if !s.Valid(1, "media/a/b", token, now.Add(time.Hour)) {
		t.Errorf("Valid(same key, an hour later) = false, want true")
	}
	if s.Valid(1, "media/a/c", token, now) {
		t.Errorf("Valid(other key) = true, want false")
	}
	if s.Valid(2, "media/a/b", token, now) {
		t.Errorf("Valid(other site) = true, want false")
	}
	if s.Valid(1, "media/a/b", token, now.Add(7*time.Hour)) {
		t.Errorf("Valid(after expiry) = true, want false")
	}
}

type files struct{}

func (files) ArticleFile(_ context.Context, _ int64, article, name string) (*db.ArticleFile, error) {
	return &db.ArticleFile{ArticleName: article, ArticleMediaName: "A-" + article, MediaName: "F-" + name}, nil
}

func (files) ArticleFileByMedia(context.Context, int64, string, string) (*db.ArticleFile, error) {
	return nil, db.ErrNotFound
}

func linker(mode storage.Mode) *Linker {
	direct := &storage.Backend{Mode: mode, PublicURL: "https://cdn.example.org"}
	return &Linker{
		Set:    storage.Single(direct),
		Signer: Signer{Secret: []byte("k")},
		Files:  files{},
		Public: func(_ context.Context, _ *db.Site, article string) bool { return article != "secret" },
		Now:    func() time.Time { return time.Unix(1_700_000_000, 0) },
	}
}

var site = &db.Site{ID: 1, Slug: "main", Domain: "wiki.example.org", MediaDomain: "files.example.org"}

func TestRewritePublicPage(t *testing.T) {
	got := linker(storage.ModeDirect).Rewrite(context.Background(), site, `<img src="//files.example.org/local--files/start/cat.png">`)
	want := `<img src="https://cdn.example.org/media/A-start/F-cat.png">`
	if got != want {
		t.Errorf("Rewrite(public) = %q, want %q", got, want)
	}
}

func TestRewriteRestrictedPageSigns(t *testing.T) {
	got := linker(storage.ModeDirect).Rewrite(context.Background(), site, `<a href="/local--files/secret/plan.pdf">`)
	if !strings.HasPrefix(got, `<a href="/local--files/secret/plan.pdf?t=`) {
		t.Errorf("Rewrite(restricted) = %q, want the pwikit address with a token", got)
	}
}

func TestRewriteLeavesOtherHosts(t *testing.T) {
	in := `<img src="https://elsewhere.org/local--files/start/cat.png">`
	if got := linker(storage.ModeDirect).Rewrite(context.Background(), site, in); got != in {
		t.Errorf("Rewrite(other host) = %q, want it unchanged", got)
	}
}

func TestRewriteOnlyInDirectMode(t *testing.T) {
	in := `<img src="/local--files/start/cat.png">`
	if got := linker(storage.ModeProxy).Rewrite(context.Background(), site, in); got != in {
		t.Errorf("Rewrite(proxy) = %q, want it unchanged", got)
	}
}

func TestRewriteSharedFiles(t *testing.T) {
	got := linker(storage.ModeDirect).Rewrite(context.Background(), site, `<img src="/local--files/-/users/1-ab.png">`)
	want := `<img src="https://cdn.example.org/-/users/1-ab.png">`
	if got != want {
		t.Errorf("Rewrite(avatar) = %q, want %q", got, want)
	}
}

func TestRewriteResizedOnRestrictedPageSigns(t *testing.T) {
	got := linker(storage.ModeDirect).Rewrite(context.Background(), site, `<img src="/local--resized-images/secret/cat.png/small.jpg">`)
	if !strings.HasPrefix(got, `<img src="/local--resized-images/secret/cat.png/small.jpg?t=`) {
		t.Errorf("Rewrite(restricted resized) = %q, want a token", got)
	}
}

func TestRewriteResizedOnPublicPageStays(t *testing.T) {
	in := `<img src="/local--resized-images/start/cat.png/small.jpg">`
	if got := linker(storage.ModeDirect).Rewrite(context.Background(), site, in); got != in {
		t.Errorf("Rewrite(public resized) = %q, want it unchanged", got)
	}
}
