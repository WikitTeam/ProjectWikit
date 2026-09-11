package db

import (
	"context"
	"slices"
	"testing"
	"time"
)

type listedProbe struct {
	site int64
	ids  map[string]int64
}

func listedPages(t *testing.T, d *DB) listedProbe {
	t.Helper()
	ctx := context.Background()
	site := scratchSite(t, d)
	t.Cleanup(func() {
		clean := context.Background()
		for _, sql := range []string{
			`DELETE FROM web_article_tags WHERE article_id IN (SELECT id FROM web_article WHERE site_id = $1)`,
			`DELETE FROM web_tag WHERE category_id IN (SELECT id FROM web_tagscategory WHERE site_id = $1)`,
			`DELETE FROM web_tagscategory WHERE site_id = $1`,
			`DELETE FROM web_articlelogentry WHERE article_id IN (SELECT id FROM web_article WHERE site_id = $1)`,
			`DELETE FROM web_articleversion WHERE article_id IN (SELECT id FROM web_article WHERE site_id = $1)`,
			`DELETE FROM web_article_authors WHERE article_id IN (SELECT id FROM web_article WHERE site_id = $1)`,
			`DELETE FROM web_article WHERE site_id = $1`,
		} {
			if _, err := d.pool.Exec(clean, sql, site); err != nil {
				t.Errorf("clean up pages of site %d err = %v, want nil", site, err)
			}
		}
	})
	pages := []struct {
		name    string
		created time.Time
		tags    []string
	}{
		{"alpha", time.Date(2021, 2, 9, 15, 0, 0, 0, time.UTC), []string{"x", "y"}},
		{"beta", time.Date(2021, 2, 10, 0, 0, 0, 0, time.UTC), []string{"x", "y", "z"}},
		{"gamma", time.Date(2021, 12, 31, 23, 30, 0, 0, time.UTC), []string{"x"}},
	}
	probe := listedProbe{site: site, ids: map[string]int64{}}
	for _, p := range pages {
		id, err := d.CreateArticle(ctx, site, "_default", p.name, p.name, nil, p.created)
		if err != nil {
			t.Fatalf("CreateArticle(%s) err = %v, want nil", p.name, err)
		}
		if _, _, err := d.SetArticleTags(ctx, site, id, p.tags, true, nil, p.created); err != nil {
			t.Fatalf("SetArticleTags(%s) err = %v, want nil", p.name, err)
		}
		if _, err := d.pool.Exec(ctx, `UPDATE web_article SET created_at = $2 WHERE id = $1`, id, p.created); err != nil {
			t.Fatal(err)
		}
		probe.ids[p.name] = id
	}
	return probe
}

func (p listedProbe) names(t *testing.T, d *DB, f ListFilter) []string {
	t.Helper()
	f.SiteID = p.site
	listed, err := d.ListArticles(context.Background(), f, 0, nil)
	if err != nil {
		t.Fatalf("ListArticles() err = %v, want nil", err)
	}
	var out []string
	for _, a := range listed {
		out = append(out, a.Name)
	}
	return out
}

func TestListArticlesExactTags(t *testing.T) {
	d := writeTestDB(t)
	p := listedPages(t, d)
	var tags []int64
	if err := d.pool.QueryRow(context.Background(), `
SELECT array_agg(tag_id ORDER BY tag_id) FROM web_article_tags WHERE article_id = $1`, p.ids["alpha"]).Scan(&tags); err != nil {
		t.Fatal(err)
	}

	got := p.names(t, d, ListFilter{ExactTags: tags})
	if !slices.Equal(got, []string{"alpha"}) {
		t.Errorf("ListArticles(ExactTags of alpha) = %v, want [alpha]", got)
	}
	got = p.names(t, d, ListFilter{RequiredTags: tags, Sort: Sort{Column: SortName, Ascending: true}})
	if !slices.Equal(got, []string{"alpha", "beta"}) {
		t.Errorf("ListArticles(RequiredTags of alpha) = %v, want [alpha beta]", got)
	}
}

func TestListArticlesNotID(t *testing.T) {
	d := writeTestDB(t)
	p := listedPages(t, d)
	got := p.names(t, d, ListFilter{NotID: new(p.ids["beta"]), Sort: Sort{Column: SortName, Ascending: true}})
	if !slices.Equal(got, []string{"alpha", "gamma"}) {
		t.Errorf("ListArticles(NotID beta) = %v, want [alpha gamma]", got)
	}
}

func TestListArticlesSortsByCreatedAtBothWays(t *testing.T) {
	d := writeTestDB(t)
	p := listedPages(t, d)
	got := p.names(t, d, ListFilter{Sort: Sort{Column: SortCreatedAt, Ascending: true}})
	if !slices.Equal(got, []string{"alpha", "beta", "gamma"}) {
		t.Errorf("ListArticles(created_at asc) = %v, want [alpha beta gamma]", got)
	}
	got = p.names(t, d, ListFilter{Sort: Sort{Column: SortCreatedAt}})
	if !slices.Equal(got, []string{"gamma", "beta", "alpha"}) {
		t.Errorf("ListArticles(created_at desc) = %v, want [gamma beta alpha]", got)
	}
}

func TestListArticlesTimeFilterCoversTheWholePeriod(t *testing.T) {
	d := writeTestDB(t)
	p := listedPages(t, d)
	day := time.Date(2021, 2, 9, 0, 0, 0, 0, time.UTC)
	year := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		f    TimeFilter
		want []string
	}{
		{"range 2021-02-09", TimeFilter{Op: TimeRange, Start: day, End: day.AddDate(0, 0, 1)}, []string{"alpha"}},
		{"range 2021", TimeFilter{Op: TimeRange, Start: year, End: year.AddDate(1, 0, 0)}, []string{"alpha", "beta", "gamma"}},
		{"exclude 2021-02-09", TimeFilter{Op: TimeExcludeRange, Start: day, End: day.AddDate(0, 0, 1)}, []string{"beta", "gamma"}},
		{"lt 2021-02-09", TimeFilter{Op: TimeLT, Start: day, End: day.AddDate(0, 0, 1)}, nil},
		{"lte 2021-02-09", TimeFilter{Op: TimeLTE, Start: day, End: day.AddDate(0, 0, 1)}, []string{"alpha"}},
		{"gt 2021-02-09", TimeFilter{Op: TimeGT, Start: day, End: day.AddDate(0, 0, 1)}, []string{"beta", "gamma"}},
		{"gte 2021-02-09", TimeFilter{Op: TimeGTE, Start: day, End: day.AddDate(0, 0, 1)}, []string{"alpha", "beta", "gamma"}},
	}
	for _, c := range cases {
		f := c.f
		got := p.names(t, d, ListFilter{CreatedAt: &f, Sort: Sort{Column: SortName, Ascending: true}})
		if !slices.Equal(got, c.want) {
			t.Errorf("ListArticles(%s) = %v, want %v", c.name, got, c.want)
		}
	}
}
