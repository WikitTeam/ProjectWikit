package db

import (
	"context"
	"fmt"
	"slices"
	"time"
)

type ActivityDay struct {
	Day   time.Time
	Count int
}

// Hidden lists site:category pairs the reader may not see. Edits and votes on
// pages there are left out. Counts come back in quarter hours rather than days,
// because a bundled PostgreSQL may not know the site's zone by name. Every zone's
// offset is a whole number of quarter hours, so no bucket straddles midnight.
var qActivityDays = register("ActivityDays", `
SELECT date_bin('15 minutes', x.at, TIMESTAMPTZ '2000-01-01 00:00:00+00'), count(*)
FROM (
	SELECT l.created_at AS at, a.site_id, a.category::text AS category
	FROM web_articlelogentry l JOIN web_article a ON a.id = l.article_id
	WHERE l.user_id = $1 AND l.created_at >= $2
	UNION ALL
	SELECT e.at, a.site_id, a.category::text
	FROM pwikit_vote_event e JOIN web_article a ON a.id = e.article_id
	WHERE e.user_id = $1 AND e.at >= $2 AND e.new_rate IS NOT NULL
	UNION ALL
	SELECT p.created_at, t.site_id, NULL
	FROM web_forumpost p JOIN web_forumthread t ON t.id = p.thread_id
	WHERE p.author_id = $1 AND p.created_at >= $2
) x
WHERE ($3::bigint IS NULL OR x.site_id = $3)
  AND (x.category IS NULL OR NOT (x.site_id::text || ':' || x.category = ANY(coalesce($4::text[], '{}'))))
GROUP BY 1
ORDER BY 1`)

func (d *DB) ActivityDays(ctx context.Context, userID int64, siteID *int64, since time.Time, zone *time.Location, hidden []string) ([]ActivityDay, error) {
	rows, err := d.pool.Query(ctx, qActivityDays, userID, since, siteID, hidden)
	if err != nil {
		return nil, fmt.Errorf("read activity of %d: %w", userID, err)
	}
	defer rows.Close()
	counts := map[time.Time]int{}
	for rows.Next() {
		var at time.Time
		var n int
		if err := rows.Scan(&at, &n); err != nil {
			return nil, err
		}
		y, m, day := at.In(zone).Date()
		counts[time.Date(y, m, day, 0, 0, 0, 0, time.UTC)] += n
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]ActivityDay, 0, len(counts))
	for day, n := range counts {
		out = append(out, ActivityDay{Day: day, Count: n})
	}
	slices.SortFunc(out, func(a, b ActivityDay) int { return a.Day.Compare(b.Day) })
	return out, nil
}

type AuthorStats struct {
	Pages int
	Score float64
	Edits int
	Posts int
}

const visibleArticle = ` AND ($2::bigint IS NULL OR a.site_id = $2)
     AND NOT (a.site_id::text || ':' || a.category::text = ANY(coalesce($3::text[], '{}')))`

var qAuthorStats = register("AuthorStats", `
SELECT
  (SELECT count(*) FROM web_article_authors aa JOIN web_article a ON a.id = aa.article_id
   WHERE aa.user_id = $1`+visibleArticle+`),
  (SELECT coalesce(sum(v.rate), 0) FROM web_article_authors aa JOIN web_article a ON a.id = aa.article_id
   JOIN web_vote v ON v.article_id = a.id
   WHERE aa.user_id = $1`+visibleArticle+`),
  (SELECT count(*) FROM web_articlelogentry l JOIN web_article a ON a.id = l.article_id
   WHERE l.user_id = $1`+visibleArticle+`),
  (SELECT count(*) FROM web_forumpost p JOIN web_forumthread t ON t.id = p.thread_id
   WHERE p.author_id = $1 AND ($2::bigint IS NULL OR t.site_id = $2))`)

func (d *DB) AuthorStats(ctx context.Context, userID int64, siteID *int64, hidden []string) (AuthorStats, error) {
	var s AuthorStats
	if err := d.pool.QueryRow(ctx, qAuthorStats, userID, siteID, hidden).Scan(&s.Pages, &s.Score, &s.Edits, &s.Posts); err != nil {
		return s, fmt.Errorf("read stats of %d: %w", userID, err)
	}
	return s, nil
}

var (
	qAuthorTrendVotes = register("AuthorTrendVotes", `
SELECT v.rate, coalesce(v.date, a.created_at),
       EXISTS (SELECT 1 FROM pwikit_vote_event e WHERE e.article_id = v.article_id AND e.user_id = v.user_id)
FROM web_article_authors aa
JOIN web_article a ON a.id = aa.article_id
JOIN web_vote v ON v.article_id = a.id
WHERE aa.user_id = $1`+visibleArticle)

	qAuthorTrendEvents = register("AuthorTrendEvents", `
SELECT e.old_rate, e.new_rate, e.at
FROM web_article_authors aa
JOIN web_article a ON a.id = aa.article_id
JOIN pwikit_vote_event e ON e.article_id = a.id
WHERE aa.user_id = $1`+visibleArticle)

	qAuthorFirstPage = register("AuthorFirstPage", `
SELECT min(a.created_at)
FROM web_article_authors aa JOIN web_article a ON a.id = aa.article_id
WHERE aa.user_id = $1`+visibleArticle)
)

func (d *DB) AuthorScoreTrend(ctx context.Context, userID int64, siteID *int64, hidden []string) ([]TrendPoint, error) {
	var first *time.Time
	if err := d.pool.QueryRow(ctx, qAuthorFirstPage, userID, siteID, hidden).Scan(&first); err != nil {
		return nil, fmt.Errorf("read first page of %d: %w", userID, err)
	}
	if first == nil {
		return nil, nil
	}
	type step struct {
		at  time.Time
		sum float64
	}
	var steps []step
	var total, placed float64
	rows, err := d.pool.Query(ctx, qAuthorTrendVotes, userID, siteID, hidden)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var rate float64
		var at time.Time
		var changed bool
		if err := rows.Scan(&rate, &at, &changed); err != nil {
			rows.Close()
			return nil, err
		}
		total += rate
		if !changed {
			steps = append(steps, step{at, rate})
			placed += rate
		}
	}
	rows.Close()
	rows, err = d.pool.Query(ctx, qAuthorTrendEvents, userID, siteID, hidden)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var before, after *float64
		var at time.Time
		if err := rows.Scan(&before, &after, &at); err != nil {
			rows.Close()
			return nil, err
		}
		var delta float64
		if before != nil {
			delta -= *before
		}
		if after != nil {
			delta += *after
		}
		steps = append(steps, step{at, delta})
		placed += delta
	}
	rows.Close()
	if rest := total - placed; rest != 0 {
		steps = append(steps, step{*first, rest})
	}
	slices.SortStableFunc(steps, func(a, b step) int { return a.at.Compare(b.at) })
	out := []TrendPoint{{At: *first}}
	var run float64
	for _, s := range steps {
		run += s.sum
		if n := len(out); out[n-1].At.Equal(s.at) {
			out[n-1].Value = run
			continue
		}
		out = append(out, TrendPoint{At: s.at, Value: run})
	}
	return out, nil
}

type UserVote struct {
	SiteID   int64
	Category string
	Name     string
	Title    string
	Rate     float64
	At       time.Time
}

var qUserVotes = register("UserVotes", `
SELECT a.site_id, a.category, a.name, a.title, e.new_rate, e.at
FROM pwikit_vote_event e JOIN web_article a ON a.id = e.article_id
WHERE e.user_id = $1 AND e.new_rate IS NOT NULL AND e.at >= $2 AND e.at < $3
  AND ($4::bigint IS NULL OR a.site_id = $4)
  AND NOT (a.site_id::text || ':' || a.category::text = ANY(coalesce($5::text[], '{}')))
ORDER BY e.at DESC
LIMIT $6`)

func (d *DB) UserVotes(ctx context.Context, userID int64, siteID *int64, from, before time.Time, hidden []string, limit int) ([]UserVote, error) {
	rows, err := d.pool.Query(ctx, qUserVotes, userID, from, before, siteID, hidden, limit)
	if err != nil {
		return nil, fmt.Errorf("read votes of %d: %w", userID, err)
	}
	defer rows.Close()
	var out []UserVote
	for rows.Next() {
		var v UserVote
		if err := rows.Scan(&v.SiteID, &v.Category, &v.Name, &v.Title, &v.Rate, &v.At); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
