package db

import (
	"context"
	"fmt"
	"time"
)

type RatedArticle struct {
	Article Article
	Rate    float64
	VotedAt *time.Time
}

var qRatedByCountOf = register("RatedByCountOf", `
SELECT count(*) FROM web_vote WHERE user_id = $1`)

func (d *DB) RatedByCountOf(ctx context.Context, userID int64) (int, error) {
	var n int
	if err := d.pool.QueryRow(ctx, qRatedByCountOf, userID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count votes of user %d: %w", userID, err)
	}
	return n, nil
}

var qRatedBy = register("RatedBy", `
SELECT `+prefixedArticleColumns+`, v.rate, v.date
FROM web_vote v
JOIN web_article a ON a.id = v.article_id
WHERE v.user_id = $1
ORDER BY v.date DESC NULLS LAST, v.id DESC
OFFSET $2 LIMIT $3`)

func (d *DB) RatedBy(ctx context.Context, userID int64, offset, limit int) ([]RatedArticle, error) {
	rows, err := d.pool.Query(ctx, qRatedBy, userID, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("list votes of user %d: %w", userID, err)
	}
	defer rows.Close()

	var out []RatedArticle
	for rows.Next() {
		var one RatedArticle
		a := &one.Article
		if err := rows.Scan(&a.ID, &a.Category, &a.Name, &a.Title, &a.ParentID,
			&a.Locked, &a.CreatedAt, &a.UpdatedAt, &a.MediaName, &one.Rate, &one.VotedAt); err != nil {
			return nil, fmt.Errorf("scan vote: %w", err)
		}
		out = append(out, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list votes of user %d: %w", userID, err)
	}
	return out, nil
}

type LikedPost struct {
	Post       ForumThreadPost
	ThreadName string
	LikedAt    time.Time
}

var qLikedPostCountOf = register("LikedPostCountOf", `
SELECT count(*) FROM web_forumpostlike WHERE user_id = $1`)

func (d *DB) LikedPostCountOf(ctx context.Context, userID int64) (int, error) {
	var n int
	if err := d.pool.QueryRow(ctx, qLikedPostCountOf, userID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count likes of user %d: %w", userID, err)
	}
	return n, nil
}

var qLikedPostsOf = register("LikedPostsOf", `
SELECT p.id, p.thread_id, p.name, p.created_at, p.updated_at, p.author_id, p.reply_to_id,
       coalesce(t.name, ''), l.created_at
FROM web_forumpostlike l
JOIN web_forumpost p ON p.id = l.post_id
JOIN web_forumthread t ON t.id = p.thread_id
WHERE l.user_id = $1
ORDER BY l.created_at DESC, l.id DESC
OFFSET $2 LIMIT $3`)

func (d *DB) LikedPostsOf(ctx context.Context, userID int64, offset, limit int) ([]LikedPost, error) {
	rows, err := d.pool.Query(ctx, qLikedPostsOf, userID, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("list likes of user %d: %w", userID, err)
	}
	defer rows.Close()

	var out []LikedPost
	for rows.Next() {
		var one LikedPost
		p := &one.Post
		if err := rows.Scan(&p.ID, &p.ThreadID, &p.Name, &p.CreatedAt, &p.UpdatedAt,
			&p.AuthorID, &p.ReplyToID, &one.ThreadName, &one.LikedAt); err != nil {
			return nil, fmt.Errorf("scan like: %w", err)
		}
		out = append(out, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list likes of user %d: %w", userID, err)
	}
	return out, nil
}
