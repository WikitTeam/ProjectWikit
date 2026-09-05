package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

var qCreateUser = register("CreateUser", `
INSERT INTO web_user (password, is_superuser, first_name, last_name, email, date_joined,
	username, display_name, type, bio, is_forum_active, is_active, can_send_direct_messages)
VALUES ($1, false, '', '', '', $2, $3, $4, $5, '', true, $6, true)
RETURNING id`)

func (d *DB) CreateUser(ctx context.Context, username, displayName, hash string, active bool, at time.Time) (int64, error) {
	var display *string
	if displayName != "" {
		display = &displayName
	}
	var id int64
	err := d.pool.QueryRow(ctx, qCreateUser, hash, at, username, display, UserTypeNormal, active).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create user %q: %w", username, err)
	}
	return id, nil
}

var qUsernameTaken = register("UsernameTaken", `
SELECT EXISTS (
	SELECT 1 FROM web_user WHERE username = $1 OR wikidot_username = $1)`)

func (d *DB) UsernameTaken(ctx context.Context, name string) (bool, error) {
	var taken bool
	if err := d.pool.QueryRow(ctx, qUsernameTaken, name).Scan(&taken); err != nil {
		return false, fmt.Errorf("check name %q: %w", name, err)
	}
	return taken, nil
}

var qRenameUser = register("RenameUser", `
UPDATE web_user
SET username = $2, display_name = $3
WHERE id = $1`)

func (d *DB) RenameUser(ctx context.Context, id int64, username, displayName string) error {
	var display *string
	if displayName != "" {
		display = &displayName
	}
	if _, err := d.pool.Exec(ctx, qRenameUser, id, username, display); err != nil {
		return fmt.Errorf("rename user %d: %w", id, err)
	}
	return nil
}

var qActivateUser = register("ActivateUser", `
UPDATE web_user
SET username = $2, display_name = COALESCE($3, display_name), type = $4,
	password = $5, is_active = true
WHERE id = $1`)

func (d *DB) ActivateUser(ctx context.Context, id int64, username string, displayName *string, hash string) error {
	if _, err := d.pool.Exec(ctx, qActivateUser, id, username, displayName, UserTypeNormal, hash); err != nil {
		return fmt.Errorf("activate user %d: %w", id, err)
	}
	return nil
}

var qGrantRole = register("GrantRole", `
INSERT INTO web_user_roles (user_id, role_id)
SELECT $1, $2
WHERE NOT EXISTS (
	SELECT 1 FROM web_user_roles WHERE user_id = $1 AND role_id = $2)`)

func (d *DB) GrantRole(ctx context.Context, userID, roleID int64) error {
	if _, err := d.pool.Exec(ctx, qGrantRole, userID, roleID); err != nil {
		return fmt.Errorf("grant role %d to user %d: %w", roleID, userID, err)
	}
	return nil
}

var qUserByEmail = register("UserByEmail", `
SELECT `+userColumns+`
FROM web_user
WHERE lower(email) = lower($1) AND email <> ''
ORDER BY id
LIMIT 1`)

func (d *DB) UserByEmail(ctx context.Context, email string) (*User, error) {
	return d.scanUser(ctx, qUserByEmail, email)
}

var qSetEmail = register("SetEmail", `UPDATE web_user SET email = $2 WHERE id = $1`)

func (d *DB) SetEmail(ctx context.Context, id int64, email string) error {
	if _, err := d.pool.Exec(ctx, qSetEmail, id, email); err != nil {
		return fmt.Errorf("store email of user %d: %w", id, err)
	}
	return nil
}

var qRoleIDBySlug = register("RoleIDBySlug", `SELECT id FROM web_role WHERE slug = $1`)

func (d *DB) RoleIDBySlug(ctx context.Context, slug string) (int64, error) {
	var id int64
	err := d.pool.QueryRow(ctx, qRoleIDBySlug, slug).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("look up role %q: %w", slug, err)
	}
	return id, nil
}

var qResetFields = register("ResetFields", `
SELECT password, last_login, email
FROM web_user
WHERE id = $1`)

func (d *DB) ResetFields(ctx context.Context, id int64) (string, *time.Time, string, error) {
	var (
		hash  string
		last  *time.Time
		email string
	)
	err := d.pool.QueryRow(ctx, qResetFields, id).Scan(&hash, &last, &email)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, "", ErrNotFound
	}
	if err != nil {
		return "", nil, "", fmt.Errorf("read reset fields of user %d: %w", id, err)
	}
	return hash, last, email, nil
}

var qSetUserPreference = register("SetUserPreference", `
INSERT INTO dynamic_preferences_users_userpreferencemodel (instance_id, section, name, raw_value)
VALUES ($1, $2, $3, $4)
ON CONFLICT (instance_id, section, name) DO UPDATE SET raw_value = EXCLUDED.raw_value`)

func (d *DB) SetUserPreference(ctx context.Context, userID int64, section, name, raw string) error {
	if _, err := d.pool.Exec(ctx, qSetUserPreference, userID, section, name, raw); err != nil {
		return fmt.Errorf("store preference %s.%s of user %d: %w", section, name, userID, err)
	}
	return nil
}

var qCreateInvitedUser = register("CreateInvitedUser", `
INSERT INTO web_user (password, is_superuser, first_name, last_name, email, date_joined,
	username, type, bio, is_forum_active, is_active, can_send_direct_messages)
VALUES ('!', false, '', '', $1::text, $2, 'invite-' || md5($1::text), $3, '', true, false, true)
RETURNING id`)

var qNameInvitedUser = register("NameInvitedUser", `
UPDATE web_user SET username = 'user-' || id WHERE id = $1`)

func (d *DB) CreateInvitedUser(ctx context.Context, email string, at time.Time) (int64, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin invited user: %w", err)
	}
	defer tx.Rollback(ctx)

	var id int64
	if err := tx.QueryRow(ctx, qCreateInvitedUser, email, at, UserTypeNormal).Scan(&id); err != nil {
		return 0, fmt.Errorf("create invited user %q: %w", email, err)
	}
	if _, err := tx.Exec(ctx, qNameInvitedUser, id); err != nil {
		return 0, fmt.Errorf("name invited user %d: %w", id, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit invited user: %w", err)
	}
	return id, nil
}

var qTokenUsed = register("TokenUsed", `
SELECT EXISTS (
	SELECT 1 FROM web_usedtoken
	WHERE (is_case_sensitive AND token = $1)
		OR (NOT is_case_sensitive AND upper(token) = upper($1)))`)

func (d *DB) TokenUsed(ctx context.Context, token string) (bool, error) {
	var used bool
	if err := d.pool.QueryRow(ctx, qTokenUsed, token).Scan(&used); err != nil {
		return false, fmt.Errorf("check token: %w", err)
	}
	return used, nil
}

var qMarkTokenUsed = register("MarkTokenUsed", `
INSERT INTO web_usedtoken (token, is_case_sensitive) VALUES ($1, $2)`)

func (d *DB) MarkTokenUsed(ctx context.Context, token string, caseSensitive bool) error {
	if _, err := d.pool.Exec(ctx, qMarkTokenUsed, token, caseSensitive); err != nil {
		return fmt.Errorf("mark token used: %w", err)
	}
	return nil
}

var qActivateInviteLink = register("ActivateInviteLink", `
UPDATE web_invitelink
SET activated_at = $2, activated_username = $3
WHERE token = $1 AND activated_at IS NULL`)

func (d *DB) ActivateInviteLink(ctx context.Context, token, username string, at time.Time) error {
	if _, err := d.pool.Exec(ctx, qActivateInviteLink, token, at, username); err != nil {
		return fmt.Errorf("mark invite link used: %w", err)
	}
	return nil
}

var qCreateInviteLink = register("CreateInviteLink", `
INSERT INTO web_invitelink (kind, delivery, email, wikidot_username, token, uidb64,
	created_at, activated_username, created_by_id, target_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, '', $8, $9)
RETURNING id`)

func (d *DB) CreateInviteLink(ctx context.Context, kind, delivery, email, wikidotName,
	token, uid string, createdBy *int64, target int64, at time.Time) (int64, error) {

	var id int64
	err := d.pool.QueryRow(ctx, qCreateInviteLink, kind, delivery, email, wikidotName,
		token, uid, at, createdBy, target).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("write invite link: %w", err)
	}
	return id, nil
}

const (
	TicketKind          = "ticket"
	MembershipApplyKind = "membershipapply"

	TicketPending = "pending"
)

var qCreateTicket = register("CreateTicket", `
INSERT INTO web_userticket (kind, subject, body, source_page, status, admin_notes, created_at, author_id)
VALUES ($1, $2, $3, $4, $5, '', $6, $7)
RETURNING id`)

func (d *DB) CreateTicket(ctx context.Context, kind, subject, body, page string, authorID int64, at time.Time) (int64, error) {
	var id int64
	err := d.pool.QueryRow(ctx, qCreateTicket, kind, subject, body, page, TicketPending, at, authorID).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("write ticket by %d: %w", authorID, err)
	}
	return id, nil
}
