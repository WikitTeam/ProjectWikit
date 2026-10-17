package db

import (
	"context"
	"fmt"
)

var (
	qToolbarHiddenRoles = register("ToolbarHiddenRoles", `
SELECT role_id FROM pwikit_category_toolbar_hidden WHERE category_id = $1`)
	qDropToolbarHidden = register("DropToolbarHidden", `
DELETE FROM pwikit_category_toolbar_hidden WHERE category_id = $1`)
	qHideToolbar = register("HideToolbar", `
INSERT INTO pwikit_category_toolbar_hidden (category_id, role_id) VALUES ($1, $2)`)
	qToolbarHidden = register("ToolbarHidden", `
SELECT EXISTS (
	SELECT 1 FROM pwikit_category_toolbar_hidden h
	JOIN web_category c ON c.id = h.category_id
	WHERE c.site_id = $1 AND c.name = $2 AND h.role_id = ANY($3)
)`)
)

func (d *DB) ToolbarHiddenRoles(ctx context.Context, categoryID int64) ([]int64, error) {
	rows, err := d.pool.Query(ctx, qToolbarHiddenRoles, categoryID)
	if err != nil {
		return nil, fmt.Errorf("read the toolbar roles of category %d: %w", categoryID, err)
	}
	defer rows.Close()

	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (d *DB) SaveToolbarHiddenRoles(ctx context.Context, categoryID int64, roleIDs []int64) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin the toolbar roles of category %d: %w", categoryID, err)
	}
	defer tx.Rollback(context.WithoutCancel(ctx))

	if _, err := tx.Exec(ctx, qDropToolbarHidden, categoryID); err != nil {
		return fmt.Errorf("clear the toolbar roles of category %d: %w", categoryID, err)
	}
	for _, id := range roleIDs {
		if _, err := tx.Exec(ctx, qHideToolbar, categoryID, id); err != nil {
			return fmt.Errorf("hide the toolbar of category %d from role %d: %w", categoryID, id, err)
		}
	}
	return tx.Commit(ctx)
}

func (d *DB) ToolbarHidden(ctx context.Context, siteID int64, category string, roleIDs []int64) (bool, error) {
	if len(roleIDs) == 0 {
		return false, nil
	}
	var hidden bool
	if err := d.pool.QueryRow(ctx, qToolbarHidden, siteID, category, roleIDs).Scan(&hidden); err != nil {
		return false, fmt.Errorf("check the toolbar of category %q: %w", category, err)
	}
	return hidden, nil
}
