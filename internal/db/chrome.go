package db

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
)

var qBreadcrumbs = register("Breadcrumbs", `
WITH RECURSIVE chain AS (
    SELECT id, parent_id, ARRAY[id] AS seen, 0 AS depth
    FROM web_article
    WHERE id = $1
  UNION ALL
    SELECT p.id, p.parent_id, chain.seen || p.id, chain.depth + 1
    FROM web_article p
    JOIN chain ON p.id = chain.parent_id
    WHERE NOT p.id = ANY(chain.seen)
)
SELECT `+prefixedArticleColumns+`
FROM chain
JOIN web_article a ON a.id = chain.id
ORDER BY chain.depth DESC`)

// Breadcrumbs walks up the parent chain and returns the root first. The seen
// array is what keeps a page that is its own ancestor from looping forever.
func (d *DB) Breadcrumbs(ctx context.Context, articleID int64) ([]Article, error) {
	rows, err := d.pool.Query(ctx, qBreadcrumbs, articleID)
	if err != nil {
		return nil, fmt.Errorf("query breadcrumbs of article %d: %w", articleID, err)
	}
	defer rows.Close()

	var out []Article
	for rows.Next() {
		var a Article
		if err := rows.Scan(&a.ID, &a.Category, &a.Name, &a.Title, &a.ParentID,
			&a.Locked, &a.CreatedAt, &a.UpdatedAt, &a.MediaName); err != nil {
			return nil, fmt.Errorf("scan breadcrumb of article %d: %w", articleID, err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read breadcrumbs of article %d: %w", articleID, err)
	}
	return out, nil
}

type Tag struct {
	Name     string
	FullName string
}

type TagCategory struct {
	ID       int64
	Name     string
	Priority *int
	Tags     []Tag
}

var qArticleTagCategories = register("ArticleTagCategories", `
SELECT c.id, c.name, c.priority, t.name,
       CASE WHEN c.slug = '_default' THEN t.name ELSE c.slug || ':' || t.name END
FROM web_article_tags link
JOIN web_tag t ON t.id = link.tag_id
JOIN web_tagscategory c ON c.id = t.category_id
WHERE link.article_id = $1 AND t.name NOT LIKE '\_%'
ORDER BY c.id, link.id`)

// A category with no priority sorts under the page's tag count, which is the
// number it is compared with.
func (d *DB) ArticleTagCategories(ctx context.Context, articleID int64) ([]TagCategory, error) {
	rows, err := d.pool.Query(ctx, qArticleTagCategories, articleID)
	if err != nil {
		return nil, fmt.Errorf("query tag categories of article %d: %w", articleID, err)
	}
	defer rows.Close()

	var out []TagCategory
	byID := make(map[int64]int)
	total := 0
	for rows.Next() {
		var cat TagCategory
		var tag Tag
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Priority, &tag.Name, &tag.FullName); err != nil {
			return nil, fmt.Errorf("scan tag category of article %d: %w", articleID, err)
		}
		at, ok := byID[cat.ID]
		if !ok {
			at = len(out)
			byID[cat.ID] = at
			out = append(out, cat)
		}
		out[at].Tags = append(out[at].Tags, tag)
		total++
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read tag categories of article %d: %w", articleID, err)
	}

	priority := func(c TagCategory) int {
		if c.Priority != nil {
			return *c.Priority
		}
		return total
	}
	sort.SliceStable(out, func(i, j int) bool { return priority(out[i]) < priority(out[j]) })
	return out, nil
}

var qLatestRevNumber = register("LatestRevNumber", `
SELECT COALESCE(MAX(rev_number), 0)
FROM web_articlelogentry
WHERE article_id = $1`)

func (d *DB) LatestRevNumber(ctx context.Context, articleID int64) (int, error) {
	var n int
	if err := d.pool.QueryRow(ctx, qLatestRevNumber, articleID).Scan(&n); err != nil {
		return 0, fmt.Errorf("query latest revision number of article %d: %w", articleID, err)
	}
	return n, nil
}

var qCategoryExists = register("CategoryExists", `
SELECT EXISTS(SELECT 1 FROM web_category WHERE name = $1)`)

// The page skips the permission check entirely when neither it nor its category
// has a row, which is how a 404 wins over a 403 there.
func (d *DB) CategoryExists(ctx context.Context, name string) (bool, error) {
	var exists bool
	if err := d.pool.QueryRow(ctx, qCategoryExists, name).Scan(&exists); err != nil {
		return false, fmt.Errorf("check category %q: %w", name, err)
	}
	return exists, nil
}

var qCategoryIndexed = register("CategoryIndexed", `
SELECT is_indexed
FROM web_category
WHERE name = $1`)

func (d *DB) CategoryIndexed(ctx context.Context, name string) (bool, error) {
	var indexed bool
	err := d.pool.QueryRow(ctx, qCategoryIndexed, name).Scan(&indexed)
	if errors.Is(err, pgx.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("query category %q: %w", name, err)
	}
	return indexed, nil
}

const (
	ThemeInline   = "inline"
	ThemeExternal = "external"
)

type Theme struct {
	Slug        string
	Mode        string
	ExternalURL string
	UpdatedAt   time.Time
}

var qThemeByID = register("ThemeByID", `
SELECT slug, mode, external_url, updated_at
FROM web_theme
WHERE id = $1`)

func (d *DB) ThemeByID(ctx context.Context, id int64) (*Theme, error) {
	var t Theme
	err := d.pool.QueryRow(ctx, qThemeByID, id).Scan(&t.Slug, &t.Mode, &t.ExternalURL, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lookup theme %d: %w", id, err)
	}
	return &t, nil
}

var qArticleTagNames = register("ArticleTagNames", `
SELECT CASE WHEN c.slug = '_default' THEN t.name ELSE c.slug || ':' || t.name END,
       CASE WHEN c.slug = '_default' THEN '' ELSE t.name END
FROM web_article_tags link
JOIN web_tag t ON t.id = link.tag_id
JOIN web_tagscategory c ON c.id = t.category_id
WHERE link.article_id = $1
ORDER BY link.id`)

// ArticleTagNames is the list ftml is told about, where a tag outside the
// default category appears twice, prefixed and bare. Hidden tags stay in.
func (d *DB) ArticleTagNames(ctx context.Context, articleID int64) ([]string, error) {
	rows, err := d.pool.Query(ctx, qArticleTagNames, articleID)
	if err != nil {
		return nil, fmt.Errorf("query tag names of article %d: %w", articleID, err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var full, bare string
		if err := rows.Scan(&full, &bare); err != nil {
			return nil, fmt.Errorf("scan tag name of article %d: %w", articleID, err)
		}
		out = append(out, full)
		if bare != "" {
			out = append(out, bare)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read tag names of article %d: %w", articleID, err)
	}
	return out, nil
}
