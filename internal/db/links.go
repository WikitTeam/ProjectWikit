package db

import (
	"context"
	"fmt"
)

const (
	LinkInclude = "include"
	LinkPlain   = "link"
)

type ExternalLink struct {
	From string
	Type string
}

var qLinksTo = register("LinksTo", `
SELECT link_from, link_type
FROM web_externallink
WHERE link_to = $1`)

func (d *DB) LinksTo(ctx context.Context, fullName string) ([]ExternalLink, error) {
	rows, err := d.pool.Query(ctx, qLinksTo, fullName)
	if err != nil {
		return nil, fmt.Errorf("query links to %q: %w", fullName, err)
	}
	defer rows.Close()

	var out []ExternalLink
	for rows.Next() {
		var link ExternalLink
		if err := rows.Scan(&link.From, &link.Type); err != nil {
			return nil, fmt.Errorf("scan link: %w", err)
		}
		out = append(out, link)
	}
	return out, rows.Err()
}

var qArticleChildren = register("ArticleChildren", `
SELECT `+articleColumns+`
FROM web_article
WHERE parent_id = $1
ORDER BY id`)

func (d *DB) ArticleChildren(ctx context.Context, articleID int64) ([]Article, error) {
	rows, err := d.pool.Query(ctx, qArticleChildren, articleID)
	if err != nil {
		return nil, fmt.Errorf("query children of %d: %w", articleID, err)
	}
	defer rows.Close()

	var out []Article
	for rows.Next() {
		var a Article
		if err := rows.Scan(&a.ID, &a.Category, &a.Name, &a.Title, &a.ParentID, &a.Locked,
			&a.CreatedAt, &a.UpdatedAt, &a.MediaName); err != nil {
			return nil, fmt.Errorf("scan child: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
