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
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/printuser"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
	"github.com/WikitTeam/ProjectWikit/internal/renderer/sidecar"
)

const (
	countPagesGolden = "testdata/countpages.golden"
	countPagesCorpus = "testdata/countpages_corpus.json"
)

type countPagesCase struct {
	Name    string            `json:"name"`
	Body    string            `json:"body"`
	Params  map[string]string `json:"params,omitempty"`
	Path    map[string]string `json:"path,omitempty"`
	Article string            `json:"article,omitempty"`
}

func countPagesCases() []countPagesCase {
	return []countPagesCase{
		{Name: "category", Body: "%%total%%", Params: map[string]string{"category": "probe"}},
		{Name: "count-alias", Body: "%%count%%", Params: map[string]string{"category": "probe"}},
		{Name: "uppercase-name", Body: "%%TOTAL%%", Params: map[string]string{"category": "probe"}},
		{Name: "unknown-name", Body: "%%nope%%", Params: map[string]string{"category": "probe"}},
		{Name: "wikitext-around-it", Body: "**%%total%%** pages", Params: map[string]string{"category": "probe"}},
		{Name: "every-category", Body: "%%total%%", Params: map[string]string{"category": "*"}},
		{Name: "no-parameters", Body: "%%total%%"},
		{Name: "per-page-does-not-cut-it", Body: "%%total%%", Params: map[string]string{"category": "probe", "perPage": "2"}},
		{Name: "limit", Body: "%%total%%", Params: map[string]string{"category": "probe", "limit": "3"}},
		{Name: "offset", Body: "%%total%%", Params: map[string]string{"category": "probe", "offset": "2"}},
		{Name: "tags", Body: "%%total%%", Params: map[string]string{"category": "*", "tags": "lang:en"}},
		{Name: "full-name", Body: "%%total%%", Params: map[string]string{"fullname": "probe:full"}},
		{Name: "full-name-missing", Body: "%%total%%", Params: map[string]string{"fullname": "probe:no-such-page"}},
		{Name: "unparsable-parameter", Body: "%%total%%", Params: map[string]string{"category": "probe", "rating": "junk"}},
		{Name: "url-parameter-default", Body: "%%total%%", Params: map[string]string{"category": "@url|probe"}},
		{Name: "url-parameter-from-path", Body: "%%total%%", Params: map[string]string{"category": "@url|probe"},
			Path: map[string]string{"category": "probestars"}},
		{Name: "empty-body"},
		{Name: "body-without-a-variable", Body: "no variable here"},
	}
}

func TestCountPagesMatchesGolden(t *testing.T) {
	bin := os.Getenv(sidecar.EnvBinary)
	if bin == "" {
		t.Skipf("%s not set, skipping the count test", sidecar.EnvBinary)
	}
	ctx := context.Background()
	d := testDB(t)
	users, loc := testUsers(t)
	cases := countPagesCases()

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
		pc := page.NewContext(article, article, sortedPath(c.Path), nil)
		r := newRenderRepo(ctx, d, users, loc, site, nil, rend)

		body, err := r.RenderModule(pc, "CountPages", copyParams(c.Params), c.Body)
		var moduleErr *callbacks.ModuleError
		switch {
		case errors.As(err, &moduleErr):
			body = "!error: " + moduleErr.Message
		case err != nil:
			t.Fatalf("RenderModule(%s) err = %v, want nil", c.Name, err)
		}
		fmt.Fprintf(&b, "=== %s\n%s\n", c.Name, body)
	}
	compareGolden(t, b.String(), cases, countPagesGolden, countPagesCorpus)
}

func newRenderRepo(ctx context.Context, d *db.DB, users *printuser.Renderer, loc *i18n.Localizer,
	site *db.Site, viewer *db.User, rend renderer.Renderer) *Repository {

	var r *Repository
	source := NewVarSource(ctx, d, site)
	render := func(text string, pc *page.Context) (string, error) {
		cb := callbacks.New(loc, r)
		cb.SetContext(pc)
		info := renderer.PageInfo{Site: site.Slug, Domain: site.Domain, MediaDomain: site.MediaDomain}
		if pc.Article != nil {
			info.Page = pc.Article.Name
			info.Category = pc.Article.Category
			info.Title = pc.Article.Title
		}
		vars := page.NewVars(pc.Article, viewer, source, loc)
		html, err := rend.RenderHTML(ctx, page.ThisVars(text, vars), info, cb, renderer.ModeArticle)
		if err != nil {
			return "", err
		}
		return html.Body, nil
	}
	r = New(ctx, d, users, Options{Loc: loc, Site: site, User: viewer, Render: render, Vars: source})
	return r
}

func copyParams(params map[string]string) map[string]string {
	out := make(map[string]string, len(params))
	for key, value := range params {
		out[key] = value
	}
	return out
}
