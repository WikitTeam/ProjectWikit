package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

const (
	CreateTagsDefault  = "default"
	CreateTagsDisabled = "disabled"
	CreateTagsEnabled  = "enabled"
)

var qSiteCanCreateTags = register("SiteCanCreateTags", `
SELECT can_user_create_tags
FROM web_settings
WHERE site_id = $1`)

// SiteCanCreateTags reads the site row alone. The page asks Site.settings,
// which never merges the category or the built-in defaults into it.
func (d *DB) SiteCanCreateTags(ctx context.Context, siteID int64) (string, error) {
	var mode string
	err := d.pool.QueryRow(ctx, qSiteCanCreateTags, siteID).Scan(&mode)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("query tag setting of site %d: %w", siteID, err)
	}
	return mode, nil
}

type CommentInfo struct {
	ThreadID int64
	Count    int
}

var qCommentInfo = register("CommentInfo", `
SELECT t.id, (SELECT count(*) FROM web_forumpost p WHERE p.thread_id = t.id)
FROM web_forumthread t
WHERE t.article_id = $1`)

// A page whose thread has never been created reports the zero value rather than
// having one written for it on a read.
func (d *DB) CommentInfo(ctx context.Context, articleID int64) (CommentInfo, error) {
	var info CommentInfo
	err := d.pool.QueryRow(ctx, qCommentInfo, articleID).Scan(&info.ThreadID, &info.Count)
	if errors.Is(err, pgx.ErrNoRows) {
		return CommentInfo{}, nil
	}
	if err != nil {
		return CommentInfo{}, fmt.Errorf("query comment thread of article %d: %w", articleID, err)
	}
	return info, nil
}

var qSubscribedToArticle = register("SubscribedToArticle", `
SELECT EXISTS(
    SELECT 1 FROM web_usernotificationsubscription
    WHERE subscriber_id = $1 AND article_id = $2 AND forum_thread_id IS NULL)`)

func (d *DB) SubscribedToArticle(ctx context.Context, userID, articleID int64) (bool, error) {
	var yes bool
	if err := d.pool.QueryRow(ctx, qSubscribedToArticle, userID, articleID).Scan(&yes); err != nil {
		return false, fmt.Errorf("check article subscription of user %d: %w", userID, err)
	}
	return yes, nil
}

var qSubscribedToThread = register("SubscribedToThread", `
SELECT EXISTS(
    SELECT 1 FROM web_usernotificationsubscription
    WHERE subscriber_id = $1 AND forum_thread_id = $2 AND article_id IS NULL)`)

func (d *DB) SubscribedToThread(ctx context.Context, userID, threadID int64) (bool, error) {
	var yes bool
	if err := d.pool.QueryRow(ctx, qSubscribedToThread, userID, threadID).Scan(&yes); err != nil {
		return false, fmt.Errorf("check thread subscription of user %d: %w", userID, err)
	}
	return yes, nil
}

var qUserPreference = register("UserPreference", `
SELECT raw_value
FROM dynamic_preferences_users_userpreferencemodel
WHERE instance_id = $1 AND section = $2 AND name = $3`)

func (d *DB) UserPreference(ctx context.Context, userID int64, section, name string) (string, error) {
	var raw string
	err := d.pool.QueryRow(ctx, qUserPreference, userID, section, name).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("query preference %s.%s of user %d: %w", section, name, userID, err)
	}
	return raw, nil
}

var qTagCategoryBySlug = register("TagCategoryBySlug", `
SELECT id, name, priority
FROM web_tagscategory
WHERE slug = $1`)

func (d *DB) TagCategoryBySlug(ctx context.Context, slug string) (TagCategory, error) {
	var c TagCategory
	err := d.pool.QueryRow(ctx, qTagCategoryBySlug, slug).Scan(&c.ID, &c.Name, &c.Priority)
	if errors.Is(err, pgx.ErrNoRows) {
		return TagCategory{}, ErrNotFound
	}
	if err != nil {
		return TagCategory{}, fmt.Errorf("lookup tag category %q: %w", slug, err)
	}
	return c, nil
}

var qCategoryNames = register("CategoryNames", `SELECT name FROM web_category`)

func (d *DB) CategoryNames(ctx context.Context) ([]string, error) {
	rows, err := d.pool.Query(ctx, qCategoryNames)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan category name: %w", err)
		}
		out = append(out, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read categories: %w", err)
	}
	return out, nil
}

var qArticlesByTag = register("ArticlesByTag", `
SELECT `+prefixedArticleColumns+`
FROM web_article a
JOIN web_article_tags link ON link.article_id = a.id
JOIN web_tag t ON t.id = link.tag_id
JOIN web_tagscategory c ON c.id = t.category_id
WHERE c.slug = $1 AND t.name = $2 AND NOT (a.category = ANY($3))
ORDER BY a.title`)

// ArticlesByTag leaves out the categories the visitor cannot see, which the
// caller resolves because permissions are not a database question.
func (d *DB) ArticlesByTag(ctx context.Context, categorySlug, name string, hidden []string) ([]Article, error) {
	if hidden == nil {
		hidden = []string{}
	}
	rows, err := d.pool.Query(ctx, qArticlesByTag, categorySlug, name, hidden)
	if err != nil {
		return nil, fmt.Errorf("query articles tagged %q: %w", name, err)
	}
	defer rows.Close()

	var out []Article
	for rows.Next() {
		var a Article
		if err := rows.Scan(&a.ID, &a.Category, &a.Name, &a.Title, &a.ParentID,
			&a.Locked, &a.CreatedAt, &a.UpdatedAt, &a.MediaName); err != nil {
			return nil, fmt.Errorf("scan article tagged %q: %w", name, err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read articles tagged %q: %w", name, err)
	}
	return out, nil
}
