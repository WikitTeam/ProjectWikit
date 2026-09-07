package db

import (
	"context"
	"fmt"

	"github.com/WikitTeam/ProjectWikit/internal/roles"
)

var qRolesByUser = register("RolesByUser", `
SELECT r.id, r.slug, r.name, r.short_name, r.category_id, r.index,
       r.is_staff, r.group_votes, r.inline_visual_mode, r.profile_visual_mode,
       r.color, r.icon, r.badge_text, r.badge_bg, r.badge_text_color, r.badge_show_border
FROM web_role r
JOIN web_user_roles ur ON ur.role_id = r.id
WHERE ur.user_id = $1 AND r.site_id = $2
ORDER BY r.index, r.id`)

// Ordered the way the name tail and showcase queries both consume it. The tie-
// break on id covers the rows whose index is not unique.
func (d *DB) RolesByUser(ctx context.Context, siteID, userID int64) ([]roles.Role, error) {
	rows, err := d.pool.Query(ctx, qRolesByUser, userID, siteID)
	if err != nil {
		return nil, fmt.Errorf("list roles of user %d: %w", userID, err)
	}
	defer rows.Close()

	var out []roles.Role
	for rows.Next() {
		var role roles.Role
		if err := rows.Scan(
			&role.ID, &role.Slug, &role.Name, &role.ShortName, &role.CategoryID, &role.Index,
			&role.IsStaff, &role.GroupVotes, &role.InlineVisualMode, &role.ProfileVisualMode,
			&role.Color, &role.Icon, &role.BadgeText, &role.BadgeBg, &role.BadgeTextColor,
			&role.BadgeShowBorder,
		); err != nil {
			return nil, fmt.Errorf("scan role of user %d: %w", userID, err)
		}
		out = append(out, role)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list roles of user %d: %w", userID, err)
	}
	return out, nil
}

var qRolesByUsers = register("RolesByUsers", `
SELECT ur.user_id, r.id, r.slug, r.name, r.short_name, r.category_id, r.index,
       r.is_staff, r.group_votes, r.inline_visual_mode, r.profile_visual_mode,
       r.color, r.icon, r.badge_text, r.badge_bg, r.badge_text_color, r.badge_show_border
FROM web_role r
JOIN web_user_roles ur ON ur.role_id = r.id
WHERE ur.user_id = ANY($1) AND r.site_id = $2
ORDER BY ur.user_id, r.index, r.id`)

func (d *DB) RolesByUsers(ctx context.Context, siteID int64, userIDs []int64) (map[int64][]roles.Role, error) {
	out := map[int64][]roles.Role{}
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := d.pool.Query(ctx, qRolesByUsers, userIDs, siteID)
	if err != nil {
		return nil, fmt.Errorf("list roles of %d users: %w", len(userIDs), err)
	}
	defer rows.Close()

	for rows.Next() {
		var userID int64
		var role roles.Role
		if err := rows.Scan(
			&userID,
			&role.ID, &role.Slug, &role.Name, &role.ShortName, &role.CategoryID, &role.Index,
			&role.IsStaff, &role.GroupVotes, &role.InlineVisualMode, &role.ProfileVisualMode,
			&role.Color, &role.Icon, &role.BadgeText, &role.BadgeBg, &role.BadgeTextColor,
			&role.BadgeShowBorder,
		); err != nil {
			return nil, fmt.Errorf("scan role of user %d: %w", userID, err)
		}
		out[userID] = append(out[userID], role)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list roles of %d users: %w", len(userIDs), err)
	}
	return out, nil
}

var qAllRoles = register("AllRoles", `
SELECT id, slug, name FROM web_role WHERE site_id = $1 ORDER BY index, id`)

type RoleChoice struct {
	ID   int64
	Slug string
	Name string
}

func (d *DB) AllRoles(ctx context.Context, siteID int64) ([]RoleChoice, error) {
	rows, err := d.pool.Query(ctx, qAllRoles, siteID)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	defer rows.Close()

	var out []RoleChoice
	for rows.Next() {
		var c RoleChoice
		if err := rows.Scan(&c.ID, &c.Slug, &c.Name); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (c RoleChoice) Label() string {
	if c.Name != "" {
		return c.Name
	}
	return c.Slug
}
