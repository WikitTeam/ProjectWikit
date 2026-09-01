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

type ArticleFileEntry struct {
	ID       int64
	Name     string
	MimeType string
	Size     int64
}

var qArticleFiles = register("ArticleFiles", `
SELECT f.id, f.name, f.mime_type, f.size
FROM web_file f
WHERE f.article_id = $1 AND f.deleted_at IS NULL
ORDER BY f.name, f.id`)

// The name is unique per article only among the rows still alive, so the id
// breaks the tie for a name that was deleted and uploaded again.
func (d *DB) ArticleFiles(ctx context.Context, articleID int64) ([]ArticleFileEntry, error) {
	rows, err := d.pool.Query(ctx, qArticleFiles, articleID)
	if err != nil {
		return nil, fmt.Errorf("list files of article %d: %w", articleID, err)
	}
	defer rows.Close()

	var out []ArticleFileEntry
	for rows.Next() {
		var f ArticleFileEntry
		if err := rows.Scan(&f.ID, &f.Name, &f.MimeType, &f.Size); err != nil {
			return nil, fmt.Errorf("scan file: %w", err)
		}
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list files of article %d: %w", articleID, err)
	}
	return out, nil
}
