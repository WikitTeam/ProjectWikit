package db

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type ForumSearchHit struct {
	PostID     int64
	PostName   string
	CreatedAt  time.Time
	AuthorID   *int64
	ThreadID   int64
	ThreadName string
	Where      string
	ArticleID  *int64
	Source     string
}

// Comments keeps to the threads under pages and wins over CategoryID.
type ForumSearchFilter struct {
	Words      []string
	AuthorID   *int64
	From, To   *time.Time
	CategoryID *int64
	Comments   bool

	// A thread of its own matches only inside these sections, even by CategoryID.
	Sections []int64
}

// Every word has to turn up somewhere in the post, its title or its thread's
// title. Each post is read at its latest version.
var qSearchForum = register("SearchForum", `
SELECT p.id, p.name, p.created_at, p.author_id, t.id, t.name, coalesce(c.name, a.title, ''), t.article_id, v.source
FROM web_forumpost p
JOIN web_forumthread t ON t.id = p.thread_id
LEFT JOIN web_forumcategory c ON c.id = t.category_id
LEFT JOIN web_forumsection s ON s.id = c.section_id
LEFT JOIN web_article a ON a.id = t.article_id
JOIN LATERAL (SELECT source FROM web_forumpostversion WHERE post_id = p.id ORDER BY created_at DESC, id DESC LIMIT 1) v ON true
WHERE t.site_id = $1
  AND NOT EXISTS (
    SELECT 1 FROM unnest($2::text[]) w
    WHERE NOT (p.name ILIKE '%' || w || '%' OR t.name ILIKE '%' || w || '%' OR v.source ILIKE '%' || w || '%'))
  AND ($5::bigint IS NULL OR p.author_id = $5)
  AND ($6::timestamptz IS NULL OR p.created_at >= $6)
  AND ($7::timestamptz IS NULL OR p.created_at <= $7)
  AND ($8::bigint IS NULL OR t.category_id = $8)
  AND (NOT $9 OR t.article_id IS NOT NULL)
  AND (t.article_id IS NOT NULL OR s.id = ANY($10))
ORDER BY p.created_at DESC, p.id DESC
OFFSET $3 LIMIT $4`)

var likeEscaper = strings.NewReplacer(`\`, `\`, `%`, `\%`, `_`, `\_`)

func (d *DB) SearchForum(ctx context.Context, siteID int64, f ForumSearchFilter, offset, limit int) ([]ForumSearchHit, error) {
	escaped := make([]string, len(f.Words))
	for i, w := range f.Words {
		escaped[i] = likeEscaper.Replace(w)
	}
	sections := f.Sections
	if sections == nil {
		sections = []int64{}
	}
	rows, err := d.pool.Query(ctx, qSearchForum, siteID, escaped, offset, limit,
		f.AuthorID, f.From, f.To, f.CategoryID, f.Comments, sections)
	if err != nil {
		return nil, fmt.Errorf("search the forum of site %d: %w", siteID, err)
	}
	defer rows.Close()
	var out []ForumSearchHit
	for rows.Next() {
		var h ForumSearchHit
		if err := rows.Scan(&h.PostID, &h.PostName, &h.CreatedAt, &h.AuthorID, &h.ThreadID, &h.ThreadName, &h.Where, &h.ArticleID, &h.Source); err != nil {
			return nil, fmt.Errorf("scan a forum hit: %w", err)
		}
		out = append(out, h)
	}
	return out, rows.Err()
}
