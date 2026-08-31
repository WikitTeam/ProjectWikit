package modules

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	newPageGolden = "testdata/newpage.golden"
	newPageCorpus = "testdata/newpage_corpus.json"
)

type newPageCase struct {
	Name   string            `json:"name"`
	Params map[string]string `json:"params,omitempty"`
}

func newPageCases() []newPageCase {
	return []newPageCase{
		{Name: "bare"},
		{Name: "example", Params: map[string]string{"example": "scp-1000"}},
		{Name: "example-empty", Params: map[string]string{"example": ""}},
		{Name: "category", Params: map[string]string{"category": "scp"}},
		{Name: "category-empty", Params: map[string]string{"category": ""}},
		{Name: "submit", Params: map[string]string{"submit": "开始写"}},
		{Name: "submit-empty", Params: map[string]string{"submit": ""}},
		{Name: "all", Params: map[string]string{"example": "scp-1000", "category": "scp", "submit": "开始写"}},
		{Name: "quoted-example", Params: map[string]string{"example": `a"b'c<d&e`}},
		{Name: "quoted-category", Params: map[string]string{"category": `a"b'c<d&e`}},
		{Name: "quoted-submit", Params: map[string]string{"submit": `a"b'c<d&e`}},
		{Name: "unknown-param", Params: map[string]string{"unknown": "x"}},
	}
}

func TestRenderNewPageMatchesGolden(t *testing.T) {
	cases := newPageCases()
	env := forumEnv(t)

	var b strings.Builder
	for _, c := range cases {
		got, err := renderNewPage(env, c.Params, "")
		if err != nil {
			t.Fatalf("renderNewPage(%s) err = %v, want nil", c.Name, err)
		}
		fmt.Fprintf(&b, "=== %s\n%s\n", c.Name, got)
	}

	if *update {
		data, err := json.MarshalIndent(cases, "", "  ")
		if err != nil {
			t.Fatalf("Marshal(%s) err = %v, want nil", newPageCorpus, err)
		}
		if err := os.WriteFile(filepath.FromSlash(newPageCorpus), append(data, '\n'), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) err = %v, want nil", newPageCorpus, err)
		}
		if err := os.WriteFile(filepath.FromSlash(newPageGolden), []byte(b.String()), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) err = %v, want nil", newPageGolden, err)
		}
		return
	}

	want, err := os.ReadFile(filepath.FromSlash(newPageGolden))
	if err != nil {
		t.Fatalf("ReadFile(%s) err = %v, want nil", newPageGolden, err)
	}
	if got := b.String(); got != string(want) {
		gotAt, wantAt := firstRateDiff(got, string(want))
		t.Errorf("renderNewPage = %q, want %q", gotAt, wantAt)
	}
}

func TestRenderNewPageKeepsAnEmptySubmit(t *testing.T) {
	env := forumEnv(t)
	got, err := renderNewPage(env, map[string]string{"submit": ""}, "")
	if err != nil {
		t.Fatalf("renderNewPage() err = %v, want nil", err)
	}
	if !strings.Contains(got, `<input class="button" value="" type="submit">`) {
		t.Errorf("renderNewPage(submit=\"\") = %q, want an empty submit value", got)
	}
}
