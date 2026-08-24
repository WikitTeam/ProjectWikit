package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

var qArticleByID = register("ArticleByID", `
SELECT `+articleColumns+`
FROM web_article
WHERE id = $1`)

func (d *DB) ArticleByID(ctx context.Context, id int64) (*Article, error) {
	var a Article
	err := d.pool.QueryRow(ctx, qArticleByID, id).Scan(
		&a.ID, &a.Category, &a.Name, &a.Title, &a.ParentID, &a.Locked,
		&a.CreatedAt, &a.UpdatedAt, &a.MediaName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lookup article %d: %w", id, err)
	}
	return &a, nil
}

var qLatestSource = register("LatestSource", `
SELECT source
FROM web_articleversion
WHERE article_id = $1
ORDER BY created_at DESC
LIMIT 1`)

func (d *DB) LatestSource(ctx context.Context, articleID int64) (string, error) {
	var source string
	err := d.pool.QueryRow(ctx, qLatestSource, articleID).Scan(&source)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("query latest source of article %d: %w", articleID, err)
	}
	return source, nil
}

// The join table carries no ordering in Django, so the author order a page
// shows is whatever Postgres returns; ordering by the link row freezes it.
var qArticleAuthors = register("ArticleAuthors", `
SELECT `+prefixed("u", userColumns)+`
FROM web_article_authors link
JOIN web_user u ON u.id = link.user_id
WHERE link.article_id = $1
ORDER BY link.id`)

func (d *DB) ArticleAuthors(ctx context.Context, articleID int64) ([]User, error) {
	rows, err := d.pool.Query(ctx, qArticleAuthors, articleID)
	if err != nil {
		return nil, fmt.Errorf("query authors of article %d: %w", articleID, err)
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		dest, finish := userDest(&u)
		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("scan author: %w", err)
		}
		finish()
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read authors of article %d: %w", articleID, err)
	}
	return out, nil
}

var qLatestEditor = register("LatestEditor", `
SELECT `+prefixed("u", userColumns)+`
FROM web_articlelogentry e
JOIN web_user u ON u.id = e.user_id
WHERE e.article_id = $1
ORDER BY e.rev_number DESC
LIMIT 1`)

func (d *DB) LatestEditor(ctx context.Context, articleID int64) (*User, error) {
	var u User
	dest, finish := userDest(&u)
	err := d.pool.QueryRow(ctx, qLatestEditor, articleID).Scan(dest...)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query latest editor of article %d: %w", articleID, err)
	}
	finish()
	return &u, nil
}

var qRevisionCount = register("RevisionCount", `
SELECT count(*)
FROM web_articlelogentry
WHERE article_id = $1`)

func (d *DB) RevisionCount(ctx context.Context, articleID int64) (int, error) {
	var n int
	if err := d.pool.QueryRow(ctx, qRevisionCount, articleID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count revisions of article %d: %w", articleID, err)
	}
	return n, nil
}

var qArticleTags = register("ArticleTags", `
SELECT CASE WHEN c.slug = '_default' THEN t.name ELSE c.slug || ':' || t.name END
FROM web_article_tags link
JOIN web_tag t ON t.id = link.tag_id
JOIN web_tagscategory c ON c.id = t.category_id
WHERE link.article_id = $1`)

// ArticleTags returns full names unsorted; the caller lowercases and sorts,
// because that pass belongs to whoever renders them.
func (d *DB) ArticleTags(ctx context.Context, articleID int64) ([]string, error) {
	rows, err := d.pool.Query(ctx, qArticleTags, articleID)
	if err != nil {
		return nil, fmt.Errorf("query tags of article %d: %w", articleID, err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		out = append(out, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read tags of article %d: %w", articleID, err)
	}
	return out, nil
}
