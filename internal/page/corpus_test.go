package page_test

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
)

var update = flag.Bool("update", false, "rewrite testdata/vars.golden and testdata/vars_corpus.json")

const (
	goldenPath = "testdata/vars.golden"
	corpusPath = "testdata/vars_corpus.json"
	envHost    = "PWIKIT_TEST_HOST"
)

type varsCorpus struct {
	Articles []corpusArticle `json:"articles"`
	Names    []string        `json:"names"`
}

type corpusArticle struct {
	Name   string `json:"name"`
	Viewer string `json:"viewer"`
}

var corpus = varsCorpus{
	Articles: []corpusArticle{
		{Name: "probe:full", Viewer: "probe-author"},
		{Name: "probe:full", Viewer: ""},
		{Name: "probe:bare", Viewer: ""},
		{Name: "probe:parent", Viewer: ""},
		{Name: "probestars:rated", Viewer: "probevoter"},
		{Name: "probe:half", Viewer: ""},
	},
	Names: []string{
		"name", "category", "fullname", "title", "title_linked", "linked_title", "link",
		"content", "rating", "rating_votes", "popularity", "current_user_voted", "revisions",
		"created_by", "created_by_linked", "created_by_linked_plain",
		"updated_by", "updated_by_linked", "updated_by_linked_plain", "authors_count",
		"tags", "tags_linked", "created_at", "updated_at",
		"parent_name", "parent_category", "parent_fullname", "parent_title",
		"parent_title_linked", "parent_linked_title",
		"created_at|%Y", "updated_at|%d.%m.%Y",
		"this|title", "this|TITLE", "this|fullname", "this|nosuchvar",
		"notthis", "Title",
	},
}

func resolveForCorpus(v *page.Vars, name string) string {
	if strings.HasPrefix(name, "this|") {
		if value, ok := v.This(name); ok {
			return value
		}
		return "%%" + name + "%%"
	}
	if value, ok := v.Lookup(name); ok {
		return value
	}
	return "%%" + name + "%%"
}

func TestVarsMatchOracle(t *testing.T) {
	dsn := os.Getenv(db.EnvDSN)
	if dsn == "" {
		t.Skipf("%s not set", db.EnvDSN)
	}

	ctx := context.Background()
	conn, err := db.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open() = %v, want nil", err)
	}
	defer conn.Close()

	host := os.Getenv(envHost)
	if host == "" {
		host = "localhost"
	}
	site, err := conn.SiteByHosts(ctx, []string{host})
	if err != nil {
		t.Fatalf("SiteByHosts(%q) = %v, want nil; set %s", host, err, envHost)
	}

	bundle, err := i18n.Load("")
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	loc := bundle.Localizer("zh-hans")

	var b strings.Builder
	for _, entry := range corpus.Articles {
		article, err := conn.ArticleByName(ctx, entry.Name)
		if err != nil {
			t.Fatalf("ArticleByName(%q) = %v, want nil; run testdata/oracle_seed.py", entry.Name, err)
		}
		var viewer *db.User
		if entry.Viewer != "" {
			viewer, err = conn.UserByName(ctx, entry.Viewer)
			if err != nil {
				t.Fatalf("UserByName(%q) = %v, want nil", entry.Viewer, err)
			}
		}
		vars := page.NewVars(article, viewer, repo.NewVarSource(ctx, conn, site.ID), loc)
		for _, name := range corpus.Names {
			b.WriteString("=== " + entry.Name + " " + name + "\n" + resolveForCorpus(vars, name) + "\n")
		}
		if err := vars.Err(); err != nil {
			t.Fatalf("Err() = %v, want nil", err)
		}
	}
	got := b.String()

	if *update {
		encoded, err := json.MarshalIndent(corpus, "", "  ")
		if err != nil {
			t.Fatalf("MarshalIndent() = %v, want nil", err)
		}
		if err := os.WriteFile(corpusPath, append(encoded, '\n'), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) = %v, want nil", corpusPath, err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) = %v, want nil", goldenPath, err)
		}
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) = %v, want nil", goldenPath, err)
	}
	if got != string(want) {
		gotLines, wantLines := strings.Split(got, "\n"), strings.Split(string(want), "\n")
		for i := range max(len(gotLines), len(wantLines)) {
			g, w := lineAt(gotLines, i), lineAt(wantLines, i)
			if g != w {
				t.Fatalf("golden line %d = %q, want %q", i+1, g, w)
			}
		}
	}
}

func lineAt(lines []string, i int) string {
	if i < len(lines) {
		return lines[i]
	}
	return ""
}
