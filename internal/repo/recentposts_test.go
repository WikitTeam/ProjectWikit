package repo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/callbacks"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/renderer/sidecar"
)

const (
	recentPostsGolden = "testdata/recentposts.golden"
	recentPostsCorpus = "testdata/recentposts_corpus.json"
)

type recentPostsCase struct {
	Name     string            `json:"name"`
	Viewer   string            `json:"viewer"`
	Category string            `json:"category,omitempty"`
	Params   map[string]string `json:"params,omitempty"`
	Path     map[string]string `json:"path,omitempty"`
}

func recentPostsCases() []recentPostsCase {
	return []recentPostsCase{
		{Name: "all", Viewer: ""},
		{Name: "all-signed-in", Viewer: "probe-author"},
		{Name: "second-page", Viewer: "", Path: map[string]string{"p": "2"}},
		{Name: "past-the-end", Viewer: "", Path: map[string]string{"p": "99"}},
		{Name: "below-one", Viewer: "", Path: map[string]string{"p": "-3"}},
		{Name: "junk-page", Viewer: "", Path: map[string]string{"p": "later"}},
		{Name: "one-category", Viewer: "", Category: "Probe Talk"},
		{Name: "empty-category", Viewer: "", Category: "Probe Quiet"},
		{Name: "comments-category", Viewer: "", Category: "Probe Comments"},
		{Name: "hidden-section-category", Viewer: "", Category: "Probe Hidden Chat"},
		{Name: "star", Viewer: "", Path: map[string]string{"c": "*"}},
		{Name: "missing-category", Viewer: "", Path: map[string]string{"c": "999999"}},
		{Name: "junk-category", Viewer: "", Path: map[string]string{"c": "not-a-number"}},
		{Name: "with-params", Viewer: "", Params: map[string]string{"limit": "5"}},
	}
}

func TestRecentPostsMatchesGolden(t *testing.T) {
	bin := os.Getenv(sidecar.EnvBinary)
	if bin == "" {
		t.Skipf("%s not set, skipping the recent posts test", sidecar.EnvBinary)
	}
	ctx := context.Background()
	d := testDB(t)
	users, loc := testUsers(t)
	cases := recentPostsCases()

	rend, err := sidecar.New(bin)
	if err != nil {
		t.Fatalf("sidecar.New(%q) err = %v, want nil", bin, err)
	}
	t.Cleanup(func() { rend.Close() })

	site, err := d.SiteByHosts(ctx, []string{"localhost"})
	if err != nil {
		t.Fatalf("SiteByHosts(localhost) err = %v, want nil", err)
	}
	article, err := d.ArticleByName(ctx, "forum:recent-posts")
	if err != nil {
		t.Fatalf("ArticleByName(forum:recent-posts) err = %v, want nil", err)
	}
	categories, err := d.ForumCategories(ctx)
	if err != nil {
		t.Fatalf("ForumCategories() err = %v, want nil", err)
	}

	var b strings.Builder
	for _, c := range cases {
		var viewer *db.User
		if c.Viewer != "" {
			viewer, err = d.UserByUsername(ctx, c.Viewer)
			if err != nil {
				t.Fatalf("UserByUsername(%q) err = %v, want nil", c.Viewer, err)
			}
		}

		path := forumNewThreadPath(t, forumNewThreadCase{Category: c.Category, Path: c.Path}, categories)
		pc := page.NewContext(article, article, path, viewer)
		r := newThreadRepo(ctx, d, users, loc, site, viewer, rend)

		body, err := r.RenderModule(pc, "RecentPosts", c.Params, "")
		var moduleErr *callbacks.ModuleError
		switch {
		case errors.As(err, &moduleErr):
			body = "!error: " + moduleErr.Message
		case err != nil:
			t.Fatalf("RenderModule(%s) err = %v, want nil", c.Name, err)
		}
		fmt.Fprintf(&b, "=== %s\ntitle: %s\nstatus: %d\n%s\n", c.Name, pc.Title, pc.Status, body)
	}
	compareGolden(t, b.String(), cases, recentPostsGolden, recentPostsCorpus)
}
