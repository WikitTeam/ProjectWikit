package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type Article struct {
	ID        int64
	Category  string
	Name      string
	Title     string
	ParentID  *int64
	Locked    bool
	CreatedAt time.Time
	UpdatedAt time.Time
	MediaName string
}

// FullName drops the _default category, which exists in the column but never in
// a URL or a page reference.
func (a *Article) FullName() string {
	if a.Category != DefaultCategory {
		return a.Category + ":" + a.Name
	}
	return a.Name
}

// DisplayName is what a breadcrumb or a link label shows when the page has no
// title of its own.
func (a *Article) DisplayName() string {
	if title := strings.TrimSpace(a.Title); title != "" {
		return title
	}
	return a.FullName()
}

const DefaultCategory = "_default"

const articleColumns = `id, category, name, title, parent_id, locked, created_at, updated_at, media_name`

const prefixedArticleColumns = `a.id, a.category, a.name, a.title, a.parent_id, a.locked, a.created_at, a.updated_at, a.media_name`

var qArticleByName = register("ArticleByName", `
SELECT `+articleColumns+`
FROM web_article
WHERE complete_full_name = $1`)

// ArticleByName takes a page reference the way a URL spells it; dumbName puts
// the implicit category back so the generated column can match.
func (d *DB) ArticleByName(ctx context.Context, ref string) (*Article, error) {
	var a Article
	err := d.pool.QueryRow(ctx, qArticleByName, dumbName(ref)).Scan(
		&a.ID, &a.Category, &a.Name, &a.Title, &a.ParentID, &a.Locked,
		&a.CreatedAt, &a.UpdatedAt, &a.MediaName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lookup article %q: %w", ref, err)
	}
	return &a, nil
}
