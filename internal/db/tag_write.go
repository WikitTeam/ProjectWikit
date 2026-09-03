package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const LogTags = "tags"

const defaultTagCategory = "_default"

type taggedName struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

var (
	qFindTagCategory = register("FindTagCategory", `
SELECT id FROM web_tagscategory WHERE slug = $1`)

	qInsertTagCategory = register("InsertTagCategory", `
INSERT INTO web_tagscategory (name, description, slug)
VALUES ($1, '', $1)
RETURNING id`)

	qFindTag = register("FindTag", `
SELECT id FROM web_tag WHERE category_id = $1 AND name = $2`)

	qInsertTag = register("InsertTag", `
INSERT INTO web_tag (category_id, name) VALUES ($1, $2) RETURNING id`)

	qReadArticleTags = register("ReadArticleTags", `
SELECT t.id, c.slug, t.name
FROM web_article_tags at
JOIN web_tag t ON t.id = at.tag_id
JOIN web_tagscategory c ON c.id = t.category_id
WHERE at.article_id = $1
ORDER BY t.id`)

	qDropArticleTags = register("DropArticleTags", `
DELETE FROM web_article_tags WHERE article_id = $1 AND NOT (tag_id = ANY($2))`)

	qInsertArticleTag = register("InsertArticleTag", `
INSERT INTO web_article_tags (article_id, tag_id) VALUES ($1, $2)`)

	qDropOrphanTags = register("DropOrphanTags", `
DELETE FROM web_tag
WHERE NOT EXISTS (SELECT 1 FROM web_article_tags at WHERE at.tag_id = web_tag.id)`)

	// Only a category whose name was never set apart from its slug is swept up,
	// which is how one somebody typed out survives losing its last tag.
	qDropOrphanTagCategories = register("DropOrphanTagCategories", `
DELETE FROM web_tagscategory
WHERE slug = name
  AND NOT EXISTS (SELECT 1 FROM web_tag t WHERE t.category_id = web_tagscategory.id)`)
)

// A name with a space in it is dropped rather than refused, so one bad entry
// does not cost the page the rest of its tags.
func (d *DB) SetArticleTags(ctx context.Context, articleID int64, tags []string,
	allowCreate bool, userID *int64, at time.Time) (int, bool, error) {

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return 0, false, fmt.Errorf("begin tags of %d: %w", articleID, err)
	}
	defer tx.Rollback(ctx)

	wanted, err := resolveTags(ctx, tx, tags, allowCreate)
	if err != nil {
		return 0, false, fmt.Errorf("resolve tags of %d: %w", articleID, err)
	}
	held, err := readArticleTags(ctx, tx, articleID)
	if err != nil {
		return 0, false, fmt.Errorf("read tags of %d: %w", articleID, err)
	}

	added := tagsMissingFrom(wanted, held)
	removed := tagsMissingFrom(held, wanted)
	if len(added) == 0 && len(removed) == 0 {
		return 0, false, nil
	}

	keep := make([]int64, 0, len(wanted))
	for _, tag := range wanted {
		keep = append(keep, tag.ID)
	}
	if _, err := tx.Exec(ctx, qDropArticleTags, articleID, keep); err != nil {
		return 0, false, fmt.Errorf("drop tags of %d: %w", articleID, err)
	}
	for _, tag := range added {
		if _, err := tx.Exec(ctx, qInsertArticleTag, articleID, tag.ID); err != nil {
			return 0, false, fmt.Errorf("tag %d: %w", articleID, err)
		}
	}
	if allowCreate {
		if _, err := tx.Exec(ctx, qDropOrphanTags); err != nil {
			return 0, false, fmt.Errorf("sweep tags: %w", err)
		}
		if _, err := tx.Exec(ctx, qDropOrphanTagCategories); err != nil {
			return 0, false, fmt.Errorf("sweep tag categories: %w", err)
		}
	}

	meta, err := json.Marshal(map[string]any{"added_tags": added, "removed_tags": removed})
	if err != nil {
		return 0, false, fmt.Errorf("encode tag meta of %d: %w", articleID, err)
	}
	if _, err := tx.Exec(ctx, qLockArticleLog, articleID); err != nil {
		return 0, false, fmt.Errorf("lock article log %d: %w", articleID, err)
	}
	var revNumber int
	if err := tx.QueryRow(ctx, qInsertArticleLog, articleID, userID, LogTags, string(meta), "", at).Scan(&revNumber); err != nil {
		return 0, false, fmt.Errorf("write revision of %d: %w", articleID, err)
	}
	if _, err := tx.Exec(ctx, qTouchArticle, articleID, at); err != nil {
		return 0, false, fmt.Errorf("touch article %d: %w", articleID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, false, fmt.Errorf("commit tags of %d: %w", articleID, err)
	}
	return revNumber, true, nil
}

func resolveTags(ctx context.Context, tx pgx.Tx, tags []string, allowCreate bool) ([]taggedName, error) {
	var out []taggedName
	for _, raw := range tags {
		if strings.Contains(raw, " ") {
			continue
		}
		category, name := splitTagName(strings.ToLower(raw))
		id, err := tagID(ctx, tx, category, name, allowCreate)
		if err != nil {
			return nil, err
		}
		if id == 0 {
			continue
		}
		tag := taggedName{ID: id, Name: tagFullName(category, name)}
		if !slices.ContainsFunc(out, func(other taggedName) bool { return other.ID == tag.ID }) {
			out = append(out, tag)
		}
	}
	return out, nil
}

func tagID(ctx context.Context, tx pgx.Tx, category, name string, allowCreate bool) (int64, error) {
	var categoryID int64
	err := tx.QueryRow(ctx, qFindTagCategory, category).Scan(&categoryID)
	if errors.Is(err, pgx.ErrNoRows) {
		if !allowCreate {
			return 0, nil
		}
		if err := tx.QueryRow(ctx, qInsertTagCategory, category).Scan(&categoryID); err != nil {
			return 0, err
		}
	} else if err != nil {
		return 0, err
	}

	var id int64
	err = tx.QueryRow(ctx, qFindTag, categoryID, name).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		if !allowCreate {
			return 0, nil
		}
		if err := tx.QueryRow(ctx, qInsertTag, categoryID, name).Scan(&id); err != nil {
			return 0, err
		}
	} else if err != nil {
		return 0, err
	}
	return id, nil
}

func readArticleTags(ctx context.Context, tx pgx.Tx, articleID int64) ([]taggedName, error) {
	rows, err := tx.Query(ctx, qReadArticleTags, articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []taggedName
	for rows.Next() {
		var id int64
		var category, name string
		if err := rows.Scan(&id, &category, &name); err != nil {
			return nil, err
		}
		out = append(out, taggedName{ID: id, Name: tagFullName(category, name)})
	}
	return out, rows.Err()
}

func splitTagName(full string) (category, name string) {
	if c, n, ok := strings.Cut(full, ":"); ok {
		return c, n
	}
	return defaultTagCategory, full
}

func tagFullName(category, name string) string {
	if category == defaultTagCategory {
		return name
	}
	return category + ":" + name
}

func tagsMissingFrom(want, have []taggedName) []taggedName {
	out := []taggedName{}
	for _, tag := range want {
		if !slices.ContainsFunc(have, func(other taggedName) bool { return other.ID == tag.ID }) {
			out = append(out, tag)
		}
	}
	return out
}
