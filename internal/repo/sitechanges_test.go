package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/callbacks"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

const (
	siteChangesGolden = "testdata/sitechanges.golden"
	siteChangesCorpus = "testdata/sitechanges_corpus.json"
)

type siteChangesCase struct {
	Name   string            `json:"name"`
	Viewer string            `json:"viewer"`
	Params map[string]string `json:"params,omitempty"`
	Path   map[string]string `json:"path,omitempty"`
	Bare   []string          `json:"bare,omitempty"`
}

func siteChangesCases() []siteChangesCase {
	return []siteChangesCase{
		{Name: "all", Viewer: ""},
		{Name: "signed-in", Viewer: "probe-author", Path: map[string]string{"perpage": "5"}},
		{Name: "second-page", Path: map[string]string{"p": "2"}},
		{Name: "last-page", Path: map[string]string{"p": "4"}},
		{Name: "past-the-end", Path: map[string]string{"p": "99", "perpage": "5"}},
		{Name: "below-one", Path: map[string]string{"p": "-3", "perpage": "5"}},
		{Name: "junk-page", Path: map[string]string{"p": "later", "perpage": "5"}},
		{Name: "perpage-10", Path: map[string]string{"p": "3", "perpage": "10"}},
		{Name: "perpage-200", Path: map[string]string{"category": "nav", "perpage": "200"}},
		{Name: "last-partial-page", Path: map[string]string{"p": "12", "perpage": "7"}},
		{Name: "perpage-junk", Path: map[string]string{"perpage": "many"}},
		{Name: "perpage-zero", Path: map[string]string{"perpage": "0"}},
		{Name: "perpage-negative", Path: map[string]string{"perpage": "-5"}},
		{Name: "perpage-bare", Bare: []string{"perpage"}},
		{Name: "category-probe", Path: map[string]string{"perpage": "5", "category": "probe"}},
		{Name: "category-uppercase", Path: map[string]string{"perpage": "5", "category": "PROBE"}},
		{Name: "category-default", Path: map[string]string{"perpage": "5", "category": "_default"}},
		{Name: "category-star", Path: map[string]string{"perpage": "5", "category": "*"}},
		{Name: "category-missing", Path: map[string]string{"perpage": "5", "category": "nosuchcategory"}},
		{Name: "category-bare", Bare: []string{"category"}},
		{Name: "user-exact", Path: map[string]string{"perpage": "5", "username": "probe-author"}},
		{Name: "user-uppercase", Path: map[string]string{"perpage": "5", "username": "PROBE-AUTHOR"}},
		{Name: "user-padded", Path: map[string]string{"perpage": "5", "username": "  probe-author  "}},
		{Name: "user-wikidot", Path: map[string]string{"perpage": "5", "username": "probe-wd-original"}},
		{Name: "user-system", Path: map[string]string{"perpage": "5", "username": "system"}},
		{Name: "user-unknown", Path: map[string]string{"perpage": "5", "username": "nobody-at-all"}},
		{Name: "user-partial", Path: map[string]string{"perpage": "5", "username": "~probe"}},
		{Name: "user-partial-system", Path: map[string]string{"perpage": "5", "username": "~sys"}},
		{Name: "user-partial-empty", Path: map[string]string{"perpage": "5", "username": "~"}},
		{Name: "user-partial-wildcard", Path: map[string]string{"perpage": "5", "username": "~%"}},
		{Name: "types-source", Path: map[string]string{"perpage": "5", "source": "true"}},
		{Name: "types-tags", Path: map[string]string{"perpage": "5", "tags": "true"}},
		{Name: "types-revert", Path: map[string]string{"perpage": "5", "revert": "true"}},
		{Name: "types-title", Path: map[string]string{"perpage": "5", "title": "true"}},
		{Name: "types-two", Path: map[string]string{"perpage": "5", "source": "true", "title": "true"}},
		{Name: "types-not-true", Path: map[string]string{"perpage": "5", "source": "false"}},
		{Name: "types-mixed-case", Path: map[string]string{"perpage": "5", "source": "TRUE"}},
		{Name: "types-unknown", Path: map[string]string{"perpage": "5", "nosuchtype": "true"}},
		{Name: "types-bare", Bare: []string{"source"}},
		{Name: "with-params", Params: map[string]string{"limit": "5"}, Path: map[string]string{"perpage": "5"}},
		{Name: "everything-at-once", Viewer: "probe-author",
			Path: map[string]string{"perpage": "5", "category": "probe", "username": "~probe", "source": "true"}},
	}
}

func TestSiteChangesMatchesGolden(t *testing.T) {
	ctx := context.Background()
	d := testDB(t)
	users, loc := testUsers(t)
	cases := siteChangesCases()

	site, err := d.SiteByHosts(ctx, []string{"localhost"})
	if err != nil {
		t.Fatalf("SiteByHosts(localhost) err = %v, want nil", err)
	}
	article, err := d.ArticleByName(ctx, "probe:changes")
	if err != nil {
		t.Fatalf("ArticleByName(probe:changes) err = %v, want nil", err)
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

		path := sortedPath(c.Path)
		for _, key := range c.Bare {
			path = path.Put(page.PathParam{Key: key, Bare: true})
		}
		pc := page.NewContext(article, article, path, viewer)
		r := New(ctx, d, users, Options{Loc: loc, Site: site, User: viewer})

		body, err := r.RenderModule(pc, "SiteChanges", c.Params, "")
		var moduleErr *callbacks.ModuleError
		switch {
		case errors.As(err, &moduleErr):
			body = "!error: " + moduleErr.Message
		case err != nil:
			t.Fatalf("RenderModule(%s) err = %v, want nil", c.Name, err)
		}
		fmt.Fprintf(&b, "=== %s\ntitle: %s\nstatus: %d\n%s\n", c.Name, pc.Title, pc.Status, body)
	}
	compareGolden(t, b.String(), cases, siteChangesGolden, siteChangesCorpus)
}
