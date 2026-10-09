package db

import (
	"context"
	"slices"
	"testing"
	"time"
)

func suggestValues(got []Suggestion) []string {
	out := make([]string, 0, len(got))
	for _, s := range got {
		out = append(out, s.Value)
	}
	return out
}

func TestSuggestionsLeaveOutUnindexedTagsAndCategories(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	probe := listedPages(t, d)
	at := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, name := range []string{"open", "shut"} {
		if _, err := d.CreateArticle(ctx, probe.site, "cat"+name, "page", "page", nil, at); err != nil {
			t.Fatalf("CreateArticle(cat%s) err = %v, want nil", name, err)
		}
	}
	if _, err := d.pool.Exec(ctx, `INSERT INTO web_category (name, is_indexed, site_id) VALUES ('catshut', false, $1)`, probe.site); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		d.pool.Exec(context.Background(), `DELETE FROM web_category WHERE site_id = $1`, probe.site)
	})
	if _, err := d.pool.Exec(ctx, `UPDATE web_tag SET is_indexed = false WHERE site_id = $1 AND name = 'z'`, probe.site); err != nil {
		t.Fatal(err)
	}

	tags, err := d.SuggestTags(ctx, probe.site, "", 10)
	if err != nil {
		t.Fatalf("SuggestTags() err = %v, want nil", err)
	}
	if got := suggestValues(tags); !slices.Equal(got, []string{"x", "y"}) {
		t.Errorf("SuggestTags() = %v, want [x y]", got)
	}

	cats, err := d.SuggestCategories(ctx, probe.site, nil, "cat", 10)
	if err != nil {
		t.Fatalf("SuggestCategories() err = %v, want nil", err)
	}
	if got := suggestValues(cats); !slices.Equal(got, []string{"catopen"}) {
		t.Errorf("SuggestCategories(cat) = %v, want [catopen]", got)
	}

	hidden, err := d.SuggestCategories(ctx, probe.site, []string{"catopen"}, "cat", 10)
	if err != nil {
		t.Fatalf("SuggestCategories(hidden) err = %v, want nil", err)
	}
	if len(hidden) != 0 {
		t.Errorf("SuggestCategories(hidden catopen) = %v, want none", suggestValues(hidden))
	}
}

func TestSuggestAuthorsOnlyListsAuthorsOfTheSite(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	probe := listedPages(t, d)
	author := scratchImportedUser(t, d, "Suggested")
	scratchImportedUser(t, d, "Suggestedelsewhere")
	if _, err := d.pool.Exec(ctx, `INSERT INTO web_article_authors (article_id, user_id) VALUES ($1, $2)`, probe.ids["alpha"], author); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		d.pool.Exec(context.Background(), `DELETE FROM web_article_authors WHERE user_id = $1`, author)
	})

	got, err := d.SuggestAuthors(ctx, probe.site, "suggested", 10)
	if err != nil {
		t.Fatalf("SuggestAuthors() err = %v, want nil", err)
	}
	if len(got) != 1 {
		t.Fatalf("SuggestAuthors(suggested) = %v, want one author", got)
	}
	if got[0].Label[:3] != "wd:" {
		t.Errorf("SuggestAuthors()[0].Label = %q, want a wd: label", got[0].Label)
	}
}
