package db

import (
	"context"
	"fmt"
	"time"
)

type MemberSanction struct {
	Kind   string
	Until  *time.Time
	Reason string
	SetBy  *int64
	SetAt  time.Time
}

var qActiveSanctions = register("ActiveSanctions", `
SELECT kind
FROM pwikit_member_sanction
WHERE site_id = $1 AND user_id = $2 AND (until IS NULL OR until > $3)`)

func (d *DB) ActiveSanctions(ctx context.Context, siteID, userID int64, now time.Time) ([]string, error) {
	rows, err := d.pool.Query(ctx, qActiveSanctions, siteID, userID, now)
	if err != nil {
		return nil, fmt.Errorf("read the sanctions of user %d: %w", userID, err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var kind string
		if err := rows.Scan(&kind); err != nil {
			return nil, err
		}
		out = append(out, kind)
	}
	return out, rows.Err()
}

var qMemberSanctions = register("MemberSanctions", `
SELECT kind, until, reason, set_by_id, set_at
FROM pwikit_member_sanction
WHERE site_id = $1 AND user_id = $2
ORDER BY kind`)

func (d *DB) MemberSanctions(ctx context.Context, siteID, userID int64) ([]MemberSanction, error) {
	rows, err := d.pool.Query(ctx, qMemberSanctions, siteID, userID)
	if err != nil {
		return nil, fmt.Errorf("read the sanctions of user %d: %w", userID, err)
	}
	defer rows.Close()

	var out []MemberSanction
	for rows.Next() {
		var one MemberSanction
		if err := rows.Scan(&one.Kind, &one.Until, &one.Reason, &one.SetBy, &one.SetAt); err != nil {
			return nil, err
		}
		out = append(out, one)
	}
	return out, rows.Err()
}

var qSetSanction = register("SetSanction", `
INSERT INTO pwikit_member_sanction (site_id, user_id, kind, until, reason, set_by_id, set_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (site_id, user_id, kind) DO UPDATE
SET until = EXCLUDED.until, reason = EXCLUDED.reason,
    set_by_id = EXCLUDED.set_by_id, set_at = EXCLUDED.set_at`)

func (d *DB) SetSanction(ctx context.Context, siteID, userID int64, kind string, until *time.Time,
	reason string, by *int64, at time.Time) error {

	if _, err := d.pool.Exec(ctx, qSetSanction, siteID, userID, kind, until, reason, by, at); err != nil {
		return fmt.Errorf("record the %s of user %d: %w", kind, userID, err)
	}
	return nil
}

var qClearSanction = register("ClearSanction", `
DELETE FROM pwikit_member_sanction WHERE site_id = $1 AND user_id = $2 AND kind = $3`)

func (d *DB) ClearSanction(ctx context.Context, siteID, userID int64, kind string) error {
	if _, err := d.pool.Exec(ctx, qClearSanction, siteID, userID, kind); err != nil {
		return fmt.Errorf("lift the %s of user %d: %w", kind, userID, err)
	}
	return nil
}
