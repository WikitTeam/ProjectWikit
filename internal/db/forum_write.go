package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type ForumPostVersion struct {
	CreatedAt time.Time
	AuthorID  *int64
}

var qForumPostVersions = register("ForumPostVersions", `
SELECT created_at, author_id
FROM web_forumpostversion
WHERE post_id = $1
ORDER BY created_at DESC`)

func (d *DB) ForumPostVersions(ctx context.Context, postID int64) ([]ForumPostVersion, error) {
	rows, err := d.pool.Query(ctx, qForumPostVersions, postID)
	if err != nil {
		return nil, fmt.Errorf("list versions of post %d: %w", postID, err)
	}
	defer rows.Close()

	var out []ForumPostVersion
	for rows.Next() {
		var v ForumPostVersion
		if err := rows.Scan(&v.CreatedAt, &v.AuthorID); err != nil {
			return nil, fmt.Errorf("scan post version: %w", err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list versions of post %d: %w", postID, err)
	}
	return out, nil
}

var (
	qForumPostSource = register("ForumPostSource", `
SELECT source FROM web_forumpostversion
WHERE post_id = $1
ORDER BY created_at DESC
LIMIT 1`)

	qForumPostSourceAt = register("ForumPostSourceAt", `
SELECT source FROM web_forumpostversion
WHERE post_id = $1 AND created_at <= $2
ORDER BY created_at DESC
LIMIT 1`)
)

// A post with no version behind it reads as empty rather than as missing, which
// is the same thing the thread page shows for it.
func (d *DB) ForumPostSource(ctx context.Context, postID int64, at *time.Time) (string, error) {
	var source string
	var err error
	if at == nil {
		err = d.pool.QueryRow(ctx, qForumPostSource, postID).Scan(&source)
	} else {
		err = d.pool.QueryRow(ctx, qForumPostSourceAt, postID, *at).Scan(&source)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read source of post %d: %w", postID, err)
	}
	return source, nil
}

type ForumPostWrite struct {
	ThreadID  int64
	Name      string
	Source    string
	AuthorID  *int64
	ReplyToID *int64
	At        time.Time
}

var (
	qInsertForumPost = register("InsertForumPost", `
INSERT INTO web_forumpost (thread_id, name, author_id, reply_to_id, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $5)
RETURNING id`)

	qInsertForumPostVersion = register("InsertForumPostVersion", `
INSERT INTO web_forumpostversion (post_id, source, author_id, created_at)
VALUES ($1, $2, $3, $4)`)

	qTouchForumThread = register("TouchForumThread", `
UPDATE web_forumthread SET updated_at = $2 WHERE id = $1`)
)

// The post, its first version and the thread's timestamp move together, so a
// post that fails halfway leaves no body-less row behind.
func (d *DB) CreateForumPost(ctx context.Context, w ForumPostWrite) (int64, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin post: %w", err)
	}
	defer tx.Rollback(ctx)

	var id int64
	if err := tx.QueryRow(ctx, qInsertForumPost, w.ThreadID, w.Name, w.AuthorID,
		w.ReplyToID, w.At).Scan(&id); err != nil {
		return 0, fmt.Errorf("write post in thread %d: %w", w.ThreadID, err)
	}
	if _, err := tx.Exec(ctx, qInsertForumPostVersion, id, w.Source, w.AuthorID, w.At); err != nil {
		return 0, fmt.Errorf("write first version of post %d: %w", id, err)
	}
	if _, err := tx.Exec(ctx, qTouchForumThread, w.ThreadID, w.At); err != nil {
		return 0, fmt.Errorf("touch thread %d: %w", w.ThreadID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit post in thread %d: %w", w.ThreadID, err)
	}
	return id, nil
}

var qRenameForumPost = register("RenameForumPost", `
UPDATE web_forumpost SET name = $2, updated_at = $3 WHERE id = $1`)

// A body that did not change leaves no version, so the edit history only holds
// the times the text actually moved.
func (d *DB) UpdateForumPost(ctx context.Context, postID int64, name, source, previous string,
	authorID *int64, at time.Time) error {

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin edit of post %d: %w", postID, err)
	}
	defer tx.Rollback(ctx)

	if source != previous {
		if _, err := tx.Exec(ctx, qInsertForumPostVersion, postID, source, authorID, at); err != nil {
			return fmt.Errorf("write version of post %d: %w", postID, err)
		}
	}
	if _, err := tx.Exec(ctx, qRenameForumPost, postID, name, at); err != nil {
		return fmt.Errorf("edit post %d: %w", postID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit edit of post %d: %w", postID, err)
	}
	return nil
}

// Replies to a deleted post become roots of the thread rather than following it
// down, so a whole branch does not vanish with one post.
var forumPostChildren = []string{
	`UPDATE web_forumpost SET reply_to_id = NULL WHERE reply_to_id = $1`,
	`DELETE FROM web_forumpostversion WHERE post_id = $1`,
	`DELETE FROM web_forumpost WHERE id = $1`,
}

var qDeleteForumPost = registerAll("DeleteForumPost", forumPostChildren)

func (d *DB) DeleteForumPost(ctx context.Context, postID int64) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete of post %d: %w", postID, err)
	}
	defer tx.Rollback(ctx)

	for _, sql := range qDeleteForumPost {
		if _, err := tx.Exec(ctx, sql, postID); err != nil {
			return fmt.Errorf("delete post %d: %w", postID, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete of post %d: %w", postID, err)
	}
	return nil
}

type ForumThreadWrite struct {
	CategoryID  int64
	Name        string
	Description string
	AuthorID    *int64
	At          time.Time
}

var qInsertForumThread = register("InsertForumThread", `
INSERT INTO web_forumthread (category_id, name, description, author_id, created_at, updated_at, is_pinned, is_locked)
VALUES ($1, $2, $3, $4, $5, $5, false, false)
RETURNING id`)

// The thread and its first post go in together, so a thread that fails halfway
// does not show up empty in the category listing.
func (d *DB) CreateForumThread(ctx context.Context, w ForumThreadWrite, source string) (int64, int64, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("begin thread: %w", err)
	}
	defer tx.Rollback(ctx)

	var threadID int64
	if err := tx.QueryRow(ctx, qInsertForumThread, w.CategoryID, w.Name, w.Description,
		w.AuthorID, w.At).Scan(&threadID); err != nil {
		return 0, 0, fmt.Errorf("write thread in category %d: %w", w.CategoryID, err)
	}
	var postID int64
	if err := tx.QueryRow(ctx, qInsertForumPost, threadID, w.Name, w.AuthorID,
		nil, w.At).Scan(&postID); err != nil {
		return 0, 0, fmt.Errorf("write first post of thread %d: %w", threadID, err)
	}
	if _, err := tx.Exec(ctx, qInsertForumPostVersion, postID, source, w.AuthorID, w.At); err != nil {
		return 0, 0, fmt.Errorf("write first version of post %d: %w", postID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, 0, fmt.Errorf("commit thread in category %d: %w", w.CategoryID, err)
	}
	return threadID, postID, nil
}

type ForumThreadEdit struct {
	Name        *string
	Description *string
	Locked      *bool
	Pinned      *bool
	CategoryID  *int64
}

var qUpdateForumThread = register("UpdateForumThread", `
UPDATE web_forumthread SET
	name = COALESCE($2, name),
	description = COALESCE($3, description),
	is_locked = COALESCE($4, is_locked),
	is_pinned = COALESCE($5, is_pinned),
	category_id = COALESCE($6, category_id)
WHERE id = $1`)

func (d *DB) UpdateForumThread(ctx context.Context, threadID int64, e ForumThreadEdit) error {
	_, err := d.pool.Exec(ctx, qUpdateForumThread, threadID,
		e.Name, e.Description, e.Locked, e.Pinned, e.CategoryID)
	if err != nil {
		return fmt.Errorf("edit thread %d: %w", threadID, err)
	}
	return nil
}

var qActiveUsersByNames = register("ActiveUsersByNames", `
SELECT `+userColumns+`
FROM web_user
WHERE lower(username) = ANY($1) AND is_active AND type IN ('normal', 'bot')
ORDER BY id`)

// A mention names someone by the name they type, which is matched without case
// so a post that writes it differently still reaches them.
func (d *DB) ActiveUsersByNames(ctx context.Context, names []string) ([]User, error) {
	if len(names) == 0 {
		return nil, nil
	}
	rows, err := d.pool.Query(ctx, qActiveUsersByNames, names)
	if err != nil {
		return nil, fmt.Errorf("look up mentioned users: %w", err)
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		dest, finish := userDest(&u)
		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("scan mentioned user: %w", err)
		}
		finish()
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read mentioned users: %w", err)
	}
	return out, nil
}
