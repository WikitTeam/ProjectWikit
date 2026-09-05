package db

import (
	"context"
	"fmt"
	"time"
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

type Notification struct {
	ID        int64
	Type      string
	Meta      []byte
	CreatedAt time.Time
	IsViewed  bool
}

// A null kind list asks for every type, which is what the unfiltered view wants.
var qNotificationsOf = register("NotificationsOf", `
SELECT n.id, n.type, n.meta, n.created_at, m.is_viewed
FROM web_usernotificationmapping m
JOIN web_usernotification n ON n.id = m.notification_id
WHERE m.recipient_id = $1
  AND ($2::bigint IS NULL OR n.id < $2)
  AND (NOT $3 OR m.is_viewed = false)
  AND ($4::text[] IS NULL OR n.type = ANY($4))
ORDER BY n.id DESC
LIMIT $5`)

func (d *DB) NotificationsOf(ctx context.Context, userID int64, cursor *int64,
	unread bool, kinds []string, limit int) ([]Notification, error) {

	var kindFilter any
	if len(kinds) > 0 {
		kindFilter = kinds
	}
	rows, err := d.pool.Query(ctx, qNotificationsOf, userID, cursor, unread, kindFilter, limit)
	if err != nil {
		return nil, fmt.Errorf("list notifications of user %d: %w", userID, err)
	}
	defer rows.Close()

	var out []Notification
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.Type, &n.Meta, &n.CreatedAt, &n.IsViewed); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list notifications of user %d: %w", userID, err)
	}
	return out, nil
}

var qMarkNotificationsViewed = register("MarkNotificationsViewed", `
UPDATE web_usernotificationmapping SET is_viewed = true
WHERE recipient_id = $1 AND notification_id = ANY($2)`)

func (d *DB) MarkNotificationsViewed(ctx context.Context, userID int64, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	if _, err := d.pool.Exec(ctx, qMarkNotificationsViewed, userID, ids); err != nil {
		return fmt.Errorf("mark notifications of user %d: %w", userID, err)
	}
	return nil
}

// Only the reader's own row goes, so a notification sent to several people
// survives for everyone who has not cleared it.
var qDeleteNotifications = register("DeleteNotifications", `
DELETE FROM web_usernotificationmapping
WHERE recipient_id = $1 AND notification_id = ANY($2)`)

func (d *DB) DeleteNotifications(ctx context.Context, userID int64, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	tag, err := d.pool.Exec(ctx, qDeleteNotifications, userID, ids)
	if err != nil {
		return 0, fmt.Errorf("delete notifications of user %d: %w", userID, err)
	}
	return tag.RowsAffected(), nil
}

var qDeleteAllNotifications = register("DeleteAllNotifications", `
DELETE FROM web_usernotificationmapping
WHERE recipient_id = $1
  AND ($2::text[] IS NULL OR notification_id IN (
	SELECT id FROM web_usernotification WHERE type = ANY($2)))`)

func (d *DB) DeleteAllNotifications(ctx context.Context, userID int64, kinds []string) (int64, error) {
	var kindFilter any
	if len(kinds) > 0 {
		kindFilter = kinds
	}
	tag, err := d.pool.Exec(ctx, qDeleteAllNotifications, userID, kindFilter)
	if err != nil {
		return 0, fmt.Errorf("clear notifications of user %d: %w", userID, err)
	}
	return tag.RowsAffected(), nil
}
