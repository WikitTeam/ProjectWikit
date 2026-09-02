package db

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5"
)

type Vote struct {
	ID     int64
	Rate   float64
	Date   *time.Time
	RoleID *int64
}

type ArticleVote struct {
	User       User
	Rate       float64
	Date       *time.Time
	RoleID     *int64
	GroupTitle string
	GroupIndex *int
}

var qArticleVotes = register("ArticleVotes", `
SELECT `+prefixed("u", userColumns)+`, v.rate, v.date, v.role_id,
       COALESCE(NULLIF(r.votes_title, ''), NULLIF(r.name, ''), r.slug), r.index
FROM web_vote v
JOIN web_user u ON u.id = v.user_id
LEFT JOIN web_role r ON r.id = v.role_id
WHERE v.article_id = $1
ORDER BY v.date DESC, u.username DESC`)

func (d *DB) ArticleVotes(ctx context.Context, articleID int64) ([]ArticleVote, error) {
	rows, err := d.pool.Query(ctx, qArticleVotes, articleID)
	if err != nil {
		return nil, fmt.Errorf("query votes of article %d: %w", articleID, err)
	}
	defer rows.Close()

	var out []ArticleVote
	for rows.Next() {
		var v ArticleVote
		var title *string
		dest, finish := userDest(&v.User)
		dest = append(dest, &v.Rate, &v.Date, &v.RoleID, &title, &v.GroupIndex)
		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("scan vote: %w", err)
		}
		finish()
		if title != nil {
			v.GroupTitle = *title
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

var (
	qVoteOfUser = register("VoteOfUser", `
SELECT id, rate, date, role_id
FROM web_vote
WHERE article_id = $1 AND user_id = $2`)

	qDeleteVotesOfUser = register("DeleteVotesOfUser", `
DELETE FROM web_vote WHERE article_id = $1 AND user_id = $2`)

	qInsertVote = register("InsertVote", `
INSERT INTO web_vote (article_id, user_id, rate, date, role_id)
VALUES ($1, $2, $3, $4, $5)`)
)

// The vote that was there comes back, since the action log records what
// changed rather than what it changed to.
func (d *DB) ReplaceVote(ctx context.Context, articleID, userID int64, rate *float64, roleID *int64, at time.Time) (*Vote, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin vote: %w", err)
	}
	defer tx.Rollback(ctx)

	var old *Vote
	var found Vote
	err = tx.QueryRow(ctx, qVoteOfUser, articleID, userID).Scan(&found.ID, &found.Rate, &found.Date, &found.RoleID)
	switch {
	case err == nil:
		old = &found
	case !errors.Is(err, pgx.ErrNoRows):
		return nil, fmt.Errorf("read vote: %w", err)
	}

	if _, err := tx.Exec(ctx, qDeleteVotesOfUser, articleID, userID); err != nil {
		return nil, fmt.Errorf("delete vote: %w", err)
	}
	if rate != nil {
		if _, err := tx.Exec(ctx, qInsertVote, articleID, userID, *rate, at, roleID); err != nil {
			return nil, fmt.Errorf("insert vote: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit vote: %w", err)
	}
	return old, nil
}

var qDeleteArticleVotes = register("DeleteArticleVotes", `
DELETE FROM web_vote WHERE article_id = $1`)

func (d *DB) DeleteArticleVotes(ctx context.Context, articleID int64) error {
	if _, err := d.pool.Exec(ctx, qDeleteArticleVotes, articleID); err != nil {
		return fmt.Errorf("delete votes of article %d: %w", articleID, err)
	}
	return nil
}

var qVoteGroupRole = register("VoteGroupRole", `
SELECT r.id
FROM web_role r
LEFT JOIN web_user_roles link ON link.role_id = r.id AND link.user_id = $1
WHERE r.group_votes AND (r.slug = ANY($2) OR link.user_id IS NOT NULL)
ORDER BY CASE r.slug WHEN 'registered' THEN 0 WHEN 'everyone' THEN 1 ELSE 2 END, r.index
LIMIT 1`)

// VoteGroupRole answers which role a vote is filed under. The two built-in
// roles outrank the user's own, and for an anonymous reader only everyone can.
func (d *DB) VoteGroupRole(ctx context.Context, userID *int64) (*int64, error) {
	slugs := []string{"everyone"}
	var of int64
	if userID != nil {
		slugs = append(slugs, "registered")
		of = *userID
	}

	var id int64
	err := d.pool.QueryRow(ctx, qVoteGroupRole, of, slugs).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query vote role: %w", err)
	}
	return &id, nil
}

// The action log's own vocabulary, which is not the article log's.
const (
	ActionVote          = "vote"
	ActionCreateArticle = "create_article"
	ActionEditArticle   = "edit_article"
	ActionRemoveArticle = "remove_article"
)

var qAddActionLog = register("AddActionLog", `
INSERT INTO web_actionlogentry (user_id, stale_username, type, meta, created_at, origin_ip)
VALUES ($1, $2, $3, $4, $5, $6)`)

func (d *DB) AddActionLog(ctx context.Context, userID *int64, username, kind, meta string, ip *netip.Addr, at time.Time) error {
	var origin *string
	if ip != nil {
		text := ip.String()
		origin = &text
	}
	if _, err := d.pool.Exec(ctx, qAddActionLog, userID, username, kind, meta, at, origin); err != nil {
		return fmt.Errorf("write action log: %w", err)
	}
	return nil
}

const (
	LogVotesDeleted = "votes_deleted"
)

var (
	qLockArticleLog = register("LockArticleLog", `SELECT pg_advisory_xact_lock($1)`)

	qInsertArticleLog = register("InsertArticleLog", `
INSERT INTO web_articlelogentry (article_id, user_id, type, meta, comment, created_at, rev_number)
SELECT $1, $2, $3, $4, $5, $6, COALESCE(MAX(rev_number), -1) + 1
FROM web_articlelogentry WHERE article_id = $1
RETURNING rev_number`)

	qTouchArticle = register("TouchArticle", `
UPDATE web_article SET updated_at = $2 WHERE id = $1`)
)

// The revision is numbered under a lock on this article alone, so two writers
// cannot pick the same number and neither waits on a page it is not touching.
func (d *DB) AddArticleLogEntry(ctx context.Context, articleID int64, userID *int64,
	kind, comment, meta string, at time.Time) (int, error) {

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin log entry: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, qLockArticleLog, articleID); err != nil {
		return 0, fmt.Errorf("lock article log %d: %w", articleID, err)
	}
	var revNumber int
	if err := tx.QueryRow(ctx, qInsertArticleLog, articleID, userID, kind, meta, comment, at).Scan(&revNumber); err != nil {
		return 0, fmt.Errorf("write log entry: %w", err)
	}
	if _, err := tx.Exec(ctx, qTouchArticle, articleID, at); err != nil {
		return 0, fmt.Errorf("touch article %d: %w", articleID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit log entry: %w", err)
	}
	return revNumber, nil
}
