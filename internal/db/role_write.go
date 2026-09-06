package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type RoleRow struct {
	ID                int64
	Slug              string
	Name              string
	ShortName         string
	CategoryID        *int64
	Index             int
	IsStaff           bool
	GroupVotes        bool
	VotesTitle        string
	InlineVisualMode  string
	ProfileVisualMode string
	Color             string
	Icon              string
	BadgeText         string
	BadgeBg           string
	BadgeTextColor    string
	BadgeShowBorder   bool

	Allow []string
	Deny  []string
	Users int
}

const roleColumns = `id, slug, name, coalesce(short_name, ''), category_id, index, is_staff,
	group_votes, coalesce(votes_title, ''), inline_visual_mode, profile_visual_mode,
	coalesce(color, ''), coalesce(icon, ''), coalesce(badge_text, ''), coalesce(badge_bg, ''),
	coalesce(badge_text_color, ''), badge_show_border`

func scanRole(row pgx.Row, r *RoleRow) error {
	return row.Scan(&r.ID, &r.Slug, &r.Name, &r.ShortName, &r.CategoryID, &r.Index, &r.IsStaff,
		&r.GroupVotes, &r.VotesTitle, &r.InlineVisualMode, &r.ProfileVisualMode,
		&r.Color, &r.Icon, &r.BadgeText, &r.BadgeBg, &r.BadgeTextColor, &r.BadgeShowBorder)
}

var qAdminRoles = register("AdminRoles", `
SELECT `+roleColumns+`, (SELECT count(*) FROM web_user_roles ur WHERE ur.role_id = web_role.id)
FROM web_role ORDER BY index, id`)

func (d *DB) AdminRoles(ctx context.Context) ([]RoleRow, error) {
	rows, err := d.pool.Query(ctx, qAdminRoles)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	defer rows.Close()

	var out []RoleRow
	for rows.Next() {
		var r RoleRow
		err := rows.Scan(&r.ID, &r.Slug, &r.Name, &r.ShortName, &r.CategoryID, &r.Index, &r.IsStaff,
			&r.GroupVotes, &r.VotesTitle, &r.InlineVisualMode, &r.ProfileVisualMode,
			&r.Color, &r.Icon, &r.BadgeText, &r.BadgeBg, &r.BadgeTextColor, &r.BadgeShowBorder, &r.Users)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

var qAdminRole = register("AdminRole", `SELECT `+roleColumns+` FROM web_role WHERE id = $1`)

var qRoleGrants = register("RoleGrants", `
SELECT p.codename, false FROM web_role_permissions rp
JOIN auth_permission p ON p.id = rp.permission_id WHERE rp.role_id = $1
UNION ALL
SELECT p.codename, true FROM web_role_restrictions rr
JOIN auth_permission p ON p.id = rr.permission_id WHERE rr.role_id = $1`)

func (d *DB) AdminRole(ctx context.Context, id int64) (RoleRow, error) {
	var r RoleRow
	err := scanRole(d.pool.QueryRow(ctx, qAdminRole, id), &r)
	if errors.Is(err, pgx.ErrNoRows) {
		return RoleRow{}, ErrNotFound
	}
	if err != nil {
		return RoleRow{}, fmt.Errorf("read role %d: %w", id, err)
	}

	rows, err := d.pool.Query(ctx, qRoleGrants, id)
	if err != nil {
		return RoleRow{}, fmt.Errorf("read the grants of role %d: %w", id, err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var denied bool
		if err := rows.Scan(&name, &denied); err != nil {
			return RoleRow{}, err
		}
		if denied {
			r.Deny = append(r.Deny, name)
		} else {
			r.Allow = append(r.Allow, name)
		}
	}
	return r, rows.Err()
}

var qPermissionCatalog = register("PermissionCatalog", `
SELECT p.codename FROM auth_permission p
JOIN django_content_type c ON c.id = p.content_type_id
WHERE c.app_label = 'web' AND c.model = 'roles'
ORDER BY p.codename`)

func (d *DB) PermissionCatalog(ctx context.Context) ([]string, error) {
	rows, err := d.pool.Query(ctx, qPermissionCatalog)
	if err != nil {
		return nil, fmt.Errorf("read the permission catalog: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

var (
	qInsertRole = register("InsertRole", `
INSERT INTO web_role (slug, name, short_name, category_id, index, is_staff, group_votes,
	votes_title, inline_visual_mode, profile_visual_mode, color, icon, badge_text, badge_bg,
	badge_text_color, badge_show_border)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16) RETURNING id`)

	qUpdateRole = register("UpdateRole", `
UPDATE web_role SET slug=$2, name=$3, short_name=$4, category_id=$5, index=$6, is_staff=$7,
	group_votes=$8, votes_title=$9, inline_visual_mode=$10, profile_visual_mode=$11,
	color=$12, icon=$13, badge_text=$14, badge_bg=$15, badge_text_color=$16, badge_show_border=$17
WHERE id=$1`)

	qClearRolePermissions  = register("ClearRolePermissions", `DELETE FROM web_role_permissions WHERE role_id = $1`)
	qClearRoleRestrictions = register("ClearRoleRestrictions", `DELETE FROM web_role_restrictions WHERE role_id = $1`)

	qGrantRolePermission = register("GrantRolePermission", `
INSERT INTO web_role_permissions (role_id, permission_id)
SELECT $1, p.id FROM auth_permission p
JOIN django_content_type c ON c.id = p.content_type_id
WHERE c.app_label = 'web' AND c.model = 'roles' AND p.codename = ANY($2)`)

	qRestrictRole = register("RestrictRole", `
INSERT INTO web_role_restrictions (role_id, permission_id)
SELECT $1, p.id FROM auth_permission p
JOIN django_content_type c ON c.id = p.content_type_id
WHERE c.app_label = 'web' AND c.model = 'roles' AND p.codename = ANY($2)`)
)

func (d *DB) SaveRole(ctx context.Context, r RoleRow, withGrants bool) (int64, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin saving role %q: %w", r.Slug, err)
	}
	defer tx.Rollback(context.WithoutCancel(ctx))

	args := []any{r.Slug, r.Name, r.ShortName, r.CategoryID, r.Index, r.IsStaff, r.GroupVotes,
		r.VotesTitle, r.InlineVisualMode, r.ProfileVisualMode, r.Color, r.Icon,
		r.BadgeText, r.BadgeBg, r.BadgeTextColor, r.BadgeShowBorder}
	if r.ID == 0 {
		if err := tx.QueryRow(ctx, qInsertRole, args...).Scan(&r.ID); err != nil {
			return 0, fmt.Errorf("create role %q: %w", r.Slug, err)
		}
	} else if _, err := tx.Exec(ctx, qUpdateRole, append([]any{r.ID}, args...)...); err != nil {
		return 0, fmt.Errorf("update role %d: %w", r.ID, err)
	}

	if withGrants {
		if _, err := tx.Exec(ctx, qClearRolePermissions, r.ID); err != nil {
			return 0, err
		}
		if _, err := tx.Exec(ctx, qClearRoleRestrictions, r.ID); err != nil {
			return 0, err
		}
		if len(r.Allow) > 0 {
			if _, err := tx.Exec(ctx, qGrantRolePermission, r.ID, r.Allow); err != nil {
				return 0, fmt.Errorf("grant to role %d: %w", r.ID, err)
			}
		}
		if len(r.Deny) > 0 {
			if _, err := tx.Exec(ctx, qRestrictRole, r.ID, r.Deny); err != nil {
				return 0, fmt.Errorf("restrict role %d: %w", r.ID, err)
			}
		}
	}
	return r.ID, tx.Commit(ctx)
}

var qDeleteRole = register("DeleteRole", `DELETE FROM web_role WHERE id = $1`)

var roleDependents = []string{
	register("DropRoleGrants", `DELETE FROM web_role_permissions WHERE role_id = $1`),
	register("DropRoleRestrictions", `DELETE FROM web_role_restrictions WHERE role_id = $1`),
	register("DropRoleHolders", `DELETE FROM web_user_roles WHERE role_id = $1`),
	register("DropRoleOverrides", `DELETE FROM web_rolepermissionsoverride WHERE role_id = $1`),
	register("ClearRoleVotes", `UPDATE web_vote SET role_id = NULL WHERE role_id = $1`),
	register("ClearRoleTickets", `UPDATE web_userticket SET granted_role_id = NULL WHERE granted_role_id = $1`),
	register("ClearRoleOnSite", `UPDATE web_site SET
		default_role_id = CASE WHEN default_role_id = $1 THEN NULL ELSE default_role_id END,
		verified_role_id = CASE WHEN verified_role_id = $1 THEN NULL ELSE verified_role_id END,
		membership_password_role_id = CASE WHEN membership_password_role_id = $1
			THEN NULL ELSE membership_password_role_id END`),
}

func (d *DB) DeleteRole(ctx context.Context, id int64) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin deleting role %d: %w", id, err)
	}
	defer tx.Rollback(context.WithoutCancel(ctx))

	for _, statement := range roleDependents {
		if _, err := tx.Exec(ctx, statement, id); err != nil {
			return fmt.Errorf("detach role %d: %w", id, err)
		}
	}
	if _, err := tx.Exec(ctx, qDeleteRole, id); err != nil {
		return fmt.Errorf("delete role %d: %w", id, err)
	}
	return tx.Commit(ctx)
}

var qOperationIndex = register("OperationIndex", `
SELECT coalesce(min(r.index), 2147483647) FROM web_role r
JOIN web_user_roles ur ON ur.role_id = r.id WHERE ur.user_id = $1`)

func (d *DB) OperationIndex(ctx context.Context, userID int64) (int, error) {
	var index int
	if err := d.pool.QueryRow(ctx, qOperationIndex, userID).Scan(&index); err != nil {
		return 0, fmt.Errorf("read the rank of user %d: %w", userID, err)
	}
	return index, nil
}

type RoleCategoryRow struct {
	ID    int64
	Name  string
	Roles int
}

var qRoleCategories = register("RoleCategories", `
SELECT c.id, c.name, (SELECT count(*) FROM web_role r WHERE r.category_id = c.id)
FROM web_rolecategory c ORDER BY c.name, c.id`)

func (d *DB) RoleCategories(ctx context.Context) ([]RoleCategoryRow, error) {
	rows, err := d.pool.Query(ctx, qRoleCategories)
	if err != nil {
		return nil, fmt.Errorf("list role categories: %w", err)
	}
	defer rows.Close()

	var out []RoleCategoryRow
	for rows.Next() {
		var c RoleCategoryRow
		if err := rows.Scan(&c.ID, &c.Name, &c.Roles); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

var (
	qInsertRoleCategory = register("InsertRoleCategory", `INSERT INTO web_rolecategory (name) VALUES ($1) RETURNING id`)
	qUpdateRoleCategory = register("UpdateRoleCategory", `UPDATE web_rolecategory SET name = $2 WHERE id = $1`)
	qDeleteRoleCategory = register("DeleteRoleCategory", `DELETE FROM web_rolecategory WHERE id = $1`)
)

func (d *DB) SaveRoleCategory(ctx context.Context, c RoleCategoryRow) error {
	if c.ID == 0 {
		var id int64
		if err := d.pool.QueryRow(ctx, qInsertRoleCategory, c.Name).Scan(&id); err != nil {
			return fmt.Errorf("create role category %q: %w", c.Name, err)
		}
		return nil
	}
	if _, err := d.pool.Exec(ctx, qUpdateRoleCategory, c.ID, c.Name); err != nil {
		return fmt.Errorf("update role category %d: %w", c.ID, err)
	}
	return nil
}

var qDetachRoleCategory = register("DetachRoleCategory", `UPDATE web_role SET category_id = NULL WHERE category_id = $1`)

func (d *DB) DeleteRoleCategory(ctx context.Context, id int64) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin deleting role category %d: %w", id, err)
	}
	defer tx.Rollback(context.WithoutCancel(ctx))

	if _, err := tx.Exec(ctx, qDetachRoleCategory, id); err != nil {
		return fmt.Errorf("detach role category %d: %w", id, err)
	}
	if _, err := tx.Exec(ctx, qDeleteRoleCategory, id); err != nil {
		return fmt.Errorf("delete role category %d: %w", id, err)
	}
	return tx.Commit(ctx)
}
