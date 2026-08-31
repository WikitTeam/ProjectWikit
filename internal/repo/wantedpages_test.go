package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/callbacks"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

const (
	wantedPagesGolden = "testdata/wantedpages.golden"
	wantedPagesCorpus = "testdata/wantedpages_corpus.json"
)

type wantedPagesCase struct {
	Name   string            `json:"name"`
	Params map[string]string `json:"params,omitempty"`
	Path   map[string]string `json:"path,omitempty"`
}

func wantedPagesCases() []wantedPagesCase {
	return []wantedPagesCase{
		{Name: "bare"},
		{Name: "every-category", Params: map[string]string{"category": "*"}},
		{Name: "category-from", Params: map[string]string{"category_from": "probestars"}},
		{Name: "category-from-two", Params: map[string]string{"category_from": "probestars, probeoff"}},
		{Name: "category-from-default", Params: map[string]string{"category_from": "_default"}},
		{Name: "category-from-empty", Params: map[string]string{"category_from": "nosuchcategory"}},
		{Name: "category-to", Params: map[string]string{"category": "*", "category_to": "wanted"}},
		{Name: "category-to-default", Params: map[string]string{"category": "*", "category_to": "_default"}},
		{Name: "category-to-not", Params: map[string]string{"category": "*", "category_to": "-wanted"}},
		{Name: "category-to-two", Params: map[string]string{"category": "*", "category_to": "wanted, probe"}},
		{Name: "category-both", Params: map[string]string{"category_from": "probestars", "category_to": "wanted"}},
		{Name: "per-page", Params: map[string]string{"category": "*", "perpage": "3"}},
		{Name: "per-page-second", Params: map[string]string{"category": "*", "perpage": "3"},
			Path: map[string]string{"p": "2"}},
		{Name: "per-page-past-the-end", Params: map[string]string{"category": "*", "perpage": "3"},
			Path: map[string]string{"p": "99"}},
		{Name: "per-page-one", Params: map[string]string{"category": "*", "perpage": "1"}},
		{Name: "per-page-junk", Params: map[string]string{"category": "*", "perpage": "many"}},
		{Name: "limit-is-dropped", Params: map[string]string{"category": "*", "limit": "1"}},
		{Name: "offset-is-dropped", Params: map[string]string{"category": "*", "offset": "5"}},
		{Name: "tags-on-the-source", Params: map[string]string{"category": "*", "tags": "alpha"}},
		{Name: "unparsable-source-filter", Params: map[string]string{"category": "*", "rating": "junk"}},
	}
}

func TestWantedPagesMatchesGolden(t *testing.T) {
	ctx := context.Background()
	d := testDB(t)
	users, loc := testUsers(t)
	cases := wantedPagesCases()

	article, err := d.ArticleByName(ctx, "main")
	if err != nil {
		t.Fatalf("ArticleByName(main) err = %v, want nil", err)
	}

	var b strings.Builder
	for _, c := range cases {
		pc := page.NewContext(article, article, sortedPath(c.Path), nil)
		r := New(ctx, d, users, Options{Loc: loc})

		body, err := r.RenderModule(pc, "WantedPages", copyParams(c.Params), "")
		var moduleErr *callbacks.ModuleError
		switch {
		case errors.As(err, &moduleErr):
			body = "!error: " + moduleErr.Message
		case err != nil:
			t.Fatalf("RenderModule(%s) err = %v, want nil", c.Name, err)
		}
		fmt.Fprintf(&b, "=== %s\n%s\n", c.Name, body)
	}
	compareGolden(t, b.String(), cases, wantedPagesGolden, wantedPagesCorpus)
}
