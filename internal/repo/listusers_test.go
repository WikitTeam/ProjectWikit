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
	listUsersGolden = "testdata/listusers.golden"
	listUsersCorpus = "testdata/listusers_corpus.json"
)

const listUsersBody = "%%number%% %%title%% %%name%% %%avatar%% %%is_authenticated%%"

type listUsersCase struct {
	Name    string            `json:"name"`
	Viewer  string            `json:"viewer"`
	Body    string            `json:"body"`
	Params  map[string]string `json:"params,omitempty"`
	Article string            `json:"article,omitempty"`
}

func listUsersCases() []listUsersCase {
	return []listUsersCase{
		{Name: "anonymous-without-always", Body: listUsersBody},
		{Name: "anonymous-always", Body: listUsersBody, Params: map[string]string{"always": "yes"}},
		{Name: "anonymous-named", Body: listUsersBody, Params: map[string]string{"always": "yes", "anonname": "Guest"}},
		{Name: "anonymous-named-empty", Body: listUsersBody, Params: map[string]string{"always": "yes", "anonname": ""}},
		{Name: "viewer-with-display-name", Viewer: "probe-author", Body: listUsersBody},
		{Name: "viewer-without-display-name", Viewer: "probevoter", Body: listUsersBody},
		{Name: "viewer-from-wikidot", Viewer: "576c0df3-8a28-4468-9770-ede851d88c67", Body: listUsersBody},
		{Name: "viewer-with-always-off", Viewer: "probe-author", Body: listUsersBody, Params: map[string]string{"always": "no"}},
		{Name: "uppercase-name", Viewer: "probe-author", Body: "%%TITLE%%"},
		{Name: "unknown-name", Viewer: "probe-author", Body: "%%nope%%"},
		{Name: "empty-body", Viewer: "probe-author"},
		{Name: "authors", Body: "%%author%%", Params: map[string]string{"authors": "yes"}, Article: "probe:full"},
		{Name: "authors-linked", Body: "%%author_linked%%", Params: map[string]string{"authors": "yes"}, Article: "probe:full"},
		{Name: "authors-both", Body: "%%author%% is %%author_linked%%", Params: map[string]string{"authors": "yes"}, Article: "probe:full"},
		{Name: "authors-wikitext", Body: "**%%author%%**", Params: map[string]string{"authors": "yes"}, Article: "probe:full"},
		{Name: "authors-none", Body: "%%author%%", Params: map[string]string{"authors": "yes"}, Article: "probe:bare"},
		{Name: "authors-while-signed-in", Viewer: "probevoter", Body: "%%author%%",
			Params: map[string]string{"authors": "yes"}, Article: "probe:full"},
		{Name: "authors-off", Viewer: "probe-author", Body: listUsersBody, Params: map[string]string{"authors": "no"}},
	}
}

func TestListUsersMatchesGolden(t *testing.T) {
	bin := os.Getenv(sidecar.EnvBinary)
	if bin == "" {
		t.Skipf("%s not set, skipping the user list test", sidecar.EnvBinary)
	}
	ctx := context.Background()
	d := testDB(t)
	users, loc := testUsers(t)
	cases := listUsersCases()

	rend, err := sidecar.New(bin)
	if err != nil {
		t.Fatalf("sidecar.New(%q) err = %v, want nil", bin, err)
	}
	t.Cleanup(func() { rend.Close() })

	site, err := d.SiteByHosts(ctx, []string{"localhost"})
	if err != nil {
		t.Fatalf("SiteByHosts(localhost) err = %v, want nil", err)
	}

	var b strings.Builder
	for _, c := range cases {
		name := c.Article
		if name == "" {
			name = "main"
		}
		article, err := d.ArticleByName(ctx, name)
		if err != nil {
			t.Fatalf("ArticleByName(%q) err = %v, want nil", name, err)
		}
		var viewer *db.User
		if c.Viewer != "" {
			if viewer, err = d.UserByUsername(ctx, c.Viewer); err != nil {
				t.Fatalf("UserByUsername(%q) err = %v, want nil", c.Viewer, err)
			}
		}

		pc := page.NewContext(article, article, nil, viewer)
		r := newRenderRepo(ctx, d, users, loc, site, viewer, rend)

		body, err := r.RenderModule(pc, "ListUsers", copyParams(c.Params), c.Body)
		var moduleErr *callbacks.ModuleError
		switch {
		case errors.As(err, &moduleErr):
			body = "!error: " + moduleErr.Message
		case err != nil:
			t.Fatalf("RenderModule(%s) err = %v, want nil", c.Name, err)
		}
		fmt.Fprintf(&b, "=== %s\n%s\n", c.Name, body)
	}
	compareGolden(t, b.String(), cases, listUsersGolden, listUsersCorpus)
}
