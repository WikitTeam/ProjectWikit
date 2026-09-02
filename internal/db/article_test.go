package db

import (
	"context"
	"errors"
	"testing"
)

func TestDumbName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"main", "_default:main"},
		{"MAIN", "_default:main"},
		{"nav:top", "nav:top"},
		{"NAV:Top", "nav:top"},
		{"_default:main", "_default:main"},
		{"", "_default:"},
	}
	for _, c := range cases {
		if got := dumbName(c.in); got != c.want {
			t.Errorf("dumbName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestArticleTitlesKeysByCallerRef(t *testing.T) {
	d := newTestDB(t)

	got, err := d.ArticleTitles(context.Background(), []string{"main", "NAV:Top", "no-such-page"})
	if err != nil {
		t.Fatalf("ArticleTitles() err = %v, want nil", err)
	}
	if len(got) != 2 {
		t.Errorf("len(ArticleTitles()) = %d, want 2", len(got))
	}
	if got["main"] != "main" {
		t.Errorf("ArticleTitles()[\"main\"] = %q, want %q", got["main"], "main")
	}
	if got["NAV:Top"] != "top" {
		t.Errorf("ArticleTitles()[\"NAV:Top\"] = %q, want %q", got["NAV:Top"], "top")
	}
	if _, ok := got["no-such-page"]; ok {
		t.Error("ArticleTitles() contains \"no-such-page\", want it absent")
	}
}

func TestArticleTitlesEmptyRefs(t *testing.T) {
	d := newTestDB(t)

	got, err := d.ArticleTitles(context.Background(), nil)
	if err != nil {
		t.Fatalf("ArticleTitles(nil) err = %v, want nil", err)
	}
	if len(got) != 0 {
		t.Errorf("len(ArticleTitles(nil)) = %d, want 0", len(got))
	}
}

func TestArticleSourcesReturnsSource(t *testing.T) {
	d := newTestDB(t)

	got, err := d.ArticleSources(context.Background(), []string{"main", "nav:top"})
	if err != nil {
		t.Fatalf("ArticleSources() err = %v, want nil", err)
	}
	if _, ok := got["main"]; !ok {
		t.Error("ArticleSources() is missing \"main\", want it present")
	}
	if got["main"] == "" {
		t.Error("ArticleSources()[\"main\"] = \"\", want the page source")
	}
}

func TestSourceAtRevisionMatchesLatestOnTheOnlyRevision(t *testing.T) {
	ctx := context.Background()
	d := newTestDB(t)

	article, err := d.ArticleByName(ctx, "probeoff:unratable")
	if err != nil {
		t.Fatalf("ArticleByName(probeoff:unratable) err = %v, want nil", err)
	}
	latest, err := d.LatestSource(ctx, article.ID)
	if err != nil {
		t.Fatalf("LatestSource() err = %v, want nil", err)
	}
	got, err := d.SourceAtRevision(ctx, article.ID, 0)
	if err != nil {
		t.Fatalf("SourceAtRevision(0) err = %v, want nil", err)
	}
	if got != latest {
		t.Errorf("SourceAtRevision(0) = %q, want %q", got, latest)
	}
}

func TestSourceAtRevisionOfRevisionThatIsNotThere(t *testing.T) {
	ctx := context.Background()
	d := newTestDB(t)

	article, err := d.ArticleByName(ctx, "probeoff:unratable")
	if err != nil {
		t.Fatalf("ArticleByName(probeoff:unratable) err = %v, want nil", err)
	}
	if _, err := d.SourceAtRevision(ctx, article.ID, 99); !errors.Is(err, ErrNotFound) {
		t.Errorf("SourceAtRevision(99) err = %v, want ErrNotFound", err)
	}
}
