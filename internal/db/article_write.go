package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

const (
	LogNew    = "new"
	LogSource = "source"
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
