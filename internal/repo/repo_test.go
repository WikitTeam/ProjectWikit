package repo

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/callbacks"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/printuser"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
	"github.com/WikitTeam/ProjectWikit/internal/renderer/sidecar"
)

func newTestRepo(t *testing.T) *Repository {
	t.Helper()
	dsn := os.Getenv(db.EnvDSN)
	if dsn == "" {
		t.Skipf("%s not set, skipping the database test", db.EnvDSN)
	}
	d, err := db.Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("db.Open() err = %v, want nil", err)
	}
	t.Cleanup(d.Close)
	bundle, err := i18n.Load("")
	if err != nil {
		t.Fatalf("i18n.Load() err = %v, want nil", err)
	}
	users := printuser.New(bundle.Localizer(i18n.DefaultLanguage), nil)
	return New(context.Background(), d, users)
}

func TestPageInfoDropsMissingPages(t *testing.T) {
	r := newTestRepo(t)

	got, err := r.PageInfo([]string{"main", "no-such-page", "nav:top"})
	if err != nil {
		t.Fatalf("PageInfo() err = %v, want nil", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(PageInfo()) = %d, want 2", len(got))
	}
	if got[0].FullName != "main" {
		t.Errorf("PageInfo()[0].FullName = %q, want %q", got[0].FullName, "main")
	}
	if !got[0].Exists {
		t.Errorf("PageInfo()[0].Exists = false, want true")
	}
	if got[0].Title == nil || *got[0].Title != "main" {
		t.Errorf("PageInfo()[0].Title = %v, want %q", got[0].Title, "main")
	}
}

func TestIncludeSourcesKeepsRefOrderAndMarksMisses(t *testing.T) {
	r := newTestRepo(t)

	got, err := r.IncludeSources([]renderer.IncludeRef{
		{FullName: "component:box"},
		{FullName: "no-such-page"},
	})
	if err != nil {
		t.Fatalf("IncludeSources() err = %v, want nil", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(IncludeSources()) = %d, want 2", len(got))
	}
	if got[0].Content == nil {
		t.Fatalf("IncludeSources()[0].Content = nil, want the page source")
	}
	if !strings.Contains(*got[0].Content, "box") {
		t.Errorf("IncludeSources()[0].Content = %q, want substring %q", *got[0].Content, "box")
	}
	if got[1].Content != nil {
		t.Errorf("IncludeSources()[1].Content = %q, want nil", *got[1].Content)
	}
}

// TestRenderAgainstDatabase drives real ftml with callbacks backed by a real
// Postgres: this is the only test where the data layer, the callback layer and
// the renderer all run together.
func TestRenderAgainstDatabase(t *testing.T) {
	bin := os.Getenv(sidecar.EnvBinary)
	if bin == "" {
		t.Skipf("%s not set, skipping the renderer integration test", sidecar.EnvBinary)
	}
	r := newTestRepo(t)

	rend, err := sidecar.New(bin)
	if err != nil {
		t.Fatalf("sidecar.New(%q) err = %v, want nil", bin, err)
	}
	t.Cleanup(func() { rend.Close() })

	bundle, err := i18n.Load("")
	if err != nil {
		t.Fatalf("i18n.Load() err = %v, want nil", err)
	}
	cb := callbacks.New(bundle.Localizer("zh-hans"), r)
	info := renderer.PageInfo{Page: "main", Category: "_default", Domain: "localhost"}

	source := "[[[main | 首页]]] [[[no-such-page | 红链]]]\n\n[[include component:box a=1]]"
	got, err := rend.RenderHTML(context.Background(), source, info, cb, renderer.ModeArticle)
	if err != nil {
		t.Fatalf("RenderHTML() err = %v, want nil", err)
	}

	if strings.Contains(got.Body, `href="/main" class="newpage"`) {
		t.Errorf("RenderHTML() marks /main as a red link, want an existing page")
	}
	if !strings.Contains(got.Body, "newpage") {
		t.Errorf("RenderHTML() = %q, want a red link for no-such-page", got.Body)
	}
	if !strings.Contains(got.Body, "这是一个被 include 的组件") {
		t.Errorf("RenderHTML() = %q, want the included page body", got.Body)
	}
}

func TestRenderUserExternalSkipsTheDatabase(t *testing.T) {
	r := newTestRepo(t)

	got, err := r.RenderUser("EXTERNAL:Some User", false)
	if err != nil {
		t.Fatalf("RenderUser() err = %v, want nil", err)
	}
	if !strings.Contains(got, `data-user-id="-1"`) {
		t.Errorf("RenderUser() = %q, want it to contain %q", got, `data-user-id="-1"`)
	}
	if !strings.Contains(got, "https://www.wikidot.com/user:info/some-user") {
		t.Errorf("RenderUser() = %q, want it to contain %q", got, "https://www.wikidot.com/user:info/some-user")
	}
}

func TestRenderUserUnknown(t *testing.T) {
	r := newTestRepo(t)

	_, err := r.RenderUser("no-such-user", false)
	if !errors.Is(err, callbacks.ErrUserNotFound) {
		t.Errorf("RenderUser(\"no-such-user\") err = %v, want ErrUserNotFound", err)
	}
}

func TestRenderUserFromTheDatabase(t *testing.T) {
	r := newTestRepo(t)

	got, err := r.RenderUser("SeedUser", false)
	if err != nil {
		t.Fatalf("RenderUser() err = %v, want nil", err)
	}
	if !strings.Contains(got, `<a href="/-/users/seeduser">seeduser</a>`) {
		t.Errorf("RenderUser() = %q, want it to contain %q", got, `<a href="/-/users/seeduser">seeduser</a>`)
	}
}
