package db

import (
	"context"
	"fmt"

	"github.com/WikitTeam/ProjectWikit/internal/perms"
)

var qRoleIDsBySlug = register("RoleIDsBySlug", `
SELECT slug, id
FROM web_role
WHERE slug = ANY($1)`)

// The roles Django reads here are ordered by index. Nothing downstream depends
// on that: each role's permissions are settled on their own and then merged.
var qRoleIDsForUser = register("RoleIDsForUser", `
SELECT ur.role_id
FROM web_user_roles ur
JOIN web_role r ON r.id = ur.role_id
WHERE ur.user_id = $1
ORDER BY r.index DESC, r.id`)

var qRolePermissions = register("RolePermissions", `
SELECT rp.role_id, p.codename, false AS restricted
FROM web_role_permissions rp
JOIN auth_permission p ON p.id = rp.permission_id
WHERE rp.role_id = ANY($1)
UNION ALL
SELECT rr.role_id, p.codename, true
FROM web_role_restrictions rr
JOIN auth_permission p ON p.id = rr.permission_id
WHERE rr.role_id = ANY($1)`)

// An override with no permissions at all still has to come back: it takes the
// one slot its role reads, and the rows after it are never looked at.
var qCategoryOverrides = register("CategoryOverrides", `
SELECT o.id, o.role_id, op.codename, op.restricted
FROM web_category c
JOIN web_category_permissions_override cpo ON cpo.category_id = c.id
JOIN web_rolepermissionsoverride o ON o.id = cpo.rolepermissionsoverride_id
LEFT JOIN (
    SELECT x.rolepermissionsoverride_id AS override_id, p.codename, false AS restricted
    FROM web_rolepermissionsoverride_permissions x
    JOIN auth_permission p ON p.id = x.permission_id
    UNION ALL
    SELECT x.rolepermissionsoverride_id, p.codename, true
    FROM web_rolepermissionsoverride_restrictions x
    JOIN auth_permission p ON p.id = x.permission_id
) op ON op.override_id = o.id
WHERE c.name = $1
ORDER BY o.id`)

var qArticleHasAuthor = register("ArticleHasAuthor", `
SELECT EXISTS (
    SELECT 1 FROM web_article_authors WHERE article_id = $1 AND user_id = $2
)`)

// RoleIDsBySlug resolves the two roles every visitor carries. A slug with no
// row is left out rather than created, which is where this parts ways with
// Django's get_or_create on a read path.
func (d *DB) RoleIDsBySlug(ctx context.Context, slugs []string) (map[string]int64, error) {
	rows, err := d.pool.Query(ctx, qRoleIDsBySlug, slugs)
	if err != nil {
		return nil, fmt.Errorf("look up roles %v: %w", slugs, err)
	}
	defer rows.Close()

	out := make(map[string]int64, len(slugs))
	for rows.Next() {
		var slug string
		var id int64
		if err := rows.Scan(&slug, &id); err != nil {
			return nil, fmt.Errorf("scan role slug: %w", err)
		}
		out[slug] = id
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("look up roles %v: %w", slugs, err)
	}
	return out, nil
}

func (d *DB) RoleIDsForUser(ctx context.Context, userID int64) ([]int64, error) {
	rows, err := d.pool.Query(ctx, qRoleIDsForUser, userID)
	if err != nil {
		return nil, fmt.Errorf("list role ids of user %d: %w", userID, err)
	}
	defer rows.Close()

	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan role id of user %d: %w", userID, err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list role ids of user %d: %w", userID, err)
	}
	return out, nil
}

// RolePermissions answers in the order the ids arrived, and a role with no rows
// of its own still comes back so the merge sees it.
func (d *DB) RolePermissions(ctx context.Context, ids []int64) ([]perms.Role, error) {
	byID := make(map[int64]*perms.Role, len(ids))
	out := make([]perms.Role, len(ids))
	for i, id := range ids {
		out[i].ID = id
		byID[id] = &out[i]
	}

	rows, err := d.pool.Query(ctx, qRolePermissions, ids)
	if err != nil {
		return nil, fmt.Errorf("list permissions of roles %v: %w", ids, err)
	}
	defer rows.Close()

	for rows.Next() {
		var roleID int64
		var codename string
		var restricted bool
		if err := rows.Scan(&roleID, &codename, &restricted); err != nil {
			return nil, fmt.Errorf("scan role permission: %w", err)
		}
		role, ok := byID[roleID]
		if !ok {
			continue
		}
		if restricted {
			role.Restrictions = append(role.Restrictions, codename)
			continue
		}
		role.Permissions = append(role.Permissions, codename)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list permissions of roles %v: %w", ids, err)
	}
	return out, nil
}

func (d *DB) CategoryOverrides(ctx context.Context, category string) ([]perms.Override, error) {
	rows, err := d.pool.Query(ctx, qCategoryOverrides, category)
	if err != nil {
		return nil, fmt.Errorf("list permission overrides of category %q: %w", category, err)
	}
	defer rows.Close()

	var out []perms.Override
	var current int64
	for rows.Next() {
		var id, roleID int64
		var codename *string
		var restricted *bool
		if err := rows.Scan(&id, &roleID, &codename, &restricted); err != nil {
			return nil, fmt.Errorf("scan permission override of category %q: %w", category, err)
		}
		if len(out) == 0 || current != id {
			out = append(out, perms.Override{RoleID: roleID})
			current = id
		}
		if codename == nil {
			continue
		}
		override := &out[len(out)-1]
		if *restricted {
			override.Restrictions = append(override.Restrictions, *codename)
			continue
		}
		override.Permissions = append(override.Permissions, *codename)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list permission overrides of category %q: %w", category, err)
	}
	return out, nil
}

func (d *DB) ArticleHasAuthor(ctx context.Context, articleID, userID int64) (bool, error) {
	var found bool
	if err := d.pool.QueryRow(ctx, qArticleHasAuthor, articleID, userID).Scan(&found); err != nil {
		return false, fmt.Errorf("check author %d of article %d: %w", userID, articleID, err)
	}
	return found, nil
}
