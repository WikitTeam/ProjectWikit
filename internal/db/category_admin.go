package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type CategoryRow struct {
	ID        int64
	Name      string
	IsIndexed bool
	Articles  int

	Settings  SiteSettings
	Overrides []CategoryOverride
}

type CategoryOverride struct {
	RoleID   int64
	RoleName string
	Allow    []string
	Deny     []string
}

var qAdminCategories = register("AdminCategories", `
SELECT c.id, c.name, c.is_indexed,
       (SELECT count(*) FROM web_article a WHERE a.category = c.name AND a.site_id = c.site_id)
FROM web_category c WHERE c.site_id = $1 ORDER BY c.name, c.id`)

func (d *DB) AdminCategories(ctx context.Context, siteID int64) ([]CategoryRow, error) {
	rows, err := d.pool.Query(ctx, qAdminCategories, siteID)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var out []CategoryRow
	for rows.Next() {
		var c CategoryRow
		if err := rows.Scan(&c.ID, &c.Name, &c.IsIndexed, &c.Articles); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

var qAdminCategory = register("AdminCategory", `
SELECT id, name, is_indexed FROM web_category WHERE id = $1`)

var qCategorySettings = register("CategorySettings", `
SELECT rating_mode, can_user_create_tags FROM web_settings WHERE category_id = $1`)

var qAdminCategoryOverrides = register("AdminCategoryOverrides", `
SELECT o.role_id, coalesce(nullif(r.name, ''), r.slug), p.codename, x.restricted
FROM web_category_permissions_override cpo
JOIN web_rolepermissionsoverride o ON o.id = cpo.rolepermissionsoverride_id
JOIN web_role r ON r.id = o.role_id
LEFT JOIN (
    SELECT rolepermissionsoverride_id AS override_id, permission_id, false AS restricted
    FROM web_rolepermissionsoverride_permissions
    UNION ALL
    SELECT rolepermissionsoverride_id, permission_id, true
    FROM web_rolepermissionsoverride_restrictions
) x ON x.override_id = o.id
LEFT JOIN auth_permission p ON p.id = x.permission_id
WHERE cpo.category_id = $1
ORDER BY r.index, o.role_id`)

func (d *DB) AdminCategory(ctx context.Context, id int64) (CategoryRow, error) {
	var c CategoryRow
	err := d.pool.QueryRow(ctx, qAdminCategory, id).Scan(&c.ID, &c.Name, &c.IsIndexed)
	if errors.Is(err, pgx.ErrNoRows) {
		return CategoryRow{}, ErrNotFound
	}
	if err != nil {
		return CategoryRow{}, fmt.Errorf("read category %d: %w", id, err)
	}

	settings, err := d.pool.Query(ctx, qCategorySettings, id)
	if err != nil {
		return CategoryRow{}, fmt.Errorf("read the settings of category %d: %w", id, err)
	}
	if settings.Next() {
		if err := settings.Scan(&c.Settings.RatingMode, &c.Settings.CreateTags); err != nil {
			settings.Close()
			return CategoryRow{}, err
		}
	}
	settings.Close()
	if err := settings.Err(); err != nil {
		return CategoryRow{}, err
	}

	rows, err := d.pool.Query(ctx, qAdminCategoryOverrides, id)
	if err != nil {
		return CategoryRow{}, fmt.Errorf("read the overrides of category %d: %w", id, err)
	}
	defer rows.Close()

	byRole := map[int64]*CategoryOverride{}
	for rows.Next() {
		var roleID int64
		var roleName string
		var codename *string
		var restricted *bool
		if err := rows.Scan(&roleID, &roleName, &codename, &restricted); err != nil {
			return CategoryRow{}, err
		}
		one, ok := byRole[roleID]
		if !ok {
			one = &CategoryOverride{RoleID: roleID, RoleName: roleName}
			byRole[roleID] = one
			c.Overrides = append(c.Overrides, CategoryOverride{})
		}
		if codename == nil || restricted == nil {
			continue
		}
		if *restricted {
			one.Deny = append(one.Deny, *codename)
		} else {
			one.Allow = append(one.Allow, *codename)
		}
	}
	if err := rows.Err(); err != nil {
		return CategoryRow{}, err
	}
	c.Overrides = c.Overrides[:0]
	for _, one := range byRole {
		c.Overrides = append(c.Overrides, *one)
	}
	return c, nil
}

var (
	qInsertCategory         = register("InsertCategory", `INSERT INTO web_category (name, is_indexed, site_id) VALUES ($1,$2,$3) RETURNING id`)
	qUpdateCategory         = register("UpdateCategory", `UPDATE web_category SET name=$2, is_indexed=$3 WHERE id=$1`)
	qUpsertCategorySettings = register("UpsertCategorySettings", `
INSERT INTO web_settings (category_id, site_id, rating_mode, can_user_create_tags)
VALUES ($1, NULL, $2, $3)
ON CONFLICT (category_id) DO UPDATE SET rating_mode = EXCLUDED.rating_mode,
	can_user_create_tags = EXCLUDED.can_user_create_tags`)
)

func (d *DB) SaveCategory(ctx context.Context, siteID int64, c CategoryRow) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin saving category %q: %w", c.Name, err)
	}
	defer tx.Rollback(context.WithoutCancel(ctx))

	if c.ID == 0 {
		if err := tx.QueryRow(ctx, qInsertCategory, c.Name, c.IsIndexed, siteID).Scan(&c.ID); err != nil {
			return fmt.Errorf("create category %q: %w", c.Name, err)
		}
	} else if _, err := tx.Exec(ctx, qUpdateCategory, c.ID, c.Name, c.IsIndexed); err != nil {
		return fmt.Errorf("update category %d: %w", c.ID, err)
	}
	if _, err := tx.Exec(ctx, qUpsertCategorySettings, c.ID, c.Settings.RatingMode, c.Settings.CreateTags); err != nil {
		return fmt.Errorf("save the settings of category %d: %w", c.ID, err)
	}
	return tx.Commit(ctx)
}

var (
	qCategoryOverrideIDs = register("CategoryOverrideIDs", `
SELECT rolepermissionsoverride_id FROM web_category_permissions_override WHERE category_id = $1`)
	qDropOverrideGrants   = register("DropOverrideGrants", `DELETE FROM web_rolepermissionsoverride_permissions WHERE rolepermissionsoverride_id = ANY($1)`)
	qDropOverrideDenies   = register("DropOverrideDenies", `DELETE FROM web_rolepermissionsoverride_restrictions WHERE rolepermissionsoverride_id = ANY($1)`)
	qDropCategoryLinks    = register("DropCategoryLinks", `DELETE FROM web_category_permissions_override WHERE category_id = $1`)
	qDropOverrides        = register("DropOverrides", `DELETE FROM web_rolepermissionsoverride WHERE id = ANY($1)`)
	qInsertOverride       = register("InsertOverride", `INSERT INTO web_rolepermissionsoverride (role_id) VALUES ($1) RETURNING id`)
	qLinkCategoryOverride = register("LinkCategoryOverride", `
INSERT INTO web_category_permissions_override (category_id, rolepermissionsoverride_id) VALUES ($1, $2)`)
	qGrantOverride = register("GrantOverride", `
INSERT INTO web_rolepermissionsoverride_permissions (rolepermissionsoverride_id, permission_id)
SELECT $1, p.id FROM auth_permission p JOIN django_content_type c ON c.id = p.content_type_id
WHERE c.app_label = 'web' AND c.model = 'roles' AND p.codename = ANY($2)`)
	qDenyOverride = register("DenyOverride", `
INSERT INTO web_rolepermissionsoverride_restrictions (rolepermissionsoverride_id, permission_id)
SELECT $1, p.id FROM auth_permission p JOIN django_content_type c ON c.id = p.content_type_id
WHERE c.app_label = 'web' AND c.model = 'roles' AND p.codename = ANY($2)`)
)

func (d *DB) SaveCategoryOverrides(ctx context.Context, categoryID int64, list []CategoryOverride) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin saving the overrides of category %d: %w", categoryID, err)
	}
	defer tx.Rollback(context.WithoutCancel(ctx))

	var existing []int64
	rows, err := tx.Query(ctx, qCategoryOverrideIDs, categoryID)
	if err != nil {
		return fmt.Errorf("read the overrides of category %d: %w", categoryID, err)
	}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		existing = append(existing, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, qDropCategoryLinks, categoryID); err != nil {
		return err
	}
	if len(existing) > 0 {
		for _, statement := range []string{qDropOverrideGrants, qDropOverrideDenies, qDropOverrides} {
			if _, err := tx.Exec(ctx, statement, existing); err != nil {
				return fmt.Errorf("clear the overrides of category %d: %w", categoryID, err)
			}
		}
	}

	for _, one := range list {
		var id int64
		if err := tx.QueryRow(ctx, qInsertOverride, one.RoleID).Scan(&id); err != nil {
			return fmt.Errorf("create an override for role %d: %w", one.RoleID, err)
		}
		if _, err := tx.Exec(ctx, qLinkCategoryOverride, categoryID, id); err != nil {
			return err
		}
		if len(one.Allow) > 0 {
			if _, err := tx.Exec(ctx, qGrantOverride, id, one.Allow); err != nil {
				return err
			}
		}
		if len(one.Deny) > 0 {
			if _, err := tx.Exec(ctx, qDenyOverride, id, one.Deny); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

var (
	qDropCategorySettings = register("DropCategorySettings", `DELETE FROM web_settings WHERE category_id = $1`)
	qDeleteCategory       = register("DeleteCategory", `DELETE FROM web_category WHERE id = $1`)
)

func (d *DB) DeleteCategory(ctx context.Context, id int64) error {
	if err := d.SaveCategoryOverrides(ctx, id, nil); err != nil {
		return err
	}
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin deleting category %d: %w", id, err)
	}
	defer tx.Rollback(context.WithoutCancel(ctx))

	if _, err := tx.Exec(ctx, qDropCategorySettings, id); err != nil {
		return fmt.Errorf("drop the settings of category %d: %w", id, err)
	}
	if _, err := tx.Exec(ctx, qDeleteCategory, id); err != nil {
		return fmt.Errorf("delete category %d: %w", id, err)
	}
	return tx.Commit(ctx)
}
