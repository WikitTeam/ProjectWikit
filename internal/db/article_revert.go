package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const LogRevert = "revert"

type RestoredVote struct {
	UserID int64
	RoleID *int64
	Rate   float64
	Date   *time.Time
}

var (
	qArticleLogAbove = register("ArticleLogAbove", `
SELECT rev_number, type, meta, comment, created_at, user_id
FROM web_articlelogentry
WHERE article_id = $1 AND rev_number > $2
ORDER BY rev_number DESC`)

	// The version before another is the one written last before it, which is not
	// the one with the next lower id when an import wrote them out of order.
	qPreviousVersionSource = register("PreviousVersionSource", `
SELECT p.source
FROM web_articleversion v
JOIN web_articleversion p ON p.article_id = v.article_id AND p.created_at < v.created_at
WHERE v.id = $1
ORDER BY p.created_at DESC
LIMIT 1`)
)

func (d *DB) ArticleLogAbove(ctx context.Context, articleID int64, revNumber int) ([]LogEntry, error) {
	rows, err := d.pool.Query(ctx, qArticleLogAbove, articleID, revNumber)
	if err != nil {
		return nil, fmt.Errorf("query log of article %d: %w", articleID, err)
	}
	defer rows.Close()

	var out []LogEntry
	for rows.Next() {
		var e LogEntry
		if err := rows.Scan(&e.RevNumber, &e.Type, &e.Meta, &e.Comment, &e.CreatedAt, &e.UserID); err != nil {
			return nil, fmt.Errorf("scan log entry of article %d: %w", articleID, err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (d *DB) PreviousVersionSource(ctx context.Context, versionID int64) (string, error) {
	var source string
	err := d.pool.QueryRow(ctx, qPreviousVersionSource, versionID).Scan(&source)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("query version before %d: %w", versionID, err)
	}
	return source, nil
}

var (
	qReadFileName = register("ReadFileName", `
SELECT name FROM web_file WHERE id = $1 FOR UPDATE`)

	qRenameFile = register("RenameFile", `
UPDATE web_file SET name = $2 WHERE id = $1`)

	qSoftDeleteFile = register("SoftDeleteFile", `
UPDATE web_file SET deleted_at = $2, deleted_by_id = $3
WHERE id = $1 AND deleted_at IS NULL
RETURNING name`)

	qRestoreFile = register("RestoreFile", `
UPDATE web_file SET deleted_at = NULL, deleted_by_id = NULL
WHERE id = $1 AND deleted_at IS NOT NULL
RETURNING name`)
)

// A file that is already in the asked-for state reports no name, which is how
// the caller knows to leave it out of the revision it writes.
func (d *DB) RenameFile(ctx context.Context, fileID int64, name string) (string, bool, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return "", false, fmt.Errorf("begin rename of file %d: %w", fileID, err)
	}
	defer tx.Rollback(ctx)

	var previous string
	err = tx.QueryRow(ctx, qReadFileName, fileID).Scan(&previous)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read file %d: %w", fileID, err)
	}
	if _, err := tx.Exec(ctx, qRenameFile, fileID, name); err != nil {
		return "", false, fmt.Errorf("rename file %d: %w", fileID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", false, fmt.Errorf("commit rename of file %d: %w", fileID, err)
	}
	return previous, true, nil
}

func (d *DB) SoftDeleteFile(ctx context.Context, fileID int64, at time.Time, byUserID *int64) (string, bool, error) {
	return d.fileName(ctx, qSoftDeleteFile, fileID, at, byUserID)
}

func (d *DB) RestoreFile(ctx context.Context, fileID int64) (string, bool, error) {
	return d.fileName(ctx, qRestoreFile, fileID)
}

func (d *DB) fileName(ctx context.Context, sql string, fileID int64, args ...any) (string, bool, error) {
	var name string
	err := d.pool.QueryRow(ctx, sql, append([]any{fileID}, args...)...).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("touch file %d: %w", fileID, err)
	}
	return name, true, nil
}

var (
	qSetArticleTagIDs = register("SetArticleTagIDs", `
DELETE FROM web_article_tags WHERE article_id = $1 AND NOT (tag_id = ANY($2))`)

	qKnownTags = register("KnownTags", `SELECT id FROM web_tag WHERE id = ANY($1)`)

	qKnownUser = register("KnownUser", `SELECT id FROM web_user WHERE id = $1`)
)

// The revision that records this is written by the caller, so nothing is logged
// here.
func (d *DB) SetArticleTagIDs(ctx context.Context, articleID int64, tagIDs []int64) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tags of %d: %w", articleID, err)
	}
	defer tx.Rollback(ctx)

	wanted, err := scanIDs(ctx, tx, qKnownTags, tagIDs)
	if err != nil {
		return fmt.Errorf("look up tags of %d: %w", articleID, err)
	}
	if _, err := tx.Exec(ctx, qSetArticleTagIDs, articleID, wanted); err != nil {
		return fmt.Errorf("drop tags of %d: %w", articleID, err)
	}
	held, err := scanIDs(ctx, tx, qArticleTagIDs, articleID)
	if err != nil {
		return fmt.Errorf("read tags of %d: %w", articleID, err)
	}
	for _, id := range missingFrom(wanted, held) {
		if _, err := tx.Exec(ctx, qInsertArticleTag, articleID, id); err != nil {
			return fmt.Errorf("tag %d: %w", articleID, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tags of %d: %w", articleID, err)
	}
	return nil
}

// A vote whose voter is gone is dropped rather than failing the restore, since
// the row it would need points at a user that no longer exists.
func (d *DB) RestoreArticleVotes(ctx context.Context, articleID int64, votes []RestoredVote) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin votes of %d: %w", articleID, err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, qDeleteArticleVotes, articleID); err != nil {
		return fmt.Errorf("drop votes of %d: %w", articleID, err)
	}
	for _, vote := range votes {
		var known int64
		err := tx.QueryRow(ctx, qKnownUser, vote.UserID).Scan(&known)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return fmt.Errorf("look up voter %d: %w", vote.UserID, err)
		}
		if _, err := tx.Exec(ctx, qInsertVote, articleID, vote.UserID, vote.Rate, vote.Date, vote.RoleID); err != nil {
			return fmt.Errorf("restore vote of %d: %w", vote.UserID, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit votes of %d: %w", articleID, err)
	}
	return nil
}

type RevertWrite struct {
	ArticleID int64
	UserID    *int64
	Meta      json.RawMessage
	At        time.Time
}

func (d *DB) WriteRevertEntry(ctx context.Context, w RevertWrite) (Revision, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return Revision{}, fmt.Errorf("begin revert of %d: %w", w.ArticleID, err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, qLockArticleLog, w.ArticleID); err != nil {
		return Revision{}, fmt.Errorf("lock article log %d: %w", w.ArticleID, err)
	}
	var rev Revision
	if err := tx.QueryRow(ctx, qInsertArticleLog, w.ArticleID, w.UserID, LogRevert,
		string(w.Meta), "", w.At).Scan(&rev.EntryID, &rev.RevNumber); err != nil {
		return Revision{}, fmt.Errorf("write revision of %d: %w", w.ArticleID, err)
	}
	if _, err := tx.Exec(ctx, qTouchArticle, w.ArticleID, w.At); err != nil {
		return Revision{}, fmt.Errorf("touch article %d: %w", w.ArticleID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Revision{}, fmt.Errorf("commit revert of %d: %w", w.ArticleID, err)
	}
	return rev, nil
}

var (
	qAddArticleVersion = register("AddArticleVersion", `
INSERT INTO web_articleversion (article_id, source, created_at)
VALUES ($1, $2, $3)
RETURNING id`)

	qSetArticleTitle = register("SetArticleTitle", `
UPDATE web_article SET title = $2 WHERE id = $1`)

	qMoveArticleParent = register("MoveArticleParent", `
UPDATE web_article SET parent_id = $2 WHERE id = $1`)

	qArticleAuthorIDs = register("ArticleAuthorIDs", `
SELECT user_id FROM web_article_authors WHERE article_id = $1 ORDER BY user_id`)
)

// The revision a revert writes names every piece it moved, so the pieces
// themselves leave no revisions of their own.
func (d *DB) AddArticleVersion(ctx context.Context, articleID int64, source string, at time.Time) (int64, error) {
	var id int64
	if err := d.pool.QueryRow(ctx, qAddArticleVersion, articleID, source, at).Scan(&id); err != nil {
		return 0, fmt.Errorf("write version of %d: %w", articleID, err)
	}
	return id, nil
}

func (d *DB) SetArticleTitle(ctx context.Context, articleID int64, title string) error {
	if _, err := d.pool.Exec(ctx, qSetArticleTitle, articleID, title); err != nil {
		return fmt.Errorf("set title of %d: %w", articleID, err)
	}
	return nil
}

func (d *DB) MoveArticleParent(ctx context.Context, articleID int64, parentID *int64) error {
	if _, err := d.pool.Exec(ctx, qMoveArticleParent, articleID, parentID); err != nil {
		return fmt.Errorf("set parent of %d: %w", articleID, err)
	}
	return nil
}

func (d *DB) MoveArticle(ctx context.Context, articleID int64, category, name, from string) error {
	to := (&Article{Category: category, Name: name}).FullName()

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin move of %d: %w", articleID, err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, qRenameArticle, articleID, category, name); err != nil {
		return fmt.Errorf("move article %d: %w", articleID, err)
	}
	if _, err := tx.Exec(ctx, qDropLinksFrom, to); err != nil {
		return fmt.Errorf("drop links of %q: %w", to, err)
	}
	if _, err := tx.Exec(ctx, qMoveLinksFrom, from, to); err != nil {
		return fmt.Errorf("move links of %q: %w", from, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit move of %d: %w", articleID, err)
	}
	return nil
}

func (d *DB) ArticleAuthorIDs(ctx context.Context, articleID int64) ([]int64, error) {
	rows, err := d.pool.Query(ctx, qArticleAuthorIDs, articleID)
	if err != nil {
		return nil, fmt.Errorf("query authors of %d: %w", articleID, err)
	}
	defer rows.Close()

	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan author of %d: %w", articleID, err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
