package db

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5"
)

type IPBan struct {
	ID      int64
	Network netip.Prefix
	Until   *time.Time
	Reason  string
	SetBy   string
	SetAt   time.Time
}

var qIPBans = register("IPBans", `
SELECT b.id, b.network::text, b.until, b.reason, coalesce(u.username, ''), b.set_at
FROM pwikit_ip_ban b LEFT JOIN web_user u ON u.id = b.set_by_id
WHERE b.site_id = $1
ORDER BY b.set_at DESC, b.id DESC`)

func (d *DB) IPBans(ctx context.Context, siteID int64) ([]IPBan, error) {
	rows, err := d.pool.Query(ctx, qIPBans, siteID)
	if err != nil {
		return nil, fmt.Errorf("list the address bans of site %d: %w", siteID, err)
	}
	defer rows.Close()

	var out []IPBan
	for rows.Next() {
		var one IPBan
		var network string
		if err := rows.Scan(&one.ID, &network, &one.Until, &one.Reason, &one.SetBy, &one.SetAt); err != nil {
			return nil, err
		}
		if one.Network, err = netip.ParsePrefix(network); err != nil {
			return nil, fmt.Errorf("read address ban %d: %w", one.ID, err)
		}
		out = append(out, one)
	}
	return out, rows.Err()
}

var qSetIPBan = register("SetIPBan", `
INSERT INTO pwikit_ip_ban (site_id, network, until, reason, set_by_id, set_at)
VALUES ($1, $2::text::cidr, $3, $4, $5, $6)
ON CONFLICT (site_id, network) DO UPDATE
SET until = EXCLUDED.until, reason = EXCLUDED.reason,
    set_by_id = EXCLUDED.set_by_id, set_at = EXCLUDED.set_at
RETURNING id`)

func (d *DB) SetIPBan(ctx context.Context, siteID int64, network netip.Prefix, until *time.Time,
	reason string, by *int64, at time.Time) (int64, error) {

	var id int64
	err := d.pool.QueryRow(ctx, qSetIPBan, siteID, network.String(), until, reason, by, at).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("ban %s on site %d: %w", network, siteID, err)
	}
	return id, nil
}

var qDeleteIPBan = register("DeleteIPBan", `
DELETE FROM pwikit_ip_ban WHERE id = $1 AND site_id = $2 RETURNING network::text`)

func (d *DB) DeleteIPBan(ctx context.Context, siteID, id int64) (string, error) {
	var network string
	err := d.pool.QueryRow(ctx, qDeleteIPBan, id, siteID).Scan(&network)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("lift address ban %d: %w", id, err)
	}
	return network, nil
}

var qIPBanned = register("IPBanned", `
SELECT EXISTS (
    SELECT 1 FROM pwikit_ip_ban
    WHERE site_id = $1 AND network >>= $2::text::inet AND (until IS NULL OR until > $3))`)

func (d *DB) IPBanned(ctx context.Context, siteID int64, addr netip.Addr, now time.Time) (bool, error) {
	var banned bool
	if err := d.pool.QueryRow(ctx, qIPBanned, siteID, addr.Unmap().String(), now).Scan(&banned); err != nil {
		return false, fmt.Errorf("check address %s on site %d: %w", addr, siteID, err)
	}
	return banned, nil
}
