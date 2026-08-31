package db

import (
	"context"
	"fmt"
	"strings"
)

// complete_full_name is a generated column that always carries an explicit
// category, so a bare page name grows the implicit _default one before it can
// match anything.
func dumbName(ref string) string {
	if strings.Contains(ref, ":") {
		return strings.ToLower(ref)
	}
	return "_default:" + strings.ToLower(ref)
}

func dumbNames(refs []string) []string {
	out := make([]string, len(refs))
	for i, ref := range refs {
		out[i] = dumbName(ref)
	}
	return out
}

var qArticleTitles = register("ArticleTitles", `
SELECT complete_full_name, title
FROM web_article
WHERE complete_full_name = ANY($1)`)

// ArticleTitles keys its result by the caller's own ref strings. Refs with no
// article are absent from the map rather than present-and-empty: fetch_internal_links
// drops missing pages instead of reporting them as non-existent.
func (d *DB) ArticleTitles(ctx context.Context, refs []string) (map[string]string, error) {
	if len(refs) == 0 {
		return map[string]string{}, nil
	}

	rows, err := d.pool.Query(ctx, qArticleTitles, dumbNames(refs))
	if err != nil {
		return nil, fmt.Errorf("query article titles: %w", err)
	}
	defer rows.Close()

	byDumb := make(map[string]string)
	for rows.Next() {
		var fullName, title string
		if err := rows.Scan(&fullName, &title); err != nil {
			return nil, fmt.Errorf("scan article title: %w", err)
		}
		byDumb[strings.ToLower(fullName)] = title
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read article titles: %w", err)
	}

	out := make(map[string]string, len(refs))
	for _, ref := range refs {
		if title, ok := byDumb[dumbName(ref)]; ok {
			out[ref] = title
		}
	}
	return out, nil
}

var qArticleSources = register("ArticleSources", `
SELECT DISTINCT ON (a.id) a.complete_full_name, v.source
FROM web_articleversion v
JOIN web_article a ON a.id = v.article_id
WHERE a.complete_full_name = ANY($1)
ORDER BY a.id, v.created_at DESC`)

// ArticleSources returns the newest version's source per article, keyed by the
// caller's own ref strings.
func (d *DB) ArticleSources(ctx context.Context, refs []string) (map[string]string, error) {
	if len(refs) == 0 {
		return map[string]string{}, nil
	}

	rows, err := d.pool.Query(ctx, qArticleSources, dumbNames(refs))
	if err != nil {
		return nil, fmt.Errorf("query article sources: %w", err)
	}
	defer rows.Close()

	byDumb := make(map[string]string)
	for rows.Next() {
		var fullName, source string
		if err := rows.Scan(&fullName, &source); err != nil {
			return nil, fmt.Errorf("scan article source: %w", err)
		}
		byDumb[strings.ToLower(fullName)] = source
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read article sources: %w", err)
	}

	out := make(map[string]string, len(refs))
	for _, ref := range refs {
		if source, ok := byDumb[dumbName(ref)]; ok {
			out[ref] = source
		}
	}
	return out, nil
}
