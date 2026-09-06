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
	AND ($2 = '' OR a.category = $2)`

var qAdminPages = register("AdminPages", `
SELECT a.id, a.category, a.name, coalesce(a.title, ''), a.updated_at,
	(SELECT count(*) FROM web_articlelogentry l WHERE l.article_id = a.id)
FROM web_article a`+adminPageWhere+`
ORDER BY a.updated_at DESC, a.id DESC
LIMIT $3 OFFSET $4`)

var qAdminPageCount = register("AdminPageCount", `
SELECT count(*) FROM web_article a`+adminPageWhere)

func (d *DB) AdminPages(ctx context.Context, query, category string, limit, offset int) ([]AdminPageRow, int, error) {
	var total int
	if err := d.pool.QueryRow(ctx, qAdminPageCount, query, category).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count pages: %w", err)
	}
	rows, err := d.pool.Query(ctx, qAdminPages, query, category, limit, offset)
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
SELECT category, count(*) FROM web_article GROUP BY category ORDER BY category`)

type PageCategoryCount struct {
	Category string
	Pages    int
}

func (d *DB) AdminPageCategories(ctx context.Context) ([]PageCategoryCount, error) {
	rows, err := d.pool.Query(ctx, qAdminPageCategories)
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
