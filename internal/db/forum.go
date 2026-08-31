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

type ForumThread struct {
	ID          int64
	Name        string
	Description string
	CategoryID  *int64
	ArticleID   *int64
	AuthorID    *int64
	IsPinned    bool
	IsLocked    bool
	CreatedAt   time.Time
}

type ForumThreadSort int

const (
	ForumThreadsByReply ForumThreadSort = iota
	ForumThreadsByStart
)

const forumThreadColumns = `id, name, description, category_id, article_id, author_id, is_pinned, is_locked, created_at`

var qForumThreadsByReply = register("ForumThreadsByReply", `
SELECT `+forumThreadColumns+`
FROM web_forumthread
WHERE category_id = $1
ORDER BY is_pinned DESC, updated_at DESC
OFFSET $2 LIMIT $3`)

var qForumThreadsByStart = register("ForumThreadsByStart", `
SELECT `+forumThreadColumns+`
FROM web_forumthread
WHERE category_id = $1
ORDER BY is_pinned DESC, created_at DESC
OFFSET $2 LIMIT $3`)

var qForumCommentThreadsByReply = register("ForumCommentThreadsByReply", `
SELECT `+forumThreadColumns+`
FROM web_forumthread
WHERE article_id IS NOT NULL
ORDER BY is_pinned DESC, updated_at DESC
OFFSET $1 LIMIT $2`)

var qForumCommentThreadsByStart = register("ForumCommentThreadsByStart", `
SELECT `+forumThreadColumns+`
FROM web_forumthread
WHERE article_id IS NOT NULL
ORDER BY is_pinned DESC, created_at DESC
OFFSET $1 LIMIT $2`)

func (d *DB) ForumThreads(ctx context.Context, categoryID int64, sort ForumThreadSort, offset, limit int) ([]ForumThread, error) {
	sql, args := qForumThreadsByReply, []any{categoryID, offset, limit}
	if sort == ForumThreadsByStart {
		sql = qForumThreadsByStart
	}
	return d.scanThreads(ctx, sql, args...)
}

func (d *DB) ForumCommentThreads(ctx context.Context, sort ForumThreadSort, offset, limit int) ([]ForumThread, error) {
	sql := qForumCommentThreadsByReply
	if sort == ForumThreadsByStart {
		sql = qForumCommentThreadsByStart
	}
	return d.scanThreads(ctx, sql, offset, limit)
}

func (d *DB) scanThreads(ctx context.Context, sql string, args ...any) ([]ForumThread, error) {
	rows, err := d.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query forum threads: %w", err)
	}
	defer rows.Close()

	var out []ForumThread
	for rows.Next() {
		var t ForumThread
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.CategoryID, &t.ArticleID,
			&t.AuthorID, &t.IsPinned, &t.IsLocked, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan forum thread: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

var qForumThreadPostCount = register("ForumThreadPostCount", `
SELECT count(*) FROM web_forumpost WHERE thread_id = $1`)

func (d *DB) ForumThreadPostCount(ctx context.Context, threadID int64) (int, error) {
	var n int
	if err := d.pool.QueryRow(ctx, qForumThreadPostCount, threadID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count posts in thread %d: %w", threadID, err)
	}
	return n, nil
}

var qForumThreadLastPost = register("ForumThreadLastPost", `
SELECT id, created_at, author_id
FROM web_forumpost
WHERE thread_id = $1
ORDER BY created_at
OFFSET $2 LIMIT 1`)

type ForumPost struct {
	ID        int64
	CreatedAt time.Time
	AuthorID  *int64
}

func (d *DB) ForumThreadLastPost(ctx context.Context, threadID int64, count int) (*ForumPost, error) {
	var p ForumPost
	err := d.pool.QueryRow(ctx, qForumThreadLastPost, threadID, count-1).Scan(&p.ID, &p.CreatedAt, &p.AuthorID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query last post in thread %d: %w", threadID, err)
	}
	return &p, nil
}

var qForumCategory = register("ForumCategory", `
SELECT id, section_id, name, description, is_for_comments
FROM web_forumcategory
WHERE id = $1`)

func (d *DB) ForumCategory(ctx context.Context, id int64) (*ForumCategory, error) {
	var c ForumCategory
	err := d.pool.QueryRow(ctx, qForumCategory, id).Scan(
		&c.ID, &c.SectionID, &c.Name, &c.Description, &c.IsForComments)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query forum category %d: %w", id, err)
	}
	return &c, nil
}

var qForumThread = register("ForumThread", `
SELECT `+forumThreadColumns+`
FROM web_forumthread
WHERE id = $1`)

func (d *DB) ForumThread(ctx context.Context, id int64) (*ForumThread, error) {
	var t ForumThread
	err := d.pool.QueryRow(ctx, qForumThread, id).Scan(&t.ID, &t.Name, &t.Description,
		&t.CategoryID, &t.ArticleID, &t.AuthorID, &t.IsPinned, &t.IsLocked, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query forum thread %d: %w", id, err)
	}
	return &t, nil
}

type ForumThreadPost struct {
	ID        int64
	ThreadID  int64
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
	AuthorID  *int64
	ReplyToID *int64
}

const forumPostColumns = `id, thread_id, name, created_at, updated_at, author_id, reply_to_id`

var qForumRootPostCount = register("ForumRootPostCount", `
SELECT count(*) FROM web_forumpost WHERE thread_id = $1 AND reply_to_id IS NULL`)

func (d *DB) ForumRootPostCount(ctx context.Context, threadID int64) (int, error) {
	var n int
	if err := d.pool.QueryRow(ctx, qForumRootPostCount, threadID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count root posts in thread %d: %w", threadID, err)
	}
	return n, nil
}

var qForumRootPosts = register("ForumRootPosts", `
SELECT `+forumPostColumns+`
FROM web_forumpost
WHERE thread_id = $1 AND reply_to_id IS NULL
ORDER BY created_at
OFFSET $2 LIMIT $3`)

func (d *DB) ForumRootPosts(ctx context.Context, threadID int64, offset, limit int) ([]ForumThreadPost, error) {
	return d.scanPosts(ctx, qForumRootPosts, threadID, offset, limit)
}

var qForumRootPostIDs = register("ForumRootPostIDs", `
SELECT id
FROM web_forumpost
WHERE thread_id = $1 AND reply_to_id IS NULL
ORDER BY created_at`)

func (d *DB) ForumRootPostIDs(ctx context.Context, threadID int64) ([]int64, error) {
	rows, err := d.pool.Query(ctx, qForumRootPostIDs, threadID)
	if err != nil {
		return nil, fmt.Errorf("query root post ids of thread %d: %w", threadID, err)
	}
	defer rows.Close()

	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan root post id: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

var qForumPostReplies = register("ForumPostReplies", `
SELECT `+forumPostColumns+`
FROM web_forumpost
WHERE reply_to_id = $1
ORDER BY created_at`)

func (d *DB) ForumPostReplies(ctx context.Context, postID int64) ([]ForumThreadPost, error) {
	return d.scanPosts(ctx, qForumPostReplies, postID)
}

var qForumPost = register("ForumPost", `
SELECT `+forumPostColumns+`
FROM web_forumpost
WHERE id = $1`)

func (d *DB) ForumPost(ctx context.Context, id int64) (*ForumThreadPost, error) {
	var p ForumThreadPost
	err := d.pool.QueryRow(ctx, qForumPost, id).Scan(&p.ID, &p.ThreadID, &p.Name,
		&p.CreatedAt, &p.UpdatedAt, &p.AuthorID, &p.ReplyToID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query forum post %d: %w", id, err)
	}
	return &p, nil
}

func (d *DB) scanPosts(ctx context.Context, sql string, args ...any) ([]ForumThreadPost, error) {
	rows, err := d.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query forum posts: %w", err)
	}
	defer rows.Close()

	var out []ForumThreadPost
	for rows.Next() {
		var p ForumThreadPost
		if err := rows.Scan(&p.ID, &p.ThreadID, &p.Name, &p.CreatedAt, &p.UpdatedAt,
			&p.AuthorID, &p.ReplyToID); err != nil {
			return nil, fmt.Errorf("scan forum post: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

type ForumPostContent struct {
	Source   string
	AuthorID *int64
}

var qForumPostContents = register("ForumPostContents", `
SELECT DISTINCT ON (post_id) post_id, source, author_id
FROM web_forumpostversion
WHERE post_id = ANY($1)
ORDER BY post_id, created_at DESC`)

func (d *DB) ForumPostContents(ctx context.Context, postIDs []int64) (map[int64]ForumPostContent, error) {
	rows, err := d.pool.Query(ctx, qForumPostContents, postIDs)
	if err != nil {
		return nil, fmt.Errorf("query forum post contents: %w", err)
	}
	defer rows.Close()

	out := make(map[int64]ForumPostContent, len(postIDs))
	for rows.Next() {
		var id int64
		var content ForumPostContent
		if err := rows.Scan(&id, &content.Source, &content.AuthorID); err != nil {
			return nil, fmt.Errorf("scan forum post content: %w", err)
		}
		out[id] = content
	}
	return out, rows.Err()
}

type RecentPost struct {
	ID        int64
	Name      string
	CreatedAt time.Time
	AuthorID  *int64

	ThreadID         int64
	ThreadName       string
	ThreadCategoryID *int64
	ThreadArticleID  *int64
	ThreadAuthorID   *int64
}

// An empty category list with comments switched off matches nothing, which is
// what a reader who may see no category is meant to get.
const recentPostFilter = `
FROM web_forumpost p
JOIN web_forumthread t ON t.id = p.thread_id
WHERE t.category_id = ANY($1) OR ($2::boolean AND t.article_id IS NOT NULL)`

var qRecentPostCount = register("RecentPostCount", `SELECT count(*)`+recentPostFilter)

func (d *DB) RecentPostCount(ctx context.Context, categoryIDs []int64, comments bool) (int, error) {
	var n int
	if err := d.pool.QueryRow(ctx, qRecentPostCount, categoryIDs, comments).Scan(&n); err != nil {
		return 0, fmt.Errorf("count recent posts: %w", err)
	}
	return n, nil
}

var qRecentPosts = register("RecentPosts", `
SELECT p.id, p.name, p.created_at, p.author_id,
       t.id, t.name, t.category_id, t.article_id, t.author_id`+recentPostFilter+`
ORDER BY p.created_at DESC
OFFSET $3 LIMIT $4`)

func (d *DB) RecentPosts(ctx context.Context, categoryIDs []int64, comments bool, offset, limit int) ([]RecentPost, error) {
	rows, err := d.pool.Query(ctx, qRecentPosts, categoryIDs, comments, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("query recent posts: %w", err)
	}
	defer rows.Close()

	var out []RecentPost
	for rows.Next() {
		var p RecentPost
		if err := rows.Scan(&p.ID, &p.Name, &p.CreatedAt, &p.AuthorID,
			&p.ThreadID, &p.ThreadName, &p.ThreadCategoryID, &p.ThreadArticleID,
			&p.ThreadAuthorID); err != nil {
			return nil, fmt.Errorf("scan recent post: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
