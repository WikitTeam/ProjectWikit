package db

import (
	"context"
	"fmt"
	"net/netip"
	"time"
)

// A repeat from the same address inside this window writes nothing, so a busy
// session costs one row and then no further writes.
const addressQuiet = time.Hour

var qSeenAddress = register("SeenAddress", `
INSERT INTO pwikit_user_address (user_id, address, first_seen, last_seen)
VALUES ($1, $2, $3, $3)
ON CONFLICT (user_id, address) DO UPDATE
SET last_seen = EXCLUDED.last_seen, hits = pwikit_user_address.hits + 1
WHERE pwikit_user_address.last_seen < EXCLUDED.last_seen - $4::interval`)

func (d *DB) SeenAddress(ctx context.Context, userID int64, address *netip.Addr, at time.Time) error {
	if address == nil || !address.IsValid() {
		return nil
	}
	_, err := d.pool.Exec(ctx, qSeenAddress, userID, address.String(), at, addressQuiet)
	if err != nil {
		return fmt.Errorf("record address of user %d: %w", userID, err)
	}
	return nil
}

const (
	AdminCreated = "create"
	AdminChanged = "change"
	AdminDeleted = "delete"
)

type AdminNote struct {
	UserID *int64
	Name   string
	Action string
	Screen string
	Target string
	Label  string
	At     time.Time
}

var qWriteAdminNote = register("WriteAdminNote", `
INSERT INTO pwikit_admin_log (user_id, stale_name, action, screen, target, label, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)`)

func (d *DB) WriteAdminNote(ctx context.Context, n AdminNote) error {
	_, err := d.pool.Exec(ctx, qWriteAdminNote,
		n.UserID, n.Name, n.Action, n.Screen, n.Target, n.Label, n.At)
	if err != nil {
		return fmt.Errorf("record admin action %q on %q: %w", n.Action, n.Screen, err)
	}
	return nil
}

type AdminNoteRow struct {
	ID        int64
	User      string
	Stale     string
	Action    string
	Screen    string
	Target    string
	Label     string
	CreatedAt time.Time
}

var qAdminNotes = register("AdminNotes", `
SELECT l.id, coalesce(u.username, ''), l.stale_name, l.action, l.screen, l.target, l.label, l.created_at
FROM pwikit_admin_log l
LEFT JOIN web_user u ON u.id = l.user_id
WHERE $1 = '' OR l.screen = $1
ORDER BY l.created_at DESC, l.id DESC
LIMIT $2 OFFSET $3`)

var qAdminNoteScreens = register("AdminNoteScreens", `
SELECT DISTINCT screen FROM pwikit_admin_log ORDER BY 1`)

func (d *DB) AdminNotes(ctx context.Context, screen string, limit, offset int) ([]AdminNoteRow, error) {
	rows, err := d.pool.Query(ctx, qAdminNotes, screen, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list admin actions: %w", err)
	}
	defer rows.Close()

	var out []AdminNoteRow
	for rows.Next() {
		var one AdminNoteRow
		if err := rows.Scan(&one.ID, &one.User, &one.Stale, &one.Action,
			&one.Screen, &one.Target, &one.Label, &one.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, one)
	}
	return out, rows.Err()
}

func (d *DB) AdminNoteScreens(ctx context.Context) ([]string, error) {
	rows, err := d.pool.Query(ctx, qAdminNoteScreens)
	if err != nil {
		return nil, fmt.Errorf("list admin action screens: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var one string
		if err := rows.Scan(&one); err != nil {
			return nil, err
		}
		out = append(out, one)
	}
	return out, rows.Err()
}

var qClearAddresses = register("ClearAddresses", `DELETE FROM pwikit_user_address`)

var qClearUserAddresses = register("ClearUserAddresses", `
DELETE FROM pwikit_user_address WHERE user_id = $1`)

func (d *DB) ClearAddresses(ctx context.Context, userID *int64) (int64, error) {
	sql, args := qClearAddresses, []any{}
	if userID != nil {
		sql, args = qClearUserAddresses, []any{*userID}
	}
	tag, err := d.pool.Exec(ctx, sql, args...)
	if err != nil {
		return 0, fmt.Errorf("clear addresses: %w", err)
	}
	return tag.RowsAffected(), nil
}

var qAddressCount = register("AddressCount", `SELECT count(*) FROM pwikit_user_address`)

func (d *DB) AddressCount(ctx context.Context) (int, error) {
	var total int
	if err := d.pool.QueryRow(ctx, qAddressCount).Scan(&total); err != nil {
		return 0, fmt.Errorf("count addresses: %w", err)
	}
	return total, nil
}
