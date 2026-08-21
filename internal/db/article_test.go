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

func TestSiteByHostsMatchesDomainAndMediaDomain(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()

	byDomain, err := d.SiteByHosts(ctx, []string{"localhost"})
	if err != nil {
		t.Fatalf("SiteByHosts([localhost]) err = %v, want nil", err)
	}
	if byDomain.Slug != "wikit" {
		t.Errorf("SiteByHosts([localhost]).Slug = %q, want %q", byDomain.Slug, "wikit")
	}

	byMedia, err := d.SiteByHosts(ctx, []string{"media.localhost"})
	if err != nil {
		t.Fatalf("SiteByHosts([media.localhost]) err = %v, want nil", err)
	}
	if byMedia.ID != byDomain.ID {
		t.Errorf("SiteByHosts([media.localhost]).ID = %d, want %d", byMedia.ID, byDomain.ID)
	}
}

func TestSiteByHostsTriesHostsInOrder(t *testing.T) {
	d := newTestDB(t)

	got, err := d.SiteByHosts(context.Background(), []string{"localhost:8000", "localhost"})
	if err != nil {
		t.Fatalf("SiteByHosts() err = %v, want nil", err)
	}
	if got.Domain != "localhost" {
		t.Errorf("SiteByHosts().Domain = %q, want %q", got.Domain, "localhost")
	}
}

func TestSiteByHostsUnknownHost(t *testing.T) {
	d := newTestDB(t)

	_, err := d.SiteByHosts(context.Background(), []string{"nope.example"})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("SiteByHosts([nope.example]) err = %v, want ErrNotFound", err)
	}
}

func TestAnySite(t *testing.T) {
	d := newTestDB(t)

	got, err := d.AnySite(context.Background())
	if err != nil {
		t.Fatalf("AnySite() err = %v, want nil", err)
	}
	if !got {
		t.Error("AnySite() = false, want true")
	}
}

func TestUserByName(t *testing.T) {
	d := newTestDB(t)

	got, err := d.UserByName(context.Background(), "seeduser")
	if err != nil {
		t.Fatalf("UserByName(\"seeduser\") err = %v, want nil", err)
	}
	if got.Username != "seeduser" {
		t.Errorf("UserByName().Username = %q, want %q", got.Username, "seeduser")
	}
	if got.Type != UserTypeNormal {
		t.Errorf("UserByName().Type = %q, want %q", got.Type, UserTypeNormal)
	}
}

func TestUserByNameUnknown(t *testing.T) {
	d := newTestDB(t)

	_, err := d.UserByName(context.Background(), "no-such-user")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("UserByName(\"no-such-user\") err = %v, want ErrNotFound", err)
	}
}
