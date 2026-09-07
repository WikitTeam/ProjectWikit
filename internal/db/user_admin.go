package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type AdminUserRow struct {
	ID              int64
	Type            string
	Username        string
	WikidotUsername string
	DisplayName     string
	Email           string
	Bio             string
	Avatar          string
	APIKey          string

	IsActive           bool
	InactiveUntil      *time.Time
	IsForumActive      bool
	ForumInactiveUntil *time.Time
	CanSendDM          bool
	IsSuperuser        bool

	OperationIndex int
	Roles          []int64
}

const adminUserColumns = `id, type, username, coalesce(wikidot_username, ''), coalesce(display_name, ''),
	coalesce(email, ''), coalesce(bio, ''), coalesce(avatar, ''), coalesce(api_key, ''),
	is_active, inactive_until, is_forum_active, forum_inactive_until,
	can_send_direct_messages, is_superuser`

const adminUserWhere = `
WHERE ($1 = '' OR username ILIKE '%' || $1 || '%' OR wikidot_username ILIKE '%' || $1 || '%'
	OR display_name ILIKE '%' || $1 || '%' OR email ILIKE '%' || $1 || '%')
	AND ($2 = '' OR type = $2)`

var qAdminUsers = register("AdminUsers", `
SELECT `+adminUserColumns+`
FROM web_user`+adminUserWhere+`
ORDER BY CASE WHEN type = 'wikidot' THEN wikidot_username ELSE username END, id
LIMIT $3 OFFSET $4`)

var qAdminUserCount = register("AdminUserCount", `
SELECT count(*) FROM web_user`+adminUserWhere)

func scanAdminUser(row pgx.Row, u *AdminUserRow) error {
	return row.Scan(&u.ID, &u.Type, &u.Username, &u.WikidotUsername, &u.DisplayName,
		&u.Email, &u.Bio, &u.Avatar, &u.APIKey, &u.IsActive, &u.InactiveUntil,
		&u.IsForumActive, &u.ForumInactiveUntil, &u.CanSendDM, &u.IsSuperuser)
}

func (d *DB) AdminUsers(ctx context.Context, query, kind string, limit, offset int) ([]AdminUserRow, int, error) {
	var total int
	if err := d.pool.QueryRow(ctx, qAdminUserCount, query, kind).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}
	rows, err := d.pool.Query(ctx, qAdminUsers, query, kind, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var out []AdminUserRow
	for rows.Next() {
		var u AdminUserRow
		if err := scanAdminUser(rows, &u); err != nil {
			return nil, 0, err
		}
		out = append(out, u)
	}
	return out, total, rows.Err()
}

var qAdminUser = register("AdminUser", `SELECT `+adminUserColumns+` FROM web_user WHERE id = $1`)

var qAdminUserRoles = register("AdminUserRoles", `SELECT role_id FROM web_user_roles WHERE user_id = $1`)

func (d *DB) AdminUser(ctx context.Context, siteID, id int64) (AdminUserRow, error) {
	var u AdminUserRow
	err := scanAdminUser(d.pool.QueryRow(ctx, qAdminUser, id), &u)
	if errors.Is(err, pgx.ErrNoRows) {
		return AdminUserRow{}, ErrNotFound
	}
	if err != nil {
		return AdminUserRow{}, fmt.Errorf("read user %d: %w", id, err)
	}

	rows, err := d.pool.Query(ctx, qAdminUserRoles, id)
	if err != nil {
		return AdminUserRow{}, fmt.Errorf("read the roles of user %d: %w", id, err)
	}
	defer rows.Close()
	for rows.Next() {
		var roleID int64
		if err := rows.Scan(&roleID); err != nil {
			return AdminUserRow{}, err
		}
		u.Roles = append(u.Roles, roleID)
	}
	if err := rows.Err(); err != nil {
		return AdminUserRow{}, err
	}
	u.OperationIndex, err = d.OperationIndex(ctx, siteID, id)
	return u, err
}

var (
	qUpdateAdminUser = register("UpdateAdminUser", `
UPDATE web_user SET username=$2, wikidot_username=$3, display_name=$4, email=$5, bio=$6,
	is_active=$7, inactive_until=$8, is_forum_active=$9, forum_inactive_until=$10,
	can_send_direct_messages=$11
WHERE id=$1`)

	qSetSuperuser  = register("SetSuperuser", `UPDATE web_user SET is_superuser = $2 WHERE id = $1`)
	qClearUserRole = register("ClearUserRoles", `DELETE FROM web_user_roles WHERE user_id = $1`)
	qAddUserRole   = register("AddUserRoles", `
INSERT INTO web_user_roles (user_id, role_id)
SELECT $1, r.id FROM web_role r WHERE r.id = ANY($2) AND r.slug <> ALL($3) AND r.site_id = $4`)
)

func (d *DB) SaveAdminUser(ctx context.Context, siteID int64, u AdminUserRow, builtin []string, withRoles, withSuperuser bool) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin saving user %d: %w", u.ID, err)
	}
	defer tx.Rollback(context.WithoutCancel(ctx))

	_, err = tx.Exec(ctx, qUpdateAdminUser, u.ID, u.Username, nullable(u.WikidotUsername),
		nullable(u.DisplayName), nullable(u.Email), u.Bio, u.IsActive, u.InactiveUntil,
		u.IsForumActive, u.ForumInactiveUntil, u.CanSendDM)
	if err != nil {
		return fmt.Errorf("save user %d: %w", u.ID, err)
	}
	if withSuperuser {
		if _, err := tx.Exec(ctx, qSetSuperuser, u.ID, u.IsSuperuser); err != nil {
			return fmt.Errorf("set the superuser flag on %d: %w", u.ID, err)
		}
	}
	if withRoles {
		if _, err := tx.Exec(ctx, qClearUserRole, u.ID); err != nil {
			return err
		}
		if len(u.Roles) > 0 {
			if _, err := tx.Exec(ctx, qAddUserRole, u.ID, u.Roles, builtin, siteID); err != nil {
				return fmt.Errorf("give roles to user %d: %w", u.ID, err)
			}
		}
	}
	return tx.Commit(ctx)
}

var qResetUserVotes = register("ResetUserVotes", `DELETE FROM web_vote WHERE user_id = $1`)

func (d *DB) ResetUserVotes(ctx context.Context, userID int64) (int64, error) {
	tag, err := d.pool.Exec(ctx, qResetUserVotes, userID)
	if err != nil {
		return 0, fmt.Errorf("reset votes of user %d: %w", userID, err)
	}
	return tag.RowsAffected(), nil
}

var qUnclaimedWikidotUsers = register("UnclaimedWikidotUsers", `
SELECT id, coalesce(wikidot_username, '')
FROM web_user
WHERE type = 'wikidot' AND NOT is_active
ORDER BY wikidot_username, id`)

type UserChoice struct {
	ID   int64
	Name string
}

func (d *DB) UnclaimedWikidotUsers(ctx context.Context) ([]UserChoice, error) {
	rows, err := d.pool.Query(ctx, qUnclaimedWikidotUsers)
	if err != nil {
		return nil, fmt.Errorf("list unclaimed wikidot users: %w", err)
	}
	defer rows.Close()

	var out []UserChoice
	for rows.Next() {
		var one UserChoice
		if err := rows.Scan(&one.ID, &one.Name); err != nil {
			return nil, err
		}
		out = append(out, one)
	}
	return out, rows.Err()
}
