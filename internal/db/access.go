package db

import (
	"context"
	"fmt"
	"time"
)

// Everyone else on the instance answers the same as a plain signed-in member,
// so these are the only accounts whose answer can differ from it.
var qAccessCandidates = register("AccessCandidates", `
SELECT id FROM (
	SELECT ur.user_id AS id
	FROM web_user_roles ur
	JOIN web_role r ON r.id = ur.role_id
	WHERE r.site_id = $1
	UNION
	SELECT user_id FROM pwikit_site_superuser WHERE site_id = $1
	UNION
	SELECT user_id FROM pwikit_member_sanction WHERE site_id = $1 AND (until IS NULL OR until > $2)
	UNION
	SELECT id FROM web_user WHERE is_superuser
) candidates
ORDER BY id
LIMIT $3`)

func (d *DB) AccessCandidates(ctx context.Context, siteID int64, now time.Time, limit int) ([]int64, error) {
	rows, err := d.pool.Query(ctx, qAccessCandidates, siteID, now, limit)
	if err != nil {
		return nil, fmt.Errorf("list accounts with roles on site %d: %w", siteID, err)
	}
	defer rows.Close()

	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
