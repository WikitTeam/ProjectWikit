package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type ForumSection struct {
	ID               int64
	Name             string
	Description      string
	IsHidden         bool
	IsHiddenForUsers bool
}

type ForumCategory struct {
	ID            int64
	SectionID     int64
	Name          string
	Description   string
	IsForComments bool
}

// ForumLastPost carries ids rather than rows so the caller can reach for the
// article and the author through the lookups it already has.
type ForumLastPost struct {
	ID               int64
	CreatedAt        time.Time
	ThreadID         int64
	ThreadName       string
	ThreadCategoryID *int64
	ThreadArticleID  *int64
	AuthorID         *int64
}

type ForumCounts struct {
	Threads int
	Posts   int
}

const forumSectionColumns = `id, name, description, is_hidden, is_hidden_for_users`

var qForumSections = register("ForumSections", `
SELECT `+forumSectionColumns+`
FROM web_forumsection
ORDER BY "order", id`)

func (d *DB) ForumSections(ctx context.Context) ([]ForumSection, error) {
	rows, err := d.pool.Query(ctx, qForumSections)
	if err != nil {
		return nil, fmt.Errorf("query forum sections: %w", err)
	}
	defer rows.Close()

	var out []ForumSection
	for rows.Next() {
		var s ForumSection
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.IsHidden, &s.IsHiddenForUsers); err != nil {
			return nil, fmt.Errorf("scan forum section: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

var qForumSection = register("ForumSection", `
SELECT `+forumSectionColumns+`
FROM web_forumsection
WHERE id = $1`)

func (d *DB) ForumSection(ctx context.Context, id int64) (*ForumSection, error) {
	var s ForumSection
	err := d.pool.QueryRow(ctx, qForumSection, id).Scan(
		&s.ID, &s.Name, &s.Description, &s.IsHidden, &s.IsHiddenForUsers)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query forum section %d: %w", id, err)
	}
	return &s, nil
}

var qForumCategories = register("ForumCategories", `
SELECT id, section_id, name, description, is_for_comments
FROM web_forumcategory
ORDER BY "order", id`)

func (d *DB) ForumCategories(ctx context.Context) ([]ForumCategory, error) {
	rows, err := d.pool.Query(ctx, qForumCategories)
	if err != nil {
		return nil, fmt.Errorf("query forum categories: %w", err)
	}
	defer rows.Close()

	var out []ForumCategory
	for rows.Next() {
		var c ForumCategory
		if err := rows.Scan(&c.ID, &c.SectionID, &c.Name, &c.Description, &c.IsForComments); err != nil {
			return nil, fmt.Errorf("scan forum category: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

var qForumCategoryCounts = register("ForumCategoryCounts", `
SELECT (SELECT count(*) FROM web_forumthread WHERE category_id = $1),
       (SELECT count(*) FROM web_forumpost p
        JOIN web_forumthread t ON t.id = p.thread_id
        WHERE t.category_id = $1)`)

func (d *DB) ForumCategoryCounts(ctx context.Context, categoryID int64) (ForumCounts, error) {
	var c ForumCounts
	if err := d.pool.QueryRow(ctx, qForumCategoryCounts, categoryID).Scan(&c.Threads, &c.Posts); err != nil {
		return ForumCounts{}, fmt.Errorf("count forum category %d: %w", categoryID, err)
	}
	return c, nil
}

// A category marked for comments counts every article's thread, whichever
// category the reader is looking at.
var qForumCommentCounts = register("ForumCommentCounts", `
SELECT (SELECT count(*) FROM web_forumthread WHERE article_id IS NOT NULL),
       (SELECT count(*) FROM web_forumpost p
        JOIN web_forumthread t ON t.id = p.thread_id
        WHERE t.article_id IS NOT NULL)`)

func (d *DB) ForumCommentCounts(ctx context.Context) (ForumCounts, error) {
	var c ForumCounts
	if err := d.pool.QueryRow(ctx, qForumCommentCounts).Scan(&c.Threads, &c.Posts); err != nil {
		return ForumCounts{}, fmt.Errorf("count forum comments: %w", err)
	}
	return c, nil
}

const forumLastPostColumns = `p.id, p.created_at, t.id, t.name, t.category_id, t.article_id, p.author_id`

// Ties are left to the database because Django orders on the timestamp alone.
var qForumCategoryLastPost = register("ForumCategoryLastPost", `
SELECT `+forumLastPostColumns+`
FROM web_forumpost p
JOIN web_forumthread t ON t.id = p.thread_id
WHERE t.category_id = $1
ORDER BY p.created_at DESC
LIMIT 1`)

func (d *DB) ForumCategoryLastPost(ctx context.Context, categoryID int64) (*ForumLastPost, error) {
	return d.scanLastPost(ctx, qForumCategoryLastPost, categoryID)
}

var qForumCommentLastPost = register("ForumCommentLastPost", `
SELECT `+forumLastPostColumns+`
FROM web_forumpost p
JOIN web_forumthread t ON t.id = p.thread_id
WHERE t.article_id IS NOT NULL
ORDER BY p.created_at DESC
LIMIT 1`)

func (d *DB) ForumCommentLastPost(ctx context.Context) (*ForumLastPost, error) {
	return d.scanLastPost(ctx, qForumCommentLastPost)
}

func (d *DB) scanLastPost(ctx context.Context, sql string, args ...any) (*ForumLastPost, error) {
	var p ForumLastPost
	err := d.pool.QueryRow(ctx, sql, args...).Scan(
		&p.ID, &p.CreatedAt, &p.ThreadID, &p.ThreadName, &p.ThreadCategoryID, &p.ThreadArticleID, &p.AuthorID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query last forum post: %w", err)
	}
	return &p, nil
}
