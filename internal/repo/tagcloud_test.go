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
	tagCloudGolden = "testdata/tagcloud.golden"
	tagCloudCorpus = "testdata/tagcloud_corpus.json"
)

type tagCloudCase struct {
	Name   string            `json:"name"`
	Params map[string]string `json:"params,omitempty"`
}

func tagCloudCases() []tagCloudCase {
	return []tagCloudCase{
		{Name: "bare"},
		{Name: "categories", Params: map[string]string{"categories": "yes"}},
		{Name: "categories-no", Params: map[string]string{"categories": "no"}},
		{Name: "categories-junk", Params: map[string]string{"categories": "true"}},
		{Name: "limit", Params: map[string]string{"limit": "2"}},
		{Name: "limit-one", Params: map[string]string{"limit": "1"}},
		{Name: "limit-zero", Params: map[string]string{"limit": "0"}},
		{Name: "limit-past-the-end", Params: map[string]string{"limit": "99"}},
		{Name: "limit-junk", Params: map[string]string{"limit": "many"}},
		{Name: "limit-negative", Params: map[string]string{"limit": "-1"}},
		{Name: "limit-underscore", Params: map[string]string{"limit": "1_0"}},
		{Name: "limit-padded", Params: map[string]string{"limit": " 2 "}},
		{Name: "limit-signed", Params: map[string]string{"limit": "+2"}},
		{Name: "target", Params: map[string]string{"target": "probe:tagged"}},
		{Name: "target-empty", Params: map[string]string{"target": ""}},
		{Name: "font-sizes", Params: map[string]string{"minfontsize": "80%", "maxfontsize": "160%"}},
		{Name: "font-sizes-px", Params: map[string]string{"minfontsize": "10px", "maxfontsize": "40px"}},
		{Name: "font-size-min-only", Params: map[string]string{"minfontsize": "80%"}},
		{Name: "font-size-fraction", Params: map[string]string{"minfontsize": "1.5em", "maxfontsize": "3.5em"}},
		{Name: "font-size-mixed-units", Params: map[string]string{"minfontsize": "10px", "maxfontsize": "200%"}},
		{Name: "font-size-junk", Params: map[string]string{"minfontsize": "big", "maxfontsize": "bigger"}},
		{Name: "colors-hex", Params: map[string]string{"mincolor": "#ff0000", "maxcolor": "#0000ff"}},
		{Name: "colors-short-hex", Params: map[string]string{"mincolor": "#f00", "maxcolor": "#00f"}},
		{Name: "colors-rgb", Params: map[string]string{"mincolor": "rgb(255, 0, 0)", "maxcolor": "rgb(0,0,255)"}},
		{Name: "colors-bare", Params: map[string]string{"mincolor": "255,0,0", "maxcolor": "0, 0, 255"}},
		{Name: "colors-out-of-range", Params: map[string]string{"mincolor": "300,0,0", "maxcolor": "999,0,0"}},
		{Name: "colors-min-only", Params: map[string]string{"mincolor": "#ff0000"}},
		{Name: "colors-junk", Params: map[string]string{"mincolor": "red", "maxcolor": "blue"}},
		{Name: "colors-bad-hex", Params: map[string]string{"mincolor": "#ff00", "maxcolor": "#0000ff"}},
		{Name: "everything", Params: map[string]string{
			"categories": "yes", "limit": "3", "target": "probe:tagged",
			"minfontsize": "90%", "maxfontsize": "250%",
			"mincolor": "#102030", "maxcolor": "rgb(200, 100, 50)"}},
	}
}

func TestTagCloudMatchesGolden(t *testing.T) {
	ctx := context.Background()
	d := testDB(t)
	users, loc := testUsers(t)
	cases := tagCloudCases()

	article, err := d.ArticleByName(ctx, "main")
	if err != nil {
		t.Fatalf("ArticleByName(main) err = %v, want nil", err)
	}

	var b strings.Builder
	for _, c := range cases {
		pc := page.NewContext(article, article, nil, nil)
		r := New(ctx, d, users, Options{Loc: loc})

		body, err := r.RenderModule(pc, "TagCloud", copyParams(c.Params), "")
		var moduleErr *callbacks.ModuleError
		switch {
		case errors.As(err, &moduleErr):
			body = "!error: " + moduleErr.Message
		case err != nil:
			t.Fatalf("RenderModule(%s) err = %v, want nil", c.Name, err)
		}
		fmt.Fprintf(&b, "=== %s\n%s\n", c.Name, body)
	}
	compareGolden(t, b.String(), cases, tagCloudGolden, tagCloudCorpus)
}
