package db

import (
	"context"
	"fmt"
	"time"
)

const (
	NotifyTicketResult = "ticket_result"
	ReportKind         = "report"
)

type OwnTicket struct {
	Kind       string
	ID         int64
	SiteID     int64
	Subject    string
	Status     string
	Reply      string
	CreatedAt  time.Time
	ReviewedAt *time.Time
}

const ownTickets = `
SELECT t.kind, t.id, coalesce(t.site_id, 0) AS site_id, t.subject, t.status, t.reply, t.created_at, t.reviewed_at
FROM web_userticket t WHERE t.author_id = $1
UNION ALL
SELECT 'report', r.id, coalesce(r.site_id, 0), coalesce(nullif(u.display_name, ''), u.username, ''), r.status, r.reply,
       r.created_at, r.reviewed_at
FROM web_userreport r LEFT JOIN web_user u ON u.id = r.reported_id WHERE r.reporter_id = $1`

var qOwnTickets = register("OwnTickets", `
SELECT kind, id, site_id, subject, status, reply, created_at, reviewed_at FROM (`+ownTickets+`) x
ORDER BY created_at DESC, id DESC
OFFSET $2 LIMIT $3`)

var qOwnTicketCount = register("OwnTicketCount", `SELECT count(*) FROM (`+ownTickets+`) x`)

func (d *DB) OwnTicketCount(ctx context.Context, userID int64) (int, error) {
	var total int
	if err := d.pool.QueryRow(ctx, qOwnTicketCount, userID).Scan(&total); err != nil {
		return 0, fmt.Errorf("count the tickets of user %d: %w", userID, err)
	}
	return total, nil
}

func (d *DB) OwnTickets(ctx context.Context, userID int64, offset, limit int) ([]OwnTicket, error) {
	rows, err := d.pool.Query(ctx, qOwnTickets, userID, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("list the tickets of user %d: %w", userID, err)
	}
	defer rows.Close()
	var out []OwnTicket
	for rows.Next() {
		var t OwnTicket
		if err := rows.Scan(&t.Kind, &t.ID, &t.SiteID, &t.Subject, &t.Status, &t.Reply, &t.CreatedAt, &t.ReviewedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
