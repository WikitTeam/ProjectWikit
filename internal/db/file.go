package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type ArticleFile struct {
	ArticleMediaName string
	MediaName        string
	MimeType         string
	Size             int64
}

// f.name is a plain text column, so this comparison is case-sensitive; only
// complete_full_name is citext.
var qArticleFile = register("ArticleFile", `
SELECT a.media_name, f.media_name, f.mime_type, f.size
FROM web_file f
JOIN web_article a ON a.id = f.article_id
WHERE a.complete_full_name = $1 AND f.name = $2 AND f.deleted_at IS NULL
ORDER BY f.id
LIMIT 1`)

// ArticleFile resolves an attachment's on-disk names. ORDER BY id only makes
// an already-unique row deterministic.
func (d *DB) ArticleFile(ctx context.Context, articleRef, fileName string) (*ArticleFile, error) {
	var f ArticleFile
	err := d.pool.QueryRow(ctx, qArticleFile, dumbName(articleRef), fileName).Scan(
		&f.ArticleMediaName, &f.MediaName, &f.MimeType, &f.Size)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lookup file %q in article %q: %w", fileName, articleRef, err)
	}
	return &f, nil
}
