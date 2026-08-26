package db

import (
	"context"
	"fmt"
)

var qUnreadNotifications = register("UnreadNotifications", `
SELECT count(*)
FROM web_usernotificationmapping
WHERE recipient_id = $1 AND is_viewed = false`)

func (d *DB) UnreadNotifications(ctx context.Context, userID int64) (int, error) {
	var n int
	if err := d.pool.QueryRow(ctx, qUnreadNotifications, userID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count unread notifications of user %d: %w", userID, err)
	}
	return n, nil
}
