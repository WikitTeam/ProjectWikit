package db

import (
	"context"
	"strings"
	"testing"
)

func TestSearchWhereKeepsUnindexedContentOut(t *testing.T) {
	b := &listBuilder{}
	got := SearchFilter{SiteID: 1}.where(b)
	for _, want := range []string{
		"a.is_indexed",
		"FROM web_category c",
		"NOT c.is_indexed",
		"JOIN web_tag t ON t.id = ax.tag_id",
		"NOT t.is_indexed",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Contains(where(), %q) = false, want true", want)
		}
	}
}

func TestSetArticleIndexedGoesBothWays(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	article := scratchArticle(t, d)
	site := seedSiteID(t, d)

	for _, want := range []bool{false, true} {
		if err := d.SetArticleIndexed(ctx, site, article, want); err != nil {
			t.Fatalf("SetArticleIndexed(%t) err = %v, want nil", want, err)
		}
		var got bool
		err := d.pool.QueryRow(ctx, `SELECT is_indexed FROM web_article WHERE id = $1`, article).Scan(&got)
		if err != nil {
			t.Fatalf("read is_indexed err = %v, want nil", err)
		}
		if got != want {
			t.Errorf("is_indexed = %t, want %t", got, want)
		}
	}
}

func TestSetArticleIndexedIsNotFoundForAnotherSite(t *testing.T) {
	d := writeTestDB(t)
	article := scratchArticle(t, d)
	if err := d.SetArticleIndexed(context.Background(), -1, article, false); err == nil {
		t.Error("SetArticleIndexed() err = nil, want ErrNotFound")
	}
}
