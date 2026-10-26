package db

import (
	"context"
	"fmt"
	"time"
)

var qIsSiteSuperuser = register("IsSiteSuperuser", `
SELECT EXISTS (SELECT 1 FROM pwikit_site_superuser WHERE site_id = $1 AND user_id = $2)`)

func (d *DB) IsSiteSuperuser(ctx context.Context, siteID, userID int64) (bool, error) {
	if siteID == 0 {
		return false, nil
	}
	var yes bool
	if err := d.pool.QueryRow(ctx, qIsSiteSuperuser, siteID, userID).Scan(&yes); err != nil {
		return false, fmt.Errorf("check site superuser %d on %d: %w", userID, siteID, err)
	}
	return yes, nil
}

var qSiteSuperusers = register("SiteSuperusers", `
SELECT user_id FROM pwikit_site_superuser WHERE site_id = $1 AND user_id = ANY($2)`)

func (d *DB) SiteSuperusers(ctx context.Context, siteID int64, among []int64) (map[int64]bool, error) {
	out := map[int64]bool{}
	if siteID == 0 || len(among) == 0 {
		return out, nil
	}
	rows, err := d.pool.Query(ctx, qSiteSuperusers, siteID, among)
	if err != nil {
		return nil, fmt.Errorf("list site superusers of %d: %w", siteID, err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

var (
	qGrantSiteSuperuser = register("GrantSiteSuperuser", `
INSERT INTO pwikit_site_superuser (site_id, user_id, granted_by_id, granted_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (site_id, user_id) DO NOTHING`)

	qRevokeSiteSuperuser = register("RevokeSiteSuperuser", `
DELETE FROM pwikit_site_superuser WHERE site_id = $1 AND user_id = $2`)
)

func (d *DB) SetSiteSuperuser(ctx context.Context, siteID, userID int64, on bool, by int64, at time.Time) (bool, error) {
	sql, args := qRevokeSiteSuperuser, []any{siteID, userID}
	if on {
		sql, args = qGrantSiteSuperuser, []any{siteID, userID, by, at}
	}
	tag, err := d.pool.Exec(ctx, sql, args...)
	if err != nil {
		return false, fmt.Errorf("set site superuser %d on %d: %w", userID, siteID, err)
	}
	return tag.RowsAffected() > 0, nil
}
