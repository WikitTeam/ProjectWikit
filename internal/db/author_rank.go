package db

import (
	"context"
	"fmt"
)

type AuthorRankRow struct {
	UserID int64
	Score  float64
	Pages  int
	Place  int
	Total  int
}

type AuthorRankFilter struct {
	Categories     []string
	SkipCategories []string
	Tags           []string
	Negate         []string
	Exclude        []string
	Hidden         []string
}

// A page's score is the sum of its votes. Pages carrying one of the negated
// tags count against it the other way round, and pages carrying an excluded tag
// do not count at all. Neither changes the rating the page itself shows.
var qAuthorRank = register("AuthorRank", `
WITH tagged AS (
	SELECT at.article_id,
	       CASE WHEN c.slug = '_default' THEN t.name::text ELSE c.slug || ':' || t.name END AS tag
	FROM web_article_tags at
	JOIN web_tag t ON t.id = at.tag_id
	JOIN web_tagscategory c ON c.id = t.category_id
),
pages AS (
	SELECT a.id,
	       coalesce((SELECT sum(v.rate) FROM web_vote v WHERE v.article_id = a.id), 0) AS score,
	       EXISTS (SELECT 1 FROM tagged g WHERE g.article_id = a.id AND g.tag = ANY($3::text[])) AS negated,
	       EXISTS (SELECT 1 FROM tagged g WHERE g.article_id = a.id AND g.tag = ANY($4::text[])) AS excluded
	FROM web_article a
	WHERE a.site_id = $1
	  AND (cardinality($2::text[]) = 0 OR a.category::text = ANY($2::text[]))
	  AND NOT (a.category::text = ANY($5::text[]))
	  AND NOT (a.category::text = ANY($8::text[]))
	  AND (cardinality($9::text[]) = 0
	       OR EXISTS (SELECT 1 FROM tagged g WHERE g.article_id = a.id AND g.tag = ANY($9::text[])))
),
ranked AS (
	SELECT aa.user_id,
	       sum(CASE WHEN p.negated THEN -p.score ELSE p.score END) AS score,
	       count(*) AS pages
	FROM pages p
	JOIN web_article_authors aa ON aa.article_id = p.id
	WHERE NOT p.excluded
	GROUP BY aa.user_id
)
SELECT user_id, score, pages, rank() OVER (ORDER BY score DESC), count(*) OVER ()
FROM ranked
ORDER BY score DESC, pages DESC, user_id
OFFSET $6 LIMIT $7`)

func (d *DB) AuthorRank(ctx context.Context, siteID int64, f AuthorRankFilter, offset, limit int) ([]AuthorRankRow, error) {
	nonNil := func(s []string) []string {
		if s == nil {
			return []string{}
		}
		return s
	}
	rows, err := d.pool.Query(ctx, qAuthorRank, siteID, nonNil(f.Categories), nonNil(f.Negate), nonNil(f.Exclude),
		nonNil(f.Hidden), offset, limit, nonNil(f.SkipCategories), nonNil(f.Tags))
	if err != nil {
		return nil, fmt.Errorf("rank authors of site %d: %w", siteID, err)
	}
	defer rows.Close()
	var out []AuthorRankRow
	for rows.Next() {
		var r AuthorRankRow
		if err := rows.Scan(&r.UserID, &r.Score, &r.Pages, &r.Place, &r.Total); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
