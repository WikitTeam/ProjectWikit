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
WHERE ur.user_id = $1
ORDER BY r.index, r.id`)

// RolesByUser returns every role of one user, ordered the way the name tail
// and showcase queries both consume it. The tie-break on id has no counterpart
// in Django, where index is rewritten to be unique on every role save.
func (d *DB) RolesByUser(ctx context.Context, userID int64) ([]roles.Role, error) {
	rows, err := d.pool.Query(ctx, qRolesByUser, userID)
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
