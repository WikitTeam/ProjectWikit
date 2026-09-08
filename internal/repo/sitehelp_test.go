package repo

import (
	"context"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/site"
)

func onlySiteID(ctx context.Context, d *db.DB) int64 {
	slugs, err := d.SiteSlugs(ctx)
	if err != nil {
		panic(err)
	}
	if len(slugs) == 0 {
		panic("the test database holds no site")
	}
	found, err := d.SiteBySlug(ctx, slugs[0])
	if err != nil {
		panic(err)
	}
	return found.ID
}

func onSite(t *testing.T, ctx context.Context, d *db.DB) (context.Context, *db.Site) {
	t.Helper()
	slugs, err := d.SiteSlugs(ctx)
	if err != nil {
		t.Fatalf("SiteSlugs() err = %v, want nil", err)
	}
	if len(slugs) == 0 {
		t.Skip("the test database holds no site")
	}
	found, err := d.SiteBySlug(ctx, slugs[0])
	if err != nil {
		t.Fatalf("SiteBySlug(%q) err = %v, want nil", slugs[0], err)
	}
	return site.WithSite(ctx, found), found
}
