package db

import (
	"context"
	"fmt"
)

type Suggestion struct {
	Value string
	Label string
}

var qSuggestTags = register("SuggestTags", `
SELECT t.name, c.slug
FROM web_tag t
JOIN web_tagscategory c ON c.id = t.category_id
WHERE t.site_id = $1 AND t.is_indexed
  AND (t.name ILIKE $2 OR (c.slug || ':' || t.name) ILIKE $2)
ORDER BY (t.name ILIKE $3) DESC, lower(t.name), c.slug
LIMIT $4`)

func (d *DB) SuggestTags(ctx context.Context, siteID int64, typed string, limit int) ([]Suggestion, error) {
	rows, err := d.pool.Query(ctx, qSuggestTags, siteID, likeContains(typed), likePrefix(typed), limit)
	if err != nil {
		return nil, fmt.Errorf("suggest tags: %w", err)
	}
	defer rows.Close()
	var out []Suggestion
	for rows.Next() {
		var tag ArticleTag
		if err := rows.Scan(&tag.Name, &tag.Category); err != nil {
			return nil, fmt.Errorf("scan tag suggestion: %w", err)
		}
		full := tag.FullName()
		out = append(out, Suggestion{Value: full, Label: full})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read tag suggestions: %w", err)
	}
	return out, nil
}

var qSuggestCategories = register("SuggestCategories", `
SELECT a.category
FROM web_article a
WHERE a.site_id = $1 AND a.is_indexed
  AND NOT (a.category = ANY($2))
  AND a.category ILIKE $3
  AND NOT EXISTS (SELECT 1 FROM web_category c
    WHERE c.site_id = a.site_id AND c.name = a.category AND NOT c.is_indexed)
GROUP BY a.category
ORDER BY (a.category ILIKE $4) DESC, a.category
LIMIT $5`)

func (d *DB) SuggestCategories(ctx context.Context, siteID int64, hidden []string, typed string, limit int) ([]Suggestion, error) {
	if hidden == nil {
		hidden = []string{}
	}
	rows, err := d.pool.Query(ctx, qSuggestCategories, siteID, hidden, likeContains(typed), likePrefix(typed), limit)
	if err != nil {
		return nil, fmt.Errorf("suggest categories: %w", err)
	}
	defer rows.Close()
	var out []Suggestion
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan category suggestion: %w", err)
		}
		out = append(out, Suggestion{Value: name, Label: name})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read category suggestions: %w", err)
	}
	return out, nil
}

var qSuggestAuthors = register("SuggestAuthors", `
SELECT u.id, u.type, u.username, u.wikidot_username, u.display_name
FROM web_user u
WHERE EXISTS (SELECT 1 FROM web_article_authors aa
    JOIN web_article a ON a.id = aa.article_id
    WHERE aa.user_id = u.id AND a.site_id = $1)
  AND (u.username ILIKE $2 OR u.wikidot_username ILIKE $2 OR u.display_name ILIKE $2)
ORDER BY (u.username ILIKE $3 OR u.wikidot_username ILIKE $3 OR u.display_name ILIKE $3) DESC,
  lower(coalesce(nullif(u.display_name, ''), nullif(u.wikidot_username, ''), u.username)), u.id
LIMIT $4`)

func (d *DB) SuggestAuthors(ctx context.Context, siteID int64, typed string, limit int) ([]Suggestion, error) {
	rows, err := d.pool.Query(ctx, qSuggestAuthors, siteID, likeContains(typed), likePrefix(typed), limit)
	if err != nil {
		return nil, fmt.Errorf("suggest authors: %w", err)
	}
	defer rows.Close()
	var out []Suggestion
	for rows.Next() {
		var u User
		var wikidot, display *string
		if err := rows.Scan(&u.ID, &u.Type, &u.Username, &wikidot, &display); err != nil {
			return nil, fmt.Errorf("scan author suggestion: %w", err)
		}
		u.WikidotUsername, u.DisplayName = deref(wikidot), deref(display)
		out = append(out, Suggestion{Value: u.URLName(), Label: u.DisplayLabel()})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read author suggestions: %w", err)
	}
	return out, nil
}
