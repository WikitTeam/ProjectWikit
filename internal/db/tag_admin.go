package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type TagCategoryRow struct {
	ID          int64
	Name        string
	Slug        string
	Description string
	Priority    *int
	Tags        int
}

var qAdminTagCategories = register("AdminTagCategories", `
SELECT c.id, c.name, c.slug, c.description, c.priority,
       (SELECT count(*) FROM web_tag t WHERE t.category_id = c.id)
FROM web_tagscategory c WHERE c.site_id = $1 ORDER BY c.priority DESC, c.name, c.id`)

func (d *DB) AdminTagCategories(ctx context.Context, siteID int64) ([]TagCategoryRow, error) {
	rows, err := d.pool.Query(ctx, qAdminTagCategories, siteID)
	if err != nil {
		return nil, fmt.Errorf("list tag categories: %w", err)
	}
	defer rows.Close()

	var out []TagCategoryRow
	for rows.Next() {
		var c TagCategoryRow
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.Priority, &c.Tags); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

var qAdminTagCategory = register("AdminTagCategory", `
SELECT id, name, slug, description, priority FROM web_tagscategory WHERE id = $1 AND site_id = $2`)

func (d *DB) AdminTagCategory(ctx context.Context, siteID, id int64) (TagCategoryRow, error) {
	var c TagCategoryRow
	err := d.pool.QueryRow(ctx, qAdminTagCategory, id, siteID).
		Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.Priority)
	if errors.Is(err, pgx.ErrNoRows) {
		return TagCategoryRow{}, ErrNotFound
	}
	if err != nil {
		return TagCategoryRow{}, fmt.Errorf("read tag category %d: %w", id, err)
	}
	return c, nil
}

var (
	qInsertTagCat = register("InsertTagCat", `
INSERT INTO web_tagscategory (name, slug, description, priority, site_id) VALUES ($1,$2,$3,$4,$5) RETURNING id`)
	qUpdateTagCat = register("UpdateTagCat", `
UPDATE web_tagscategory SET name=$2, slug=$3, description=$4, priority=$5 WHERE id=$1 AND site_id=$6`)
	qDetachTagCat = register("DetachTagCat", `UPDATE web_tag SET category_id = NULL WHERE category_id = $1`)
	qDeleteTagCat = register("DeleteTagCat", `DELETE FROM web_tagscategory WHERE id = $1 AND site_id = $2`)
)

func (d *DB) SaveTagCategory(ctx context.Context, siteID int64, c TagCategoryRow) error {
	if c.ID == 0 {
		var id int64
		err := d.pool.QueryRow(ctx, qInsertTagCat, c.Name, c.Slug, c.Description, c.Priority, siteID).Scan(&id)
		if err != nil {
			return fmt.Errorf("create tag category %q: %w", c.Slug, err)
		}
		return nil
	}
	_, err := d.pool.Exec(ctx, qUpdateTagCat, c.ID, c.Name, c.Slug, c.Description, c.Priority, siteID)
	if err != nil {
		return fmt.Errorf("update tag category %d: %w", c.ID, err)
	}
	return nil
}

func (d *DB) DeleteTagCategory(ctx context.Context, siteID, id int64) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin deleting tag category %d: %w", id, err)
	}
	defer tx.Rollback(context.WithoutCancel(ctx))

	if _, err := tx.Exec(ctx, qDetachTagCat, id); err != nil {
		return fmt.Errorf("detach tag category %d: %w", id, err)
	}
	if _, err := tx.Exec(ctx, qDeleteTagCat, id, siteID); err != nil {
		return fmt.Errorf("delete tag category %d: %w", id, err)
	}
	return tx.Commit(ctx)
}

type TagRow struct {
	ID           int64
	Name         string
	CategoryID   *int64
	CategoryName string
	Articles     int
	IsIndexed    bool
}

var qAdminTags = register("AdminTags", `
SELECT t.id, t.name, t.category_id, coalesce(c.name, ''),
       (SELECT count(*) FROM web_article_tags at WHERE at.tag_id = t.id)
FROM web_tag t LEFT JOIN web_tagscategory c ON c.id = t.category_id
WHERE ($1 = '' OR t.name ILIKE '%' || $1 || '%') AND t.site_id = $4
ORDER BY c.name NULLS FIRST, t.name
LIMIT $2 OFFSET $3`)

var qAdminTagCount = register("AdminTagCount", `
SELECT count(*) FROM web_tag t WHERE ($1 = '' OR t.name ILIKE '%' || $1 || '%') AND t.site_id = $2`)

func (d *DB) AdminTags(ctx context.Context, siteID int64, query string, limit, offset int) ([]TagRow, int, error) {
	var total int
	if err := d.pool.QueryRow(ctx, qAdminTagCount, query, siteID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tags: %w", err)
	}
	rows, err := d.pool.Query(ctx, qAdminTags, query, limit, offset, siteID)
	if err != nil {
		return nil, 0, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()

	var out []TagRow
	for rows.Next() {
		var t TagRow
		if err := rows.Scan(&t.ID, &t.Name, &t.CategoryID, &t.CategoryName, &t.Articles); err != nil {
			return nil, 0, err
		}
		out = append(out, t)
	}
	return out, total, rows.Err()
}

var qAdminTag = register("AdminTag", `SELECT id, name, category_id, is_indexed FROM web_tag WHERE id = $1 AND site_id = $2`)

func (d *DB) AdminTag(ctx context.Context, siteID, id int64) (TagRow, error) {
	var t TagRow
	err := d.pool.QueryRow(ctx, qAdminTag, id, siteID).Scan(&t.ID, &t.Name, &t.CategoryID, &t.IsIndexed)
	if errors.Is(err, pgx.ErrNoRows) {
		return TagRow{}, ErrNotFound
	}
	if err != nil {
		return TagRow{}, fmt.Errorf("read tag %d: %w", id, err)
	}
	return t, nil
}

var (
	qInsertTagRow = register("InsertTagRow", `INSERT INTO web_tag (name, category_id, site_id, is_indexed) VALUES ($1,$2,$3,$4) RETURNING id`)
	qUpdateTag    = register("UpdateTag", `UPDATE web_tag SET name=$2, category_id=$3, is_indexed=$5 WHERE id=$1 AND site_id=$4`)
	qDetachTag    = register("DetachTag", `DELETE FROM web_article_tags WHERE tag_id = $1`)
	qDeleteTagRow = register("DeleteTagRow", `DELETE FROM web_tag WHERE id = $1 AND site_id = $2`)
)

func (d *DB) SaveTag(ctx context.Context, siteID int64, t TagRow) error {
	if t.ID == 0 {
		var id int64
		if err := d.pool.QueryRow(ctx, qInsertTagRow, t.Name, t.CategoryID, siteID, t.IsIndexed).Scan(&id); err != nil {
			return fmt.Errorf("create tag %q: %w", t.Name, err)
		}
		return nil
	}
	if _, err := d.pool.Exec(ctx, qUpdateTag, t.ID, t.Name, t.CategoryID, siteID, t.IsIndexed); err != nil {
		return fmt.Errorf("update tag %d: %w", t.ID, err)
	}
	return nil
}

func (d *DB) DeleteTag(ctx context.Context, siteID, id int64) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin deleting tag %d: %w", id, err)
	}
	defer tx.Rollback(context.WithoutCancel(ctx))

	if _, err := tx.Exec(ctx, qDetachTag, id); err != nil {
		return fmt.Errorf("detach tag %d: %w", id, err)
	}
	if _, err := tx.Exec(ctx, qDeleteTagRow, id, siteID); err != nil {
		return fmt.Errorf("delete tag %d: %w", id, err)
	}
	return tx.Commit(ctx)
}
