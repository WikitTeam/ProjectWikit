package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const NotifyAnnouncement = "announcement"

// A site's audience is whoever holds a role on it. The built-in roles live on no
// row, so a visitor who never joined anything is not counted. Bots, unclaimed
// imports and disabled accounts cannot read it.
var qAnnouncementAudience = register("AnnouncementAudience", `
SELECT u.id
FROM web_user u
WHERE u.type <> 'bot' AND u.type <> 'system' AND u.password NOT LIKE '!%'
  AND CASE WHEN u.inactive_until IS NULL THEN u.is_active ELSE u.inactive_until <= now() END
  AND ($1::bigint IS NULL
    OR EXISTS (SELECT 1 FROM web_user_roles ur JOIN web_role r ON r.id = ur.role_id
               WHERE ur.user_id = u.id AND r.site_id = $1))
ORDER BY u.id`)

func (d *DB) AnnouncementAudience(ctx context.Context, siteID *int64) ([]int64, error) {
	rows, err := d.pool.Query(ctx, qAnnouncementAudience, siteID)
	if err != nil {
		return nil, fmt.Errorf("read the announcement audience: %w", err)
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan the announcement audience: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

type Announcement struct {
	ID       int64
	Title    string
	Author   string
	Instance bool
	SentAt   time.Time
}

var qAnnouncements = register("Announcements", `
SELECT n.id, coalesce(n.meta->>'title', ''), coalesce(n.meta->>'author', ''), n.site_id IS NULL, n.created_at
FROM web_usernotification n
WHERE n.type = 'announcement' AND (n.site_id = $1 OR n.site_id IS NULL)
ORDER BY n.created_at DESC
LIMIT 20`)

func (d *DB) Announcements(ctx context.Context, siteID int64) ([]Announcement, error) {
	rows, err := d.pool.Query(ctx, qAnnouncements, siteID)
	if err != nil {
		return nil, fmt.Errorf("list announcements of site %d: %w", siteID, err)
	}
	defer rows.Close()
	var out []Announcement
	for rows.Next() {
		var a Announcement
		if err := rows.Scan(&a.ID, &a.Title, &a.Author, &a.Instance, &a.SentAt); err != nil {
			return nil, fmt.Errorf("scan announcement: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

var qAnnouncement = register("Announcement", `
SELECT n.id, coalesce(n.meta->>'title', ''), coalesce(n.meta->>'author', ''), n.site_id IS NULL, n.created_at,
       coalesce(n.meta->>'message', '')
FROM web_usernotification n
WHERE n.id = $2 AND n.type = 'announcement' AND (n.site_id = $1 OR n.site_id IS NULL)`)

// Announcement answers with ErrNotFound for one sent on another site, which this
// site's staff have no business reading.
func (d *DB) Announcement(ctx context.Context, siteID, id int64) (Announcement, string, error) {
	var a Announcement
	var message string
	err := d.pool.QueryRow(ctx, qAnnouncement, siteID, id).Scan(&a.ID, &a.Title, &a.Author, &a.Instance, &a.SentAt, &message)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, "", ErrNotFound
	}
	if err != nil {
		return a, "", fmt.Errorf("read announcement %d: %w", id, err)
	}
	return a, message, nil
}
