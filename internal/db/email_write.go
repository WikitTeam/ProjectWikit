package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type AccountEmail struct {
	Email      string
	Pending    string
	Previous   string
	VerifiedAt *time.Time
	ChangedAt  *time.Time
}

var qAccountEmail = register("AccountEmail", `
SELECT email, pending_email, previous_email, email_verified_at, email_changed_at
FROM web_user
WHERE id = $1`)

func (d *DB) AccountEmail(ctx context.Context, id int64) (AccountEmail, error) {
	var a AccountEmail
	err := d.pool.QueryRow(ctx, qAccountEmail, id).
		Scan(&a.Email, &a.Pending, &a.Previous, &a.VerifiedAt, &a.ChangedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return AccountEmail{}, ErrNotFound
	}
	if err != nil {
		return AccountEmail{}, fmt.Errorf("read account email of %d: %w", id, err)
	}
	return a, nil
}

var qVerifiedEmailTaken = register("VerifiedEmailTaken", `
SELECT EXISTS (
	SELECT 1 FROM web_user
	WHERE lower(email) = lower($1) AND email_verified_at IS NOT NULL AND id <> $2)`)

func (d *DB) VerifiedEmailTaken(ctx context.Context, email string, except int64) (bool, error) {
	var taken bool
	if err := d.pool.QueryRow(ctx, qVerifiedEmailTaken, email, except).Scan(&taken); err != nil {
		return false, fmt.Errorf("check email %q: %w", email, err)
	}
	return taken, nil
}

var qMarkEmailVerified = register("MarkEmailVerified", `
UPDATE web_user
SET email_verified_at = $2, pending_email = ''
WHERE id = $1 AND lower(email) = lower($3)`)

func (d *DB) MarkEmailVerified(ctx context.Context, id int64, email string, at time.Time) (bool, error) {
	tag, err := d.pool.Exec(ctx, qMarkEmailVerified, id, at, email)
	if err != nil {
		return false, fmt.Errorf("verify email of %d: %w", id, err)
	}
	return tag.RowsAffected() > 0, nil
}

var qSetPendingEmail = register("SetPendingEmail", `
UPDATE web_user SET pending_email = $2 WHERE id = $1`)

func (d *DB) SetPendingEmail(ctx context.Context, id int64, email string) error {
	if _, err := d.pool.Exec(ctx, qSetPendingEmail, id, email); err != nil {
		return fmt.Errorf("store pending email of %d: %w", id, err)
	}
	return nil
}

var qApplyPendingEmail = register("ApplyPendingEmail", `
UPDATE web_user
SET previous_email = email, email = pending_email, pending_email = '',
	email_verified_at = $2, email_changed_at = $2
WHERE id = $1 AND lower(pending_email) = lower($3)`)

func (d *DB) ApplyPendingEmail(ctx context.Context, id int64, pending string, at time.Time) (bool, error) {
	tag, err := d.pool.Exec(ctx, qApplyPendingEmail, id, at, pending)
	if err != nil {
		return false, fmt.Errorf("apply pending email of %d: %w", id, err)
	}
	return tag.RowsAffected() > 0, nil
}

var qRevertEmail = register("RevertEmail", `
UPDATE web_user
SET email = previous_email, previous_email = '', pending_email = '',
	email_verified_at = $2, email_changed_at = $2
WHERE id = $1 AND previous_email <> '' AND lower(previous_email) = lower($3)`)

func (d *DB) RevertEmail(ctx context.Context, id int64, previous string, at time.Time) (bool, error) {
	tag, err := d.pool.Exec(ctx, qRevertEmail, id, at, previous)
	if err != nil {
		return false, fmt.Errorf("revert email of %d: %w", id, err)
	}
	return tag.RowsAffected() > 0, nil
}

var qSetUsername = register("SetUsername", `
UPDATE web_user
SET username = $2, display_name = $3, username_changed_at = $4
WHERE id = $1`)

func (d *DB) SetUsername(ctx context.Context, id int64, username, displayName string, at time.Time) error {
	var display *string
	if displayName != "" {
		display = &displayName
	}
	if _, err := d.pool.Exec(ctx, qSetUsername, id, username, display, at); err != nil {
		return fmt.Errorf("rename user %d: %w", id, err)
	}
	return nil
}

var qUsernameChangedAt = register("UsernameChangedAt", `
SELECT username_changed_at FROM web_user WHERE id = $1`)

func (d *DB) UsernameChangedAt(ctx context.Context, id int64) (*time.Time, error) {
	var at *time.Time
	err := d.pool.QueryRow(ctx, qUsernameChangedAt, id).Scan(&at)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read rename time of %d: %w", id, err)
	}
	return at, nil
}
