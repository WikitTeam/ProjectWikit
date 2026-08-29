package modules

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

var update = flag.Bool("update", false, "rewrite the golden file and the corpus the oracle reads")

const (
	rateGolden = "testdata/rate.golden"
	rateCorpus = "testdata/rate_corpus.json"
)

type rateCase struct {
	Name   string `json:"name"`
	Page   string `json:"page"`
	Viewer string `json:"viewer"`
}

func rateCases() []rateCase {
	pages := []string{
		"probe:full",
		"probe:half",
		"probe:bare",
		"probestars:rated",
		"probestars:unrated",
		"probestars:third",
		"probestars:quarter",
		"probeoff:unratable",
	}
	viewers := []string{"", "probevoter", "probe-author", "probecrowd0"}

	var out []rateCase
	for _, p := range pages {
		for _, v := range viewers {
			name := strings.ReplaceAll(p, ":", "-")
			if v == "" {
				name += "-anonymous"
			} else {
				name += "-" + v
			}
			out = append(out, rateCase{Name: name, Page: p, Viewer: v})
		}
	}
	return out
}

type rateData struct {
	ctx    context.Context
	d      *db.DB
	siteID int64
	module.Data
}

func (r rateData) SiteRatingMode() (string, error) {
	return r.d.SiteRatingMode(r.ctx, r.siteID)
}

func (r rateData) CategoryRatingMode(category string) (string, error) {
	return r.d.CategoryRatingMode(r.ctx, category)
}

func (r rateData) VoteStats(articleID int64) (db.VoteStats, error) {
	return r.d.VoteStats(r.ctx, articleID)
}

func (r rateData) HasVoted(articleID int64, userID *int64) (bool, error) {
	return r.d.HasVoted(r.ctx, articleID, userID)
}

func rateSource(t *testing.T) rateData {
	t.Helper()
	dsn := os.Getenv(db.EnvDSN)
	if dsn == "" {
		t.Skipf("%s not set, skipping the database test", db.EnvDSN)
	}
	ctx := context.Background()
	d, err := db.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("db.Open() err = %v, want nil", err)
	}
	t.Cleanup(d.Close)
	site, err := d.SiteByHosts(ctx, []string{"localhost"})
	if err != nil {
		t.Fatalf("SiteByHosts([localhost]) err = %v, want nil", err)
	}
	return rateData{ctx: ctx, d: d, siteID: site.ID}
}

func TestRenderRateMatchesGolden(t *testing.T) {
	src := rateSource(t)
	loc := rateLocalizer(t)
	cases := rateCases()

	var b strings.Builder
	for _, c := range cases {
		article, err := src.d.ArticleByName(src.ctx, c.Page)
		if err != nil {
			t.Fatalf("ArticleByName(%q) err = %v, want nil", c.Page, err)
		}
		var viewer *db.User
		if c.Viewer != "" {
			viewer, err = src.d.UserByUsername(src.ctx, c.Viewer)
			if err != nil {
				t.Fatalf("UserByUsername(%q) err = %v, want nil", c.Viewer, err)
			}
		}
		env := module.Env{
			Page: page.NewContext(article, article, nil, viewer),
			Loc:  loc,
			User: viewer,
			Data: src,
		}
		got, err := renderRate(env, nil, "")
		if err != nil {
			t.Fatalf("renderRate(%s) err = %v, want nil", c.Name, err)
		}
		fmt.Fprintf(&b, "=== %s\n%s\n", c.Name, got)
	}
	compareRateGolden(t, b.String(), cases)
}

func rateLocalizer(t *testing.T) *i18n.Localizer {
	t.Helper()
	bundle, err := i18n.Load("")
	if err != nil {
		t.Fatalf("i18n.Load() err = %v, want nil", err)
	}
	return bundle.Localizer(i18n.DefaultLanguage)
}

func compareRateGolden(t *testing.T, got string, corpus any) {
	t.Helper()
	if *update {
		data, err := json.MarshalIndent(corpus, "", "  ")
		if err != nil {
			t.Fatalf("Marshal(%s) err = %v, want nil", rateCorpus, err)
		}
		if err := os.WriteFile(filepath.FromSlash(rateCorpus), append(data, '\n'), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) err = %v, want nil", rateCorpus, err)
		}
		if err := os.WriteFile(filepath.FromSlash(rateGolden), []byte(got), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) err = %v, want nil", rateGolden, err)
		}
		return
	}
	want, err := os.ReadFile(filepath.FromSlash(rateGolden))
	if err != nil {
		t.Fatalf("ReadFile(%s) err = %v, want nil", rateGolden, err)
	}
	if got != string(want) {
		gotAt, wantAt := firstRateDiff(got, string(want))
		t.Errorf("renderRate = %q, want %q", gotAt, wantAt)
	}
}

func firstRateDiff(got, want string) (string, string) {
	at := 0
	for at < len(got) && at < len(want) && got[at] == want[at] {
		at++
	}
	cut := func(s string) string { return s[max(0, at-40):min(len(s), at+40)] }
	return cut(got), cut(want)
}
