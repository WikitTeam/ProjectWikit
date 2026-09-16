package webapi

import (
	"context"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/site"
)

type siteLinks struct {
	current int64
	byID    map[int64]db.Site
}

func loadSiteLinks(ctx context.Context, d *db.DB) (siteLinks, error) {
	all, err := d.Sites(ctx)
	if err != nil {
		return siteLinks{}, err
	}
	links := siteLinks{byID: make(map[int64]db.Site, len(all))}
	if current := site.FromContext(ctx); current != nil {
		links.current = current.ID
	}
	for _, one := range all {
		links.byID[one.ID] = one
	}
	return links, nil
}

func (l siteLinks) title(id int64) string {
	return l.byID[id].Title
}

// A row from the site being read keeps a relative link, which survives any host
// name that site answers to.
func (l siteLinks) href(id int64, path string) string {
	one, ok := l.byID[id]
	if !ok || id == l.current {
		return path
	}
	return "//" + one.Domain + path
}
