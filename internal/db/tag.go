package db

import (
	"context"
	"fmt"
)

var qTagIDsByName = register("TagIDsByName", `
SELECT t.id
FROM web_tag t
JOIN web_tagscategory c ON c.id = t.category_id
WHERE c.slug = $1 AND t.name = $2
ORDER BY t.id`)

var qTagIDsByBareName = register("TagIDsByBareName", `
SELECT id
FROM web_tag
WHERE name = $1
ORDER BY id`)

func (d *DB) TagIDsByName(ctx context.Context, categorySlug, name string) ([]int64, error) {
	sql, args := qTagIDsByBareName, []any{name}
	if categorySlug != "" {
		sql, args = qTagIDsByName, []any{categorySlug, name}
	}
	rows, err := d.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query tag %q: %w", name, err)
	}
	defer rows.Close()

	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan tag %q: %w", name, err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read tag %q: %w", name, err)
	}
	return out, nil
}

var qArticleTagIDs = register("ArticleTagIDs", `
SELECT tag_id
FROM web_article_tags
WHERE article_id = $1
ORDER BY tag_id`)

func (d *DB) ArticleTagIDs(ctx context.Context, articleID int64) ([]int64, error) {
	rows, err := d.pool.Query(ctx, qArticleTagIDs, articleID)
	if err != nil {
		return nil, fmt.Errorf("query tags of article %d: %w", articleID, err)
	}
	defer rows.Close()

	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan tag of article %d: %w", articleID, err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read tags of article %d: %w", articleID, err)
	}
	return out, nil
}
