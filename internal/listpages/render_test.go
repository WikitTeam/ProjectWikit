package listpages

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

var update = flag.Bool("update", false, "rewrite the golden files and the corpora the oracles read")

const (
	pagerGolden    = "testdata/pagination.golden"
	pagerCorpus    = "testdata/pagination_corpus.json"
	sectionsGolden = "testdata/sections.golden"
	sectionsCorpus = "testdata/sections_corpus.json"
)

type pagerCase struct {
	Name     string `json:"name"`
	BasePath string `json:"base_path"`
	Page     int    `json:"page"`
	Total    int    `json:"total_pages"`
}

func pagerCases() []pagerCase {
	var out []pagerCase
	add := func(name, basePath string, page, total int) {
		out = append(out, pagerCase{Name: name, BasePath: basePath, Page: page, Total: total})
	}
	add("single-page", "/main", 1, 1)
	add("no-pages", "/main", 1, 0)
	for _, total := range []int{2, 3, 4, 5, 6, 7, 9, 12, 30} {
		for _, current := range []int{1, 2, 3, 4, 5, 6, total - 1, total} {
			if current < 1 || current > total {
				continue
			}
			add(fmt.Sprintf("total-%d-page-%d", total, current), "/main", current, total)
		}
	}
	add("hash-base", "#", 3, 9)
	add("base-with-params", "/scp:series/tag/euclid", 2, 4)
	add("base-needing-escape", `/main/q/a"b&c`, 2, 4)
	return out
}

func localizer(t *testing.T) *i18n.Localizer {
	t.Helper()
	bundle, err := i18n.Load("")
	if err != nil {
		t.Fatalf("i18n.Load() err = %v, want nil", err)
	}
	return bundle.Localizer(i18n.DefaultLanguage)
}

func TestPaginationMatchesGolden(t *testing.T) {
	loc := localizer(t)
	cases := pagerCases()

	var b strings.Builder
	for _, c := range cases {
		fmt.Fprintf(&b, "=== %s\n%s\n", c.Name, Pagination(loc, c.BasePath, c.Page, c.Total))
	}
	compareGolden(t, pagerGolden, b.String(), pagerCorpus, cases)
}

type sectionCase struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

func sectionCases() []sectionCase {
	return []sectionCase{
		{"empty", ""},
		{"plain", "%%title%%"},
		{"head-only", "[[head]]\nbefore\n[[/head]]"},
		{"body-only", "[[body]]\n%%title%%\n[[/body]]"},
		{"foot-only", "[[foot]]\nafter\n[[/foot]]"},
		{"all-three", "[[head]]\nbefore\n[[/head]]\n[[body]]\n%%title%%\n[[/body]]\n[[foot]]\nafter\n[[/foot]]"},
		{"no-newline-after-open", "[[head]]before[[/head]]"},
		{"uppercase-tags", "[[HEAD]]\nbefore\n[[/HEAD]]"},
		{"multiline-body", "[[body]]\none\ntwo\nthree\n[[/body]]"},
		{"leading-text", "junk\n[[body]]\nrow\n[[/body]]"},
		{"empty-body", "[[body]]\n[[/body]]"},
		{"body-before-head", "[[body]]\nrow\n[[/body]]\n[[head]]\ntop\n[[/head]]"},
	}
}

func TestSplitMatchesGolden(t *testing.T) {
	cases := sectionCases()

	var b strings.Builder
	for _, c := range cases {
		s := Split(c.Content)
		fmt.Fprintf(&b, "=== %s\nhead=%s\nbody=%s\nfoot=%s\n", c.Name,
			wikijson.String(s.Head), wikijson.String(s.Body), wikijson.String(s.Foot))
	}
	compareGolden(t, sectionsGolden, b.String(), sectionsCorpus, cases)
}

func compareGolden(t *testing.T, goldenPath, got, corpusPath string, corpus any) {
	t.Helper()
	if *update {
		writeJSON(t, corpusPath, corpus)
		if err := os.WriteFile(filepath.FromSlash(goldenPath), []byte(got), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) err = %v, want nil", goldenPath, err)
		}
		return
	}
	want, err := os.ReadFile(filepath.FromSlash(goldenPath))
	if err != nil {
		t.Fatalf("ReadFile(%s) err = %v, want nil", goldenPath, err)
	}
	if got != string(want) {
		gotAt, wantAt := firstDiff(got, string(want))
		t.Errorf("render = %q, want %q", gotAt, wantAt)
	}
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("Marshal(%s) err = %v, want nil", path, err)
	}
	if err := os.WriteFile(filepath.FromSlash(path), append(data, '\n'), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) err = %v, want nil", path, err)
	}
}

func firstDiff(got, want string) (string, string) {
	for i := 0; i < len(got) && i < len(want); i++ {
		if got[i] != want[i] {
			return excerpt(got, i), excerpt(want, i)
		}
	}
	at := min(len(got), len(want))
	return excerpt(got, at), excerpt(want, at)
}

func excerpt(s string, at int) string {
	return s[max(0, at-40):min(len(s), at+40)]
}

func TestBasePathDropsThePageNumber(t *testing.T) {
	params := page.PathParams{
		{Key: "tag", Value: "euclid"},
		{Key: "p", Value: "3"},
		{Key: "sort", Value: "a b"},
	}
	got := BasePath("scp:series", params)
	want := "/scp:series/tag/euclid/sort/a+b"
	if got != want {
		t.Errorf("BasePath(scp:series, ...) = %q, want %q", got, want)
	}
}

func TestBasePathSpellsABareKeyAsNone(t *testing.T) {
	got := BasePath("main", page.PathParams{{Key: "edit", Bare: true}})
	if want := "/main/edit/None"; got != want {
		t.Errorf("BasePath(main, edit) = %q, want %q", got, want)
	}
}

func TestBasePathWithoutAPage(t *testing.T) {
	if got := BasePath("", nil); got != "#" {
		t.Errorf("BasePath(\"\", nil) = %q, want %q", got, "#")
	}
}

func TestURLParamsReadsThePath(t *testing.T) {
	params := map[string]string{"tags": "@url|default", "category": "scp"}
	got, null := URLParams(params, page.PathParams{{Key: "tags", Value: "euclid"}})
	if got["tags"] != "euclid" {
		t.Errorf("URLParams()[tags] = %q, want %q", got["tags"], "euclid")
	}
	if got["category"] != "scp" {
		t.Errorf("URLParams()[category] = %q, want %q", got["category"], "scp")
	}
	if len(null) != 0 {
		t.Errorf("len(null) = %d, want 0", len(null))
	}
}

func TestURLParamsFallsBackToTheDefault(t *testing.T) {
	got, _ := URLParams(map[string]string{"tags": "@url|euclid"}, nil)
	if got["tags"] != "euclid" {
		t.Errorf("URLParams()[tags] = %q, want %q", got["tags"], "euclid")
	}
}

func TestURLParamsMarksABareKeyAsNull(t *testing.T) {
	_, null := URLParams(map[string]string{"tags": "@url|x"}, page.PathParams{{Key: "tags", Bare: true}})
	if !null["tags"] {
		t.Errorf("null[tags] = false, want true")
	}
}

func TestURLParamsIgnoresCaseOfThePrefix(t *testing.T) {
	got, _ := URLParams(map[string]string{"tags": "@URL|euclid"}, nil)
	if got["tags"] != "euclid" {
		t.Errorf("URLParams()[tags] = %q, want %q", got["tags"], "euclid")
	}
}

func TestWrapEscapesTheDataAttributes(t *testing.T) {
	got := Wrap("body", "", `{"a": "b"}`, `{}`, `"x"`, "main")
	if !strings.Contains(got, `data-list-pages-path-params="{&quot;a&quot;: &quot;b&quot;}"`) {
		t.Errorf("Wrap() = %q, want the path params escaped", got)
	}
	if !strings.HasPrefix(got, `<div class="list-pages-box w-list-pages"`+"\n") {
		t.Errorf("Wrap() = %q, want the box to open on its own line", got)
	}
	if !strings.HasSuffix(got, "\n                </div>") {
		t.Errorf("Wrap() = %q, want the box to close indented", got)
	}
}
