package db

import (
	"context"
	"fmt"
	"time"
)

var qArticleFavouriteCount = register("ArticleFavouriteCount", `
SELECT count(*) FROM web_articlefavourite WHERE article_id = $1`)

func (d *DB) ArticleFavouriteCount(ctx context.Context, articleID int64) (int, error) {
	var n int
	if err := d.pool.QueryRow(ctx, qArticleFavouriteCount, articleID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count favourites of article %d: %w", articleID, err)
	}
	return n, nil
}

var qHasFavourited = register("HasFavourited", `
SELECT EXISTS (SELECT 1 FROM web_articlefavourite WHERE article_id = $1 AND user_id = $2)`)

func (d *DB) HasFavourited(ctx context.Context, articleID, userID int64) (bool, error) {
	var yes bool
	if err := d.pool.QueryRow(ctx, qHasFavourited, articleID, userID).Scan(&yes); err != nil {
		return false, fmt.Errorf("read favourite of article %d: %w", articleID, err)
	}
	return yes, nil
}

var qAddFavourite = register("AddFavourite", `
INSERT INTO web_articlefavourite (article_id, user_id, created_at)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING`)

func (d *DB) AddFavourite(ctx context.Context, articleID, userID int64, at time.Time) error {
	if _, err := d.pool.Exec(ctx, qAddFavourite, articleID, userID, at); err != nil {
		return fmt.Errorf("favourite article %d: %w", articleID, err)
	}
	return nil
}

var qRemoveFavourite = register("RemoveFavourite", `
DELETE FROM web_articlefavourite WHERE article_id = $1 AND user_id = $2`)

func (d *DB) RemoveFavourite(ctx context.Context, articleID, userID int64) error {
	if _, err := d.pool.Exec(ctx, qRemoveFavourite, articleID, userID); err != nil {
		return fmt.Errorf("unfavourite article %d: %w", articleID, err)
	}
	return nil
}

type Favourite struct {
	Article Article
	AddedAt time.Time
}

var qFavouritesOf = register("FavouritesOf", `
SELECT `+prefixedArticleColumns+`, f.created_at
FROM web_articlefavourite f
JOIN web_article a ON a.id = f.article_id
WHERE f.user_id = $1
ORDER BY f.created_at DESC, f.id DESC
OFFSET $2 LIMIT $3`)

// Only the owner ever reads this, so no permission filter runs here. Whoever
// calls it has already established that the rows belong to the reader.
func (d *DB) FavouritesOf(ctx context.Context, userID int64, offset, limit int) ([]Favourite, error) {
	rows, err := d.pool.Query(ctx, qFavouritesOf, userID, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("list favourites of user %d: %w", userID, err)
	}
	defer rows.Close()

	var out []Favourite
	for rows.Next() {
		var one Favourite
		a := &one.Article
		if err := rows.Scan(&a.ID, &a.Category, &a.Name, &a.Title, &a.ParentID,
			&a.Locked, &a.CreatedAt, &a.UpdatedAt, &a.MediaName, &one.AddedAt); err != nil {
			return nil, fmt.Errorf("scan favourite: %w", err)
		}
		out = append(out, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list favourites of user %d: %w", userID, err)
	}
	return out, nil
}

var qFavouriteCountOf = register("FavouriteCountOf", `
SELECT count(*) FROM web_articlefavourite WHERE user_id = $1`)

func (d *DB) FavouriteCountOf(ctx context.Context, userID int64) (int, error) {
	var n int
	if err := d.pool.QueryRow(ctx, qFavouriteCountOf, userID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count favourites of user %d: %w", userID, err)
	}
	return n, nil
}
