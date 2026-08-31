package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

var qSiteRatingMode = register("SiteRatingMode", `
SELECT rating_mode
FROM web_settings
WHERE site_id = $1`)

func (d *DB) SiteRatingMode(ctx context.Context, siteID int64) (string, error) {
	var mode string
	err := d.pool.QueryRow(ctx, qSiteRatingMode, siteID).Scan(&mode)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("query rating mode of site %d: %w", siteID, err)
	}
	return mode, nil
}

var qCategoryRatingMode = register("CategoryRatingMode", `
SELECT s.rating_mode
FROM web_settings s
JOIN web_category c ON c.id = s.category_id
WHERE c.name = $1`)

// CategoryRatingMode reports ErrNotFound for a category that has no row of its
// own as well as for one that exists without settings; both fall back to the
// site.
func (d *DB) CategoryRatingMode(ctx context.Context, category string) (string, error) {
	var mode string
	err := d.pool.QueryRow(ctx, qCategoryRatingMode, category).Scan(&mode)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("query rating mode of category %q: %w", category, err)
	}
	return mode, nil
}

type VoteStats struct {
	Sum        float64
	Count      int
	GoodUpDown int
	Average    float64
	GoodStars  int
}

// Both rating modes are counted in one pass. Which pair of columns is read
// depends on a setting the caller resolves, and a second round trip to learn
// that first would cost more than the two unused counts.
var qVoteStats = register("VoteStats", `
SELECT COALESCE(SUM(rate), 0),
       COUNT(rate),
       COUNT(rate) FILTER (WHERE rate = 1),
       COALESCE(AVG(rate), 0),
       COUNT(rate) FILTER (WHERE rate >= 3)
FROM web_vote
WHERE article_id = $1`)

func (d *DB) VoteStats(ctx context.Context, articleID int64) (VoteStats, error) {
	var s VoteStats
	err := d.pool.QueryRow(ctx, qVoteStats, articleID).Scan(
		&s.Sum, &s.Count, &s.GoodUpDown, &s.Average, &s.GoodStars)
	if err != nil {
		return VoteStats{}, fmt.Errorf("query votes of article %d: %w", articleID, err)
	}
	return s, nil
}

var qHasVoted = register("HasVoted", `
SELECT EXISTS(
	SELECT 1 FROM web_vote
	WHERE article_id = $1 AND user_id IS NOT DISTINCT FROM $2::bigint)`)

func (d *DB) HasVoted(ctx context.Context, articleID int64, userID *int64) (bool, error) {
	var voted bool
	if err := d.pool.QueryRow(ctx, qHasVoted, articleID, userID).Scan(&voted); err != nil {
		return false, fmt.Errorf("query vote of article %d: %w", articleID, err)
	}
	return voted, nil
}

var qVoteByUser = register("VoteByUser", `
SELECT rate
FROM web_vote
WHERE article_id = $1 AND user_id IS NOT DISTINCT FROM $2
ORDER BY id DESC
LIMIT 1`)

func (d *DB) VoteByUser(ctx context.Context, articleID int64, userID *int64) (float64, bool, error) {
	var rate float64
	err := d.pool.QueryRow(ctx, qVoteByUser, articleID, userID).Scan(&rate)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("query vote on article %d: %w", articleID, err)
	}
	return rate, true, nil
}
