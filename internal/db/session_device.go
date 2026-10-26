package db

import (
	"context"
	"fmt"
	"net/netip"
	"time"
)

const DeviceQuiet = 5 * time.Minute

type Device struct {
	ID         int64
	Key        string
	SiteID     *int64
	SiteTitle  string
	SignedInAt *time.Time
	SeenAt     time.Time
	Address    string
	Agent      string
}

// The session row is checked in the same statement, so a sign-out racing a
// request leaves nothing behind instead of failing on the foreign key.
var qSeenDevice = register("SeenDevice", `
INSERT INTO pwikit_session_device (session_key, user_id, site_id, signed_in_at, seen_at, address, agent)
SELECT $1::varchar, $2::bigint, $3::bigint, $4::timestamptz, $5::timestamptz, $6::inet, $7::text
WHERE EXISTS (SELECT 1 FROM django_session WHERE session_key = $1::varchar)
ON CONFLICT (session_key) DO UPDATE
SET seen_at = EXCLUDED.seen_at, address = EXCLUDED.address, agent = EXCLUDED.agent,
	signed_in_at = coalesce(EXCLUDED.signed_in_at, pwikit_session_device.signed_in_at)
WHERE pwikit_session_device.user_id = EXCLUDED.user_id
	AND (EXCLUDED.signed_in_at IS NOT NULL OR pwikit_session_device.seen_at < EXCLUDED.seen_at - $8::interval)`)

func (d *DB) SeenDevice(ctx context.Context, key string, userID int64, siteID *int64, signedIn bool,
	address *netip.Addr, agent string, at time.Time) error {
	var since *time.Time
	if signedIn {
		since = &at
	}
	var addr *string
	if address != nil && address.IsValid() {
		s := address.String()
		addr = &s
	}
	if len(agent) > 512 {
		agent = agent[:512]
	}
	if _, err := d.pool.Exec(ctx, qSeenDevice, key, userID, siteID, since, at, addr, agent, DeviceQuiet); err != nil {
		return fmt.Errorf("record device of user %d: %w", userID, err)
	}
	return nil
}

var qDevices = register("Devices", `
SELECT d.id, d.session_key, d.site_id, coalesce(s.title, ''), d.signed_in_at, d.seen_at,
	coalesce(host(d.address), ''), d.agent
FROM pwikit_session_device d
JOIN django_session j ON j.session_key = d.session_key
LEFT JOIN web_site s ON s.id = d.site_id
WHERE d.user_id = $1 AND j.expire_date > now()
ORDER BY d.seen_at DESC, d.id DESC`)

func (d *DB) Devices(ctx context.Context, userID int64) ([]Device, error) {
	rows, err := d.pool.Query(ctx, qDevices, userID)
	if err != nil {
		return nil, fmt.Errorf("list devices of user %d: %w", userID, err)
	}
	defer rows.Close()
	var out []Device
	for rows.Next() {
		var v Device
		if err := rows.Scan(&v.ID, &v.Key, &v.SiteID, &v.SiteTitle, &v.SignedInAt, &v.SeenAt, &v.Address, &v.Agent); err != nil {
			return nil, fmt.Errorf("scan device: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

var qEndDevice = register("EndDevice", `
DELETE FROM django_session
WHERE session_key IN (SELECT session_key FROM pwikit_session_device WHERE user_id = $1 AND id = $2)`)

func (d *DB) EndDevice(ctx context.Context, userID, id int64) (bool, error) {
	tag, err := d.pool.Exec(ctx, qEndDevice, userID, id)
	if err != nil {
		return false, fmt.Errorf("end device %d of user %d: %w", id, userID, err)
	}
	return tag.RowsAffected() > 0, nil
}

var qEndOtherDevices = register("EndOtherDevices", `
DELETE FROM django_session
WHERE session_key IN (SELECT session_key FROM pwikit_session_device WHERE user_id = $1 AND session_key <> $2)`)

func (d *DB) EndOtherDevices(ctx context.Context, userID int64, keep string) (int64, error) {
	tag, err := d.pool.Exec(ctx, qEndOtherDevices, userID, keep)
	if err != nil {
		return 0, fmt.Errorf("end other devices of user %d: %w", userID, err)
	}
	return tag.RowsAffected(), nil
}
