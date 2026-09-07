package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type ForumSectionRow struct {
	ID              int64
	Name            string
	Description     string
	Order           int
	IsHidden        bool
	IsHiddenForUser bool
	Categories      int
}

var qAdminForumSections = register("AdminForumSections", `
SELECT s.id, s.name, s.description, s."order", s.is_hidden, s.is_hidden_for_users,
       (SELECT count(*) FROM web_forumcategory c WHERE c.section_id = s.id)
FROM web_forumsection s WHERE s.site_id = $1 ORDER BY s."order", s.id`)

func (d *DB) AdminForumSections(ctx context.Context, siteID int64) ([]ForumSectionRow, error) {
	rows, err := d.pool.Query(ctx, qAdminForumSections, siteID)
	if err != nil {
		return nil, fmt.Errorf("list forum sections: %w", err)
	}
	defer rows.Close()

	var out []ForumSectionRow
	for rows.Next() {
		var s ForumSectionRow
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.Order, &s.IsHidden, &s.IsHiddenForUser, &s.Categories); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

var qAdminForumSection = register("AdminForumSection", `
SELECT id, name, description, "order", is_hidden, is_hidden_for_users
FROM web_forumsection WHERE id = $1`)

func (d *DB) AdminForumSection(ctx context.Context, id int64) (ForumSectionRow, error) {
	var s ForumSectionRow
	err := d.pool.QueryRow(ctx, qAdminForumSection, id).
		Scan(&s.ID, &s.Name, &s.Description, &s.Order, &s.IsHidden, &s.IsHiddenForUser)
	if errors.Is(err, pgx.ErrNoRows) {
		return ForumSectionRow{}, ErrNotFound
	}
	if err != nil {
		return ForumSectionRow{}, fmt.Errorf("read forum section %d: %w", id, err)
	}
	return s, nil
}

var (
	qInsertForumSection = register("InsertForumSection", `
INSERT INTO web_forumsection (name, description, "order", is_hidden, is_hidden_for_users, site_id)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`)

	qUpdateForumSection = register("UpdateForumSection", `
UPDATE web_forumsection SET name=$2, description=$3, "order"=$4, is_hidden=$5, is_hidden_for_users=$6
WHERE id=$1`)

	qDeleteForumSection = register("DeleteForumSection", `DELETE FROM web_forumsection WHERE id = $1`)
)

func (d *DB) SaveForumSection(ctx context.Context, siteID int64, s ForumSectionRow) error {
	if s.ID == 0 {
		var id int64
		err := d.pool.QueryRow(ctx, qInsertForumSection, s.Name, s.Description, s.Order, s.IsHidden, s.IsHiddenForUser, siteID).Scan(&id)
		if err != nil {
			return fmt.Errorf("create forum section %q: %w", s.Name, err)
		}
		return nil
	}
	_, err := d.pool.Exec(ctx, qUpdateForumSection, s.ID, s.Name, s.Description, s.Order, s.IsHidden, s.IsHiddenForUser)
	if err != nil {
		return fmt.Errorf("update forum section %d: %w", s.ID, err)
	}
	return nil
}

func (d *DB) DeleteForumSection(ctx context.Context, id int64) error {
	if _, err := d.pool.Exec(ctx, qDeleteForumSection, id); err != nil {
		return fmt.Errorf("delete forum section %d: %w", id, err)
	}
	return nil
}

type ForumCategoryRow struct {
	ID            int64
	Name          string
	Description   string
	Order         int
	IsForComments bool
	SectionID     int64
	SectionName   string
	Threads       int
}

var qAdminForumCategories = register("AdminForumCategories", `
SELECT c.id, c.name, c.description, c."order", c.is_for_comments, c.section_id, s.name,
       (SELECT count(*) FROM web_forumthread t WHERE t.category_id = c.id)
FROM web_forumcategory c JOIN web_forumsection s ON s.id = c.section_id
WHERE s.site_id = $1
ORDER BY s."order", c."order", c.id`)

func (d *DB) AdminForumCategories(ctx context.Context, siteID int64) ([]ForumCategoryRow, error) {
	rows, err := d.pool.Query(ctx, qAdminForumCategories, siteID)
	if err != nil {
		return nil, fmt.Errorf("list forum categories: %w", err)
	}
	defer rows.Close()

	var out []ForumCategoryRow
	for rows.Next() {
		var c ForumCategoryRow
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.Order, &c.IsForComments, &c.SectionID, &c.SectionName, &c.Threads); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

var qAdminForumCategory = register("AdminForumCategory", `
SELECT id, name, description, "order", is_for_comments, section_id
FROM web_forumcategory WHERE id = $1`)

func (d *DB) AdminForumCategory(ctx context.Context, id int64) (ForumCategoryRow, error) {
	var c ForumCategoryRow
	err := d.pool.QueryRow(ctx, qAdminForumCategory, id).
		Scan(&c.ID, &c.Name, &c.Description, &c.Order, &c.IsForComments, &c.SectionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ForumCategoryRow{}, ErrNotFound
	}
	if err != nil {
		return ForumCategoryRow{}, fmt.Errorf("read forum category %d: %w", id, err)
	}
	return c, nil
}

var (
	qInsertForumCategory = register("InsertForumCategory", `
INSERT INTO web_forumcategory (name, description, "order", is_for_comments, section_id)
VALUES ($1, $2, $3, $4, $5) RETURNING id`)

	qUpdateForumCategory = register("UpdateForumCategory", `
UPDATE web_forumcategory SET name=$2, description=$3, "order"=$4, is_for_comments=$5, section_id=$6
WHERE id=$1`)

	qDeleteForumCategory = register("DeleteForumCategory", `DELETE FROM web_forumcategory WHERE id = $1`)
)

func (d *DB) SaveForumCategory(ctx context.Context, c ForumCategoryRow) error {
	if c.ID == 0 {
		var id int64
		err := d.pool.QueryRow(ctx, qInsertForumCategory, c.Name, c.Description, c.Order, c.IsForComments, c.SectionID).Scan(&id)
		if err != nil {
			return fmt.Errorf("create forum category %q: %w", c.Name, err)
		}
		return nil
	}
	_, err := d.pool.Exec(ctx, qUpdateForumCategory, c.ID, c.Name, c.Description, c.Order, c.IsForComments, c.SectionID)
	if err != nil {
		return fmt.Errorf("update forum category %d: %w", c.ID, err)
	}
	return nil
}

func (d *DB) DeleteForumCategory(ctx context.Context, id int64) error {
	if _, err := d.pool.Exec(ctx, qDeleteForumCategory, id); err != nil {
		return fmt.Errorf("delete forum category %d: %w", id, err)
	}
	return nil
}
