package db

import (
	"context"
	"fmt"
	"time"
)

// One query for the whole page. A thread renders dozens of posts and asking per
// post would make the page cost grow with it.
var qPostLikeCounts = register("PostLikeCounts", `
SELECT post_id, count(*)
FROM web_forumpostlike
WHERE post_id = ANY($1)
GROUP BY post_id`)

func (d *DB) PostLikeCounts(ctx context.Context, postIDs []int64) (map[int64]int, error) {
	out := map[int64]int{}
	if len(postIDs) == 0 {
		return out, nil
	}
	rows, err := d.pool.Query(ctx, qPostLikeCounts, postIDs)
	if err != nil {
		return nil, fmt.Errorf("count likes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, fmt.Errorf("scan like count: %w", err)
		}
		out[id] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("count likes: %w", err)
	}
	return out, nil
}

var qPostsLikedBy = register("PostsLikedBy", `
SELECT post_id
FROM web_forumpostlike
WHERE user_id = $1 AND post_id = ANY($2)`)

func (d *DB) PostsLikedBy(ctx context.Context, userID int64, postIDs []int64) (map[int64]bool, error) {
	out := map[int64]bool{}
	if len(postIDs) == 0 {
		return out, nil
	}
	rows, err := d.pool.Query(ctx, qPostsLikedBy, userID, postIDs)
	if err != nil {
		return nil, fmt.Errorf("read likes of user %d: %w", userID, err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan liked post: %w", err)
		}
		out[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read likes of user %d: %w", userID, err)
	}
	return out, nil
}

var qLikePost = register("LikePost", `
INSERT INTO web_forumpostlike (post_id, user_id, created_at)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING`)

// A false answer means the like was already there, which is how the caller
// knows not to tell the author a second time.
func (d *DB) LikePost(ctx context.Context, postID, userID int64, at time.Time) (bool, error) {
	tag, err := d.pool.Exec(ctx, qLikePost, postID, userID, at)
	if err != nil {
		return false, fmt.Errorf("like post %d: %w", postID, err)
	}
	return tag.RowsAffected() > 0, nil
}

var qUnlikePost = register("UnlikePost", `
DELETE FROM web_forumpostlike WHERE post_id = $1 AND user_id = $2`)

func (d *DB) UnlikePost(ctx context.Context, postID, userID int64) error {
	if _, err := d.pool.Exec(ctx, qUnlikePost, postID, userID); err != nil {
		return fmt.Errorf("unlike post %d: %w", postID, err)
	}
	return nil
}

var qPostLikers = register("PostLikers", `
SELECT `+prefixed("u", userColumns)+`
FROM web_forumpostlike l
JOIN web_user u ON u.id = l.user_id
WHERE l.post_id = $1
ORDER BY l.created_at DESC, l.id DESC
OFFSET $2 LIMIT $3`)

func (d *DB) PostLikers(ctx context.Context, postID int64, offset, limit int) ([]User, error) {
	rows, err := d.pool.Query(ctx, qPostLikers, postID, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("list likers of post %d: %w", postID, err)
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		dest, finish := userDest(&u)
		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("scan liker: %w", err)
		}
		finish()
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list likers of post %d: %w", postID, err)
	}
	return out, nil
}

var qPostLikeCount = register("PostLikeCount", `
SELECT count(*) FROM web_forumpostlike WHERE post_id = $1`)

func (d *DB) PostLikeCount(ctx context.Context, postID int64) (int, error) {
	var n int
	if err := d.pool.QueryRow(ctx, qPostLikeCount, postID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count likes of post %d: %w", postID, err)
	}
	return n, nil
}
