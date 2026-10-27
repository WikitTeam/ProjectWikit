package db

import (
	"context"
	"fmt"
	"time"
)

var qSitemapArticles = register("SitemapArticles", `
SELECT a.category, a.name, a.updated_at
FROM web_article a
WHERE a.site_id = $1
  AND NOT (a.category = ANY($2))
  AND a.is_indexed
  AND NOT EXISTS (SELECT 1 FROM web_category c
      WHERE c.site_id = a.site_id AND c.name = a.category AND NOT c.is_indexed)
  AND NOT EXISTS (SELECT 1 FROM web_article_tags ax
      JOIN web_tag t ON t.id = ax.tag_id
      WHERE ax.article_id = a.id AND NOT t.is_indexed)
ORDER BY a.id`)

func (d *DB) SitemapArticles(ctx context.Context, siteID int64, hidden []string) ([]Article, error) {
	if hidden == nil {
		hidden = []string{}
	}
	rows, err := d.pool.Query(ctx, qSitemapArticles, siteID, hidden)
	if err != nil {
		return nil, fmt.Errorf("query sitemap articles: %w", err)
	}
	defer rows.Close()

	var out []Article
	for rows.Next() {
		var a Article
		if err := rows.Scan(&a.Category, &a.Name, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan sitemap article: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

type SitemapThread struct {
	ID        int64
	Name      string
	UpdatedAt time.Time
}

var qSitemapThreads = register("SitemapThreads", `
SELECT t.id, t.name, t.updated_at
FROM web_forumthread t
JOIN web_forumcategory c ON c.id = t.category_id
WHERE t.site_id = $1 AND c.section_id = ANY($2)
ORDER BY t.id`)

func (d *DB) SitemapThreads(ctx context.Context, siteID int64, sectionIDs []int64) ([]SitemapThread, error) {
	if len(sectionIDs) == 0 {
		return nil, nil
	}
	rows, err := d.pool.Query(ctx, qSitemapThreads, siteID, sectionIDs)
	if err != nil {
		return nil, fmt.Errorf("query sitemap threads: %w", err)
	}
	defer rows.Close()

	var out []SitemapThread
	for rows.Next() {
		var t SitemapThread
		if err := rows.Scan(&t.ID, &t.Name, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan sitemap thread: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
