package db

import (
	"context"
	"fmt"
	"time"
)

type LogEntry struct {
	RevNumber int
	Type      string
	Meta      []byte
	Comment   string
	CreatedAt time.Time
	UserID    *int64
}

var (
	// A null limit asks for the whole history, which is what the caller wants
	// when it is not paging.
	qArticleLog = register("ArticleLog", `
SELECT rev_number, type, meta, comment, created_at, user_id
FROM web_articlelogentry
WHERE article_id = $1
ORDER BY rev_number DESC
OFFSET $2 LIMIT $3`)

	qArticleLogCount = register("ArticleLogCount", `
SELECT count(*) FROM web_articlelogentry WHERE article_id = $1`)
)

func (d *DB) ArticleLog(ctx context.Context, articleID int64, offset int, limit *int) ([]LogEntry, error) {
	rows, err := d.pool.Query(ctx, qArticleLog, articleID, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("query log of article %d: %w", articleID, err)
	}
	defer rows.Close()

	var out []LogEntry
	for rows.Next() {
		var e LogEntry
		if err := rows.Scan(&e.RevNumber, &e.Type, &e.Meta, &e.Comment, &e.CreatedAt, &e.UserID); err != nil {
			return nil, fmt.Errorf("scan log entry of article %d: %w", articleID, err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (d *DB) ArticleLogCount(ctx context.Context, articleID int64) (int, error) {
	var n int
	if err := d.pool.QueryRow(ctx, qArticleLogCount, articleID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count log of article %d: %w", articleID, err)
	}
	return n, nil
}
