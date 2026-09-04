package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type FileRecord struct {
	ID        int64
	Name      string
	MimeType  string
	Size      int64
	CreatedAt time.Time
	AuthorID  *int64
}

// Newest last. Ordering by name instead would reshuffle the whole list every
// time one file is renamed.
var qArticleFileList = register("ArticleFileList", `
SELECT id, name, mime_type, size, created_at, author_id
FROM web_file
WHERE article_id = $1 AND deleted_at IS NULL
ORDER BY id`)

func (d *DB) ArticleFileList(ctx context.Context, articleID int64) ([]FileRecord, error) {
	rows, err := d.pool.Query(ctx, qArticleFileList, articleID)
	if err != nil {
		return nil, fmt.Errorf("list files of article %d: %w", articleID, err)
	}
	defer rows.Close()

	var out []FileRecord
	for rows.Next() {
		var f FileRecord
		if err := rows.Scan(&f.ID, &f.Name, &f.MimeType, &f.Size, &f.CreatedAt, &f.AuthorID); err != nil {
			return nil, fmt.Errorf("scan file: %w", err)
		}
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list files of article %d: %w", articleID, err)
	}
	return out, nil
}

// A deleted row still occupies the disk, so the two totals differ by exactly
// what a purge would free.
var qFileSpaceUsage = register("FileSpaceUsage", `
SELECT COALESCE(SUM(size) FILTER (WHERE deleted_at IS NULL), 0), COALESCE(SUM(size), 0)
FROM web_file`)

func (d *DB) FileSpaceUsage(ctx context.Context) (live, total int64, err error) {
	if err := d.pool.QueryRow(ctx, qFileSpaceUsage).Scan(&live, &total); err != nil {
		return 0, 0, fmt.Errorf("sum file sizes: %w", err)
	}
	return live, total, nil
}

type FileRow struct {
	ID        int64
	ArticleID int64
	Name      string
	MediaName string
	Deleted   bool
}

var qFileByID = register("FileByID", `
SELECT id, article_id, name, media_name, deleted_at IS NOT NULL
FROM web_file
WHERE id = $1`)

func (d *DB) FileByID(ctx context.Context, fileID int64) (*FileRow, error) {
	var f FileRow
	err := d.pool.QueryRow(ctx, qFileByID, fileID).Scan(&f.ID, &f.ArticleID, &f.Name, &f.MediaName, &f.Deleted)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lookup file %d: %w", fileID, err)
	}
	return &f, nil
}

var qLiveFileNamed = register("LiveFileNamed", `
SELECT id
FROM web_file
WHERE article_id = $1 AND name = $2 AND deleted_at IS NULL
ORDER BY id
LIMIT 1`)

func (d *DB) LiveFileNamed(ctx context.Context, articleID int64, name string) (int64, error) {
	var id int64
	err := d.pool.QueryRow(ctx, qLiveFileNamed, articleID, name).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("lookup file %q in article %d: %w", name, articleID, err)
	}
	return id, nil
}

type FileWrite struct {
	ArticleID int64
	Name      string
	MediaName string
	MimeType  string
	Size      int64
	AuthorID  *int64
	At        time.Time
}

var qInsertFile = register("InsertFile", `
INSERT INTO web_file (article_id, name, media_name, mime_type, size, author_id, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id`)

func (d *DB) AddArticleFile(ctx context.Context, w FileWrite) (int64, error) {
	var id int64
	err := d.pool.QueryRow(ctx, qInsertFile, w.ArticleID, w.Name, w.MediaName,
		w.MimeType, w.Size, w.AuthorID, w.At).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("write file %q of article %d: %w", w.Name, w.ArticleID, err)
	}
	return id, nil
}
