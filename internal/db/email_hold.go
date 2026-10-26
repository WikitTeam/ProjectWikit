package db

import (
	"context"
	"fmt"
	"time"
)

var qHoldEmail = register("HoldEmail", `
UPDATE web_user SET held_email = $2, held_until = NULL, held_site_id = $3
WHERE id = $1`)

func (d *DB) HoldEmail(ctx context.Context, id int64, email string, siteID int64) error {
	if _, err := d.pool.Exec(ctx, qHoldEmail, id, email, siteID); err != nil {
		return fmt.Errorf("hold email of %d: %w", id, err)
	}
	return nil
}

var qConfirmEmailHold = register("ConfirmEmailHold", `
UPDATE web_user SET held_until = $3
WHERE id = $1 AND held_email <> '' AND lower(held_email) = lower($2) AND held_until IS NULL`)

func (d *DB) ConfirmEmailHold(ctx context.Context, id int64, email string, until time.Time) (bool, error) {
	tag, err := d.pool.Exec(ctx, qConfirmEmailHold, id, email, until)
	if err != nil {
		return false, fmt.Errorf("confirm held email of %d: %w", id, err)
	}
	return tag.RowsAffected() > 0, nil
}

var qDropEmailHold = register("DropEmailHold", `
UPDATE web_user SET held_email = '', held_until = NULL, held_site_id = NULL
WHERE id = $1 AND held_email <> '' AND lower(held_email) = lower($2)`)

func (d *DB) DropEmailHold(ctx context.Context, id int64, email string) (bool, error) {
	tag, err := d.pool.Exec(ctx, qDropEmailHold, id, email)
	if err != nil {
		return false, fmt.Errorf("drop held email of %d: %w", id, err)
	}
	return tag.RowsAffected() > 0, nil
}

type SettledEmail struct {
	UserID   int64
	Username string
	From     string
	To       string
	SiteID   *int64
	Taken    bool
}

var (
	qDueEmailHolds = register("DueEmailHolds", `
SELECT id, username, email, held_email, held_site_id
FROM web_user
WHERE held_until IS NOT NULL AND held_until <= $1
ORDER BY id
FOR UPDATE SKIP LOCKED`)

	qApplyEmailHold = register("ApplyEmailHold", `
UPDATE web_user
SET previous_email = email, email = held_email, pending_email = '',
	email_verified_at = $2, email_changed_at = $2,
	held_email = '', held_until = NULL, held_site_id = NULL
WHERE id = $1`)
)

// Rows are locked and skipped rather than waited on, so two requests settling
// at once never apply the same hold twice.
func (d *DB) SettleEmailHolds(ctx context.Context, now time.Time) ([]SettledEmail, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin settling held emails: %w", err)
	}
	defer tx.Rollback(context.WithoutCancel(ctx))

	rows, err := tx.Query(ctx, qDueEmailHolds, now)
	if err != nil {
		return nil, fmt.Errorf("list due held emails: %w", err)
	}
	var due []SettledEmail
	for rows.Next() {
		var one SettledEmail
		if err := rows.Scan(&one.UserID, &one.Username, &one.From, &one.To, &one.SiteID); err != nil {
			rows.Close()
			return nil, err
		}
		due = append(due, one)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range due {
		one := &due[i]
		if err := tx.QueryRow(ctx, qVerifiedEmailTaken, one.To, one.UserID).Scan(&one.Taken); err != nil {
			return nil, fmt.Errorf("check held email of %d: %w", one.UserID, err)
		}
		if one.Taken {
			if _, err := tx.Exec(ctx, qDropEmailHold, one.UserID, one.To); err != nil {
				return nil, fmt.Errorf("drop held email of %d: %w", one.UserID, err)
			}
			continue
		}
		if _, err := tx.Exec(ctx, qApplyEmailHold, one.UserID, now); err != nil {
			return nil, fmt.Errorf("apply held email of %d: %w", one.UserID, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit settled emails: %w", err)
	}
	return due, nil
}
