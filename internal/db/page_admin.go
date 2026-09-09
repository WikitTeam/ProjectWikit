package db

import (
	"context"
	"fmt"
	"time"
)

type AdminPageRow struct {
	ID        int64
	Category  string
	Name      string
	Title     string
	Revisions int
	UpdatedAt time.Time
}

func (p *AdminPageRow) FullName() string {
	if p.Category != DefaultCategory {
		return p.Category + ":" + p.Name
	}
	return p.Name
}

const adminPageWhere = `
WHERE ($1 = '' OR a.name ILIKE '%' || $1 || '%' OR a.title ILIKE '%' || $1 || '%')
	AND ($2 = '' OR a.category = $2)
	AND a.site_id = $3`

var qAdminPages = register("AdminPages", `
SELECT a.id, a.category, a.name, coalesce(a.title, ''), a.updated_at,
	(SELECT count(*) FROM web_articlelogentry l WHERE l.article_id = a.id)
FROM web_article a`+adminPageWhere+`
ORDER BY a.updated_at DESC, a.id DESC
LIMIT $4 OFFSET $5`)

var qAdminPageCount = register("AdminPageCount", `
SELECT count(*) FROM web_article a`+adminPageWhere)

func (d *DB) AdminPages(ctx context.Context, siteID int64, query, category string, limit, offset int) ([]AdminPageRow, int, error) {
	var total int
	if err := d.pool.QueryRow(ctx, qAdminPageCount, query, category, siteID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count pages: %w", err)
	}
	rows, err := d.pool.Query(ctx, qAdminPages, query, category, siteID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list pages: %w", err)
	}
	defer rows.Close()

	var out []AdminPageRow
	for rows.Next() {
		var p AdminPageRow
		if err := rows.Scan(&p.ID, &p.Category, &p.Name, &p.Title, &p.UpdatedAt, &p.Revisions); err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}

var qAdminPageCategories = register("AdminPageCategories", `
SELECT category, count(*) FROM web_article WHERE site_id = $1 GROUP BY category ORDER BY category`)

type PageCategoryCount struct {
	Category string
	Pages    int
}

func (d *DB) AdminPageCategories(ctx context.Context, siteID int64) ([]PageCategoryCount, error) {
	rows, err := d.pool.Query(ctx, qAdminPageCategories, siteID)
	if err != nil {
		return nil, fmt.Errorf("list page categories: %w", err)
	}
	defer rows.Close()

	var out []PageCategoryCount
	for rows.Next() {
		var one PageCategoryCount
		if err := rows.Scan(&one.Category, &one.Pages); err != nil {
			return nil, err
		}
		out = append(out, one)
	}
	return out, rows.Err()
}

var qSetArticleIndexed = register("SetArticleIndexed", `
UPDATE web_article SET is_indexed = $3 WHERE id = $1 AND site_id = $2`)

func (d *DB) SetArticleIndexed(ctx context.Context, siteID, id int64, indexed bool) error {
	tag, err := d.pool.Exec(ctx, qSetArticleIndexed, id, siteID, indexed)
	if err != nil {
		return fmt.Errorf("set indexed on article %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
