package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/roles"
	"github.com/jackc/pgx/v5"
)

type Member struct {
	User
	JoinedAt time.Time
}

// A null role means every account, so one statement covers both the filtered
// and the unfiltered listing rather than two that can drift apart.
const memberFilter = `
WHERE $1::bigint IS NULL
   OR EXISTS (SELECT 1 FROM web_user_roles ur WHERE ur.user_id = u.id AND ur.role_id = $1)`

var qMembers = register("Members", `
SELECT `+prefixed("u", userColumns)+`, u.date_joined
FROM web_user u`+memberFilter+`
ORDER BY u.id
OFFSET $2
LIMIT $3`)

var qMemberCount = register("MemberCount", `
SELECT count(*)
FROM web_user u`+memberFilter)

func (d *DB) Members(ctx context.Context, roleID *int64, offset, limit int) ([]Member, error) {
	rows, err := d.pool.Query(ctx, qMembers, roleID, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	var out []Member
	for rows.Next() {
		var m Member
		dest, finish := userDest(&m.User)
		if err := rows.Scan(append(dest, &m.JoinedAt)...); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		finish()
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	return out, nil
}

func (d *DB) MemberCount(ctx context.Context, roleID *int64) (int, error) {
	var total int
	if err := d.pool.QueryRow(ctx, qMemberCount, roleID).Scan(&total); err != nil {
		return 0, fmt.Errorf("count members: %w", err)
	}
	return total, nil
}

var qRoleByRef = register("RoleByRef", `
SELECT r.id, r.slug, r.name, r.short_name, r.category_id, r.index,
       r.is_staff, r.group_votes, r.inline_visual_mode, r.profile_visual_mode,
       r.color, r.icon, r.badge_text, r.badge_bg, r.badge_text_color, r.badge_show_border
FROM web_role r
WHERE lower(r.slug) = lower($1) OR r.id::text = $1
ORDER BY r.index, r.id
LIMIT 1`)

func (d *DB) RoleByRef(ctx context.Context, ref string) (*roles.Role, error) {
	var role roles.Role
	err := d.pool.QueryRow(ctx, qRoleByRef, ref).Scan(
		&role.ID, &role.Slug, &role.Name, &role.ShortName, &role.CategoryID, &role.Index,
		&role.IsStaff, &role.GroupVotes, &role.InlineVisualMode, &role.ProfileVisualMode,
		&role.Color, &role.Icon, &role.BadgeText, &role.BadgeBg, &role.BadgeTextColor,
		&role.BadgeShowBorder,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lookup role %q: %w", ref, err)
	}
	return &role, nil
}
