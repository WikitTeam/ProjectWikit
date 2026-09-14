package db

import (
	"context"
	"fmt"
)

var (
	// Two configurations are stacked because a page can hold either language and
	// the column is the only place the search reads from.
	qUpdateSearchIndex = register("UpdateSearchIndex", `
UPDATE web_articlesearchindex
SET content_source = $2,
    content_plaintext = $3,
    vector_plaintext = to_tsvector('english', $3) || to_tsvector('russian', $3)
WHERE article_id = $1`)

	qInsertSearchIndex = register("InsertSearchIndex", `
INSERT INTO web_articlesearchindex (article_id, content_source, content_plaintext, vector_plaintext)
SELECT $1, $2, $3, to_tsvector('english', $3) || to_tsvector('russian', $3)
WHERE NOT EXISTS (SELECT 1 FROM web_articlesearchindex WHERE article_id = $1)`)
)

func (d *DB) UpdateSearchIndex(ctx context.Context, articleID int64, source, plaintext string) error {
	tag, err := d.pool.Exec(ctx, qUpdateSearchIndex, articleID, source, plaintext)
	if err != nil {
		return fmt.Errorf("update search index of %d: %w", articleID, err)
	}
	if tag.RowsAffected() > 0 {
		return nil
	}
	if _, err := d.pool.Exec(ctx, qInsertSearchIndex, articleID, source, plaintext); err != nil {
		return fmt.Errorf("write search index of %d: %w", articleID, err)
	}
	return nil
}
