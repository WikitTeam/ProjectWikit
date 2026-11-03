package db

import (
	"context"
	"fmt"
	"slices"
	"time"
)

type TrendPoint struct {
	At    time.Time
	Value float64
}

var (
	qTrendVotes = register("TrendVotes", `
SELECT v.user_id, v.rate, coalesce(v.date, a.created_at),
       EXISTS (SELECT 1 FROM pwikit_vote_event e WHERE e.article_id = v.article_id AND e.user_id = v.user_id)
FROM web_vote v JOIN web_article a ON a.id = v.article_id
WHERE v.article_id = $1 AND a.site_id = $2`)

	qTrendEvents = register("TrendEvents", `
SELECT old_rate, new_rate, at FROM pwikit_vote_event WHERE article_id = $1 ORDER BY at, id`)

	qArticleCreated = register("ArticleCreated", `SELECT created_at FROM web_article WHERE id = $1 AND site_id = $2`)
)

type trendStep struct {
	at    time.Time
	sum   float64
	count int
	order int
}

// Votes from before the history was kept count from when they were cast, or
// from the page's creation when that is unknown, so the last point is the
// rating now.
func (d *DB) RatingTrend(ctx context.Context, siteID, articleID int64, average bool) ([]TrendPoint, error) {
	var created time.Time
	if err := d.pool.QueryRow(ctx, qArticleCreated, articleID, siteID).Scan(&created); err != nil {
		return nil, fmt.Errorf("read creation of %d: %w", articleID, err)
	}

	var steps []trendStep
	var sum float64
	var count int
	rows, err := d.pool.Query(ctx, qTrendVotes, articleID, siteID)
	if err != nil {
		return nil, fmt.Errorf("read votes of %d: %w", articleID, err)
	}
	for rows.Next() {
		var user int64
		var rate float64
		var at time.Time
		var changed bool
		if err := rows.Scan(&user, &rate, &at, &changed); err != nil {
			rows.Close()
			return nil, err
		}
		sum += rate
		count++
		if !changed {
			steps = append(steps, trendStep{at: at, sum: rate, count: 1, order: 0})
		}
	}
	rows.Close()

	var eventSum float64
	var eventCount int
	rows, err = d.pool.Query(ctx, qTrendEvents, articleID)
	if err != nil {
		return nil, fmt.Errorf("read vote history of %d: %w", articleID, err)
	}
	for rows.Next() {
		var before, after *float64
		var at time.Time
		if err := rows.Scan(&before, &after, &at); err != nil {
			rows.Close()
			return nil, err
		}
		step := trendStep{at: at, order: 1}
		if before != nil {
			step.sum -= *before
			step.count--
		}
		if after != nil {
			step.sum += *after
			step.count++
		}
		eventSum += step.sum
		eventCount += step.count
		steps = append(steps, step)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// What the dated votes and the history do not account for was already there
	// before either, so it stands at the page's creation.
	placedSum, placedCount := eventSum, eventCount
	for _, s := range steps {
		if s.order == 0 {
			placedSum += s.sum
			placedCount += s.count
		}
	}
	if rest, n := sum-placedSum, count-placedCount; rest != 0 || n != 0 {
		steps = append(steps, trendStep{at: created, sum: rest, count: n, order: -1})
	}

	slices.SortStableFunc(steps, func(a, b trendStep) int {
		if c := a.at.Compare(b.at); c != 0 {
			return c
		}
		return a.order - b.order
	})
	out := make([]TrendPoint, 0, len(steps)+1)
	out = append(out, TrendPoint{At: created})
	var runSum float64
	var runCount int
	for _, s := range steps {
		runSum += s.sum
		runCount += s.count
		value := runSum
		if average {
			value = 0
			if runCount > 0 {
				value = runSum / float64(runCount)
			}
		}
		if n := len(out); n > 0 && out[n-1].At.Equal(s.at) {
			out[n-1].Value = value
			continue
		}
		out = append(out, TrendPoint{At: s.at, Value: value})
	}
	return out, nil
}
