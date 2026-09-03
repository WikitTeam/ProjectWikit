package db

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"
)

const (
	LogNew    = "new"
	LogSource = "source"
	LogParent = "parent"
)

type Revision struct {
	VersionID int64
	RevNumber int
}

type VersionWrite struct {
	ArticleID int64
	Source    string
	UserID    *int64
	Kind      string
	Comment   string
	At        time.Time

	// Title rides along on the revision that created the page and nowhere else,
	// so an empty one still has to be written.
	Title string
}

var qInsertArticleVersion = register("InsertArticleVersion", `
INSERT INTO web_articleversion (article_id, source, created_at)
VALUES ($1, $2, $3)
RETURNING id`)

// The version and the revision naming it go in together, so a failed write
// cannot leave behind a version nothing points at.
func (d *DB) CreateArticleVersion(ctx context.Context, w VersionWrite) (Revision, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return Revision{}, fmt.Errorf("begin version: %w", err)
	}
	defer tx.Rollback(ctx)

	var rev Revision
	if err := tx.QueryRow(ctx, qInsertArticleVersion, w.ArticleID, w.Source, w.At).Scan(&rev.VersionID); err != nil {
		return Revision{}, fmt.Errorf("write version of %d: %w", w.ArticleID, err)
	}

	meta, err := versionMeta(rev.VersionID, w)
	if err != nil {
		return Revision{}, err
	}
	if _, err := tx.Exec(ctx, qLockArticleLog, w.ArticleID); err != nil {
		return Revision{}, fmt.Errorf("lock article log %d: %w", w.ArticleID, err)
	}
	if err := tx.QueryRow(ctx, qInsertArticleLog, w.ArticleID, w.UserID, w.Kind, meta,
		w.Comment, w.At).Scan(&rev.RevNumber); err != nil {
		return Revision{}, fmt.Errorf("write revision of %d: %w", w.ArticleID, err)
	}
	if _, err := tx.Exec(ctx, qTouchArticle, w.ArticleID, w.At); err != nil {
		return Revision{}, fmt.Errorf("touch article %d: %w", w.ArticleID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Revision{}, fmt.Errorf("commit version of %d: %w", w.ArticleID, err)
	}
	return rev, nil
}

func versionMeta(versionID int64, w VersionWrite) (string, error) {
	fields := map[string]any{"version_id": versionID}
	if w.Kind == LogNew {
		fields["title"] = w.Title
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		return "", fmt.Errorf("encode revision meta of %d: %w", w.ArticleID, err)
	}
	return string(encoded), nil
}

type ArticleLink struct {
	To   string
	Kind string
}

var (
	qDropArticleLinks = register("DropArticleLinks", `
DELETE FROM web_externallink WHERE link_from = $1`)

	qInsertArticleLinks = register("InsertArticleLinks", `
INSERT INTO web_externallink (link_from, link_type, link_to)
SELECT $1, kind, target
FROM unnest($2::text[], $3::text[]) AS t(kind, target)
ON CONFLICT DO NOTHING`)
)

// The whole set is replaced under one transaction, so nobody reads a page as
// having no links at all while it is being saved.
func (d *DB) ReplaceArticleLinks(ctx context.Context, from string, links []ArticleLink) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin links of %q: %w", from, err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, qDropArticleLinks, from); err != nil {
		return fmt.Errorf("drop links of %q: %w", from, err)
	}
	if len(links) > 0 {
		kinds := make([]string, 0, len(links))
		targets := make([]string, 0, len(links))
		for _, link := range links {
			kinds = append(kinds, link.Kind)
			targets = append(targets, link.To)
		}
		if _, err := tx.Exec(ctx, qInsertArticleLinks, from, kinds, targets); err != nil {
			return fmt.Errorf("write links of %q: %w", from, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit links of %q: %w", from, err)
	}
	return nil
}

var (
	qInsertArticle = register("InsertArticle", `
INSERT INTO web_article (category, name, title, locked, created_at, updated_at, media_name)
VALUES ($1, $2, $3, false, $4, $4, $5)
RETURNING id`)

	qInsertArticleAuthor = register("InsertArticleAuthor", `
INSERT INTO web_article_authors (article_id, user_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING`)
)

func (d *DB) CreateArticle(ctx context.Context, category, name, title string, authorID *int64, at time.Time) (int64, error) {
	media, err := mediaName()
	if err != nil {
		return 0, err
	}

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin article %q: %w", name, err)
	}
	defer tx.Rollback(ctx)

	var id int64
	if err := tx.QueryRow(ctx, qInsertArticle, category, name, title, at, media).Scan(&id); err != nil {
		return 0, fmt.Errorf("write article %q: %w", name, err)
	}
	if authorID != nil {
		if _, err := tx.Exec(ctx, qInsertArticleAuthor, id, *authorID); err != nil {
			return 0, fmt.Errorf("credit author of %q: %w", name, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit article %q: %w", name, err)
	}
	return id, nil
}

// The directory a page's files live in is named after the row rather than after
// the page, so renaming a page moves nothing on disk.
func mediaName() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("pick a media name: %w", err)
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

var qSetArticleParent = register("SetArticleParent", `
UPDATE web_article SET parent_id = $2 WHERE id = $1`)

func (d *DB) SetArticleParent(ctx context.Context, articleID int64, parentID *int64,
	userID *int64, meta string, at time.Time) (int, error) {

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin parent of %d: %w", articleID, err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, qSetArticleParent, articleID, parentID); err != nil {
		return 0, fmt.Errorf("set parent of %d: %w", articleID, err)
	}
	if _, err := tx.Exec(ctx, qLockArticleLog, articleID); err != nil {
		return 0, fmt.Errorf("lock article log %d: %w", articleID, err)
	}
	var revNumber int
	if err := tx.QueryRow(ctx, qInsertArticleLog, articleID, userID, LogParent, meta, "", at).Scan(&revNumber); err != nil {
		return 0, fmt.Errorf("write revision of %d: %w", articleID, err)
	}
	if _, err := tx.Exec(ctx, qTouchArticle, articleID, at); err != nil {
		return 0, fmt.Errorf("touch article %d: %w", articleID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit parent of %d: %w", articleID, err)
	}
	return revNumber, nil
}

// Nothing stops a second row for the same pair, so the check rides inside the
// insert rather than following it.
var qSubscribeToArticle = register("SubscribeToArticle", `
INSERT INTO web_usernotificationsubscription (subscriber_id, article_id, forum_thread_id)
SELECT $1, $2, NULL
WHERE NOT EXISTS (SELECT 1 FROM web_usernotificationsubscription
                  WHERE subscriber_id = $1 AND article_id = $2 AND forum_thread_id IS NULL)`)

func (d *DB) SubscribeToArticle(ctx context.Context, userID, articleID int64) error {
	if _, err := d.pool.Exec(ctx, qSubscribeToArticle, userID, articleID); err != nil {
		return fmt.Errorf("subscribe %d to article %d: %w", userID, articleID, err)
	}
	return nil
}
