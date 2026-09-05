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

type CloudTag struct {
	Name         string
	FullName     string
	Articles     int
	CategoryID   int64
	CategoryName string
	CategorySlug string
	CategoryText string
	Priority     *int
}

var qTagCloud = register("TagCloud", `
SELECT t.name,
       CASE WHEN c.slug = '_default' THEN t.name ELSE c.slug || ':' || t.name END,
       count(link.article_id),
       c.id, c.name, c.slug, c.description, c.priority
FROM web_tag t
JOIN web_tagscategory c ON c.id = t.category_id
LEFT JOIN web_article_tags link ON link.tag_id = t.id
WHERE t.name NOT LIKE '\_%'
GROUP BY t.id, t.name, c.id, c.name, c.slug, c.description, c.priority
ORDER BY count(link.article_id) DESC`)

// The limit lands after the busiest tags come first, so it decides which tags
// are in the cloud and not just how many.
func (d *DB) TagCloud(ctx context.Context, limit *int) ([]CloudTag, error) {
	sql := qTagCloud
	args := []any{}
	if limit != nil {
		sql += "\nLIMIT $1"
		args = append(args, *limit)
	}
	rows, err := d.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query tag cloud: %w", err)
	}
	defer rows.Close()

	var out []CloudTag
	for rows.Next() {
		var t CloudTag
		if err := rows.Scan(&t.Name, &t.FullName, &t.Articles,
			&t.CategoryID, &t.CategoryName, &t.CategorySlug, &t.CategoryText, &t.Priority); err != nil {
			return nil, fmt.Errorf("scan tag cloud: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read tag cloud: %w", err)
	}
	return out, nil
}

type TagsCategory struct {
	ID          int64
	Name        string
	Description string
	Slug        string
}

var qTagsCategories = register("TagsCategories", `
SELECT id, name, description, slug
FROM web_tagscategory
ORDER BY priority, id`)

func (d *DB) TagsCategories(ctx context.Context) ([]TagsCategory, error) {
	rows, err := d.pool.Query(ctx, qTagsCategories)
	if err != nil {
		return nil, fmt.Errorf("list tag categories: %w", err)
	}
	defer rows.Close()

	var out []TagsCategory
	for rows.Next() {
		var c TagsCategory
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.Slug); err != nil {
			return nil, fmt.Errorf("scan tag category: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list tag categories: %w", err)
	}
	return out, nil
}

type NamedTag struct {
	CategoryID int64
	Name       string
}

var qAllTags = register("AllTags", `SELECT category_id, name FROM web_tag ORDER BY id`)

func (d *DB) AllTags(ctx context.Context) ([]NamedTag, error) {
	rows, err := d.pool.Query(ctx, qAllTags)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()

	var out []NamedTag
	for rows.Next() {
		var t NamedTag
		if err := rows.Scan(&t.CategoryID, &t.Name); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	return out, nil
}
