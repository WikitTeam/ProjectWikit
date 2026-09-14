package modules

import (
	"context"

	"github.com/WikitTeam/ProjectWikit/internal/db"
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
