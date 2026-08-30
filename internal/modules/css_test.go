package modules

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

const (
	cssGolden = "testdata/css.golden"
	cssCorpus = "testdata/css_corpus.json"
)

type cssCase struct {
	Name   string            `json:"name"`
	Body   string            `json:"body"`
	Params map[string]string `json:"params,omitempty"`
	Path   map[string]string `json:"path,omitempty"`
}

func cssCases() []cssCase {
	return []cssCase{
		{Name: "plain", Body: "#page-content { color : red ; }"},
		{Name: "head", Body: "#page-content { color : red ; }", Params: map[string]string{"head": "true"}},
		{Name: "head-from-path", Body: "a { color : red }", Path: map[string]string{"head": "yes"}},
		{Name: "head-false", Body: "a { color : red }", Params: map[string]string{"head": "no"}},
		{Name: "head-junk", Body: "a { color : red }", Params: map[string]string{"head": "maybe"}},
		{Name: "empty", Body: ""},
		{Name: "comment-only", Body: "/* nothing but a comment */"},
		{Name: "nbsp", Body: "a { color:red }"},
		{Name: "angle-bracket", Body: "a > b { content: \"</style>\" }"},
		{Name: "at-import", Body: "@import url(\"x.css\");\na { color: red }"},
		{Name: "media", Body: "@media (max-width: 767px) {\n  #main { padding : 0 ; }\n}"},
		{Name: "two-rules", Body: ".a { color: #ff0000 }\n.b { margin: 0.50em }"},
		{Name: "malformed", Body: "a { color: red"},
	}
}

func cssEnv(path page.PathParams) module.Env {
	return module.Env{Page: page.NewContext(&db.Article{ID: 1, Name: "main", Category: "_default"}, nil, path, nil)}
}

func TestRenderCSSMatchesGolden(t *testing.T) {
	cases := cssCases()

	var b strings.Builder
	for _, c := range cases {
		env := cssEnv(sortedCSSPath(c.Path))
		got, err := renderCSS(env, c.Params, c.Body)
		if err != nil {
			t.Fatalf("renderCSS(%s) err = %v, want nil", c.Name, err)
		}
		fmt.Fprintf(&b, "=== %s\nreturned: %q\ncomputed: %q\naddcss: %q\n",
			c.Name, got, env.Page.ComputedStyle, env.Page.AddCSS)
	}
	compareCSSGolden(t, b.String(), cases)
}

func sortedCSSPath(params map[string]string) page.PathParams {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var path page.PathParams
	for _, key := range keys {
		path = path.Put(page.PathParam{Key: key, Value: params[key]})
	}
	return path
}

func compareCSSGolden(t *testing.T, got string, corpus any) {
	t.Helper()
	if *update {
		data, err := json.MarshalIndent(corpus, "", "  ")
		if err != nil {
			t.Fatalf("Marshal(%s) err = %v, want nil", cssCorpus, err)
		}
		if err := os.WriteFile(filepath.FromSlash(cssCorpus), append(data, '\n'), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) err = %v, want nil", cssCorpus, err)
		}
		if err := os.WriteFile(filepath.FromSlash(cssGolden), []byte(got), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) err = %v, want nil", cssGolden, err)
		}
		return
	}
	want, err := os.ReadFile(filepath.FromSlash(cssGolden))
	if err != nil {
		t.Fatalf("ReadFile(%s) err = %v, want nil", cssGolden, err)
	}
	if got != string(want) {
		gotAt, wantAt := firstRateDiff(got, string(want))
		t.Errorf("renderCSS = %q, want %q", gotAt, wantAt)
	}
}
