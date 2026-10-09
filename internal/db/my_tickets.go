package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
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
FROM web_userticket t WHERE t.author_id = $1 AND t.hidden_by_author_at IS NULL
UNION ALL
SELECT 'report', r.id, coalesce(r.site_id, 0), coalesce(nullif(u.display_name, ''), u.username, ''), r.status, r.reply,
       r.created_at, r.reviewed_at
FROM web_userreport r LEFT JOIN web_user u ON u.id = r.reported_id WHERE r.reporter_id = $1 AND r.hidden_by_reporter_at IS NULL`

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

type OwnTicketDetail struct {
	OwnTicket
	Body       string
	SourcePage string
	Messages   string
}

var qOwnTicketDetail = register("OwnTicketDetail", `
SELECT t.kind, t.id, coalesce(t.site_id, 0), t.subject, t.status, t.reply, t.created_at, t.reviewed_at,
       t.body, t.source_page, ''
FROM web_userticket t WHERE t.id = $1 AND t.author_id = $2 AND t.kind = $3 AND t.hidden_by_author_at IS NULL`)

var qOwnReportDetail = register("OwnReportDetail", `
SELECT 'report', r.id, coalesce(r.site_id, 0), coalesce(nullif(u.display_name, ''), u.username, ''), r.status, r.reply,
       r.created_at, r.reviewed_at, r.reason, '', coalesce(r.reported_messages::text, '')
FROM web_userreport r LEFT JOIN web_user u ON u.id = r.reported_id
WHERE r.id = $1 AND r.reporter_id = $2 AND r.hidden_by_reporter_at IS NULL`)

func (d *DB) OwnTicketDetail(ctx context.Context, userID int64, kind string, id int64) (OwnTicketDetail, error) {
	var row pgx.Row
	if kind == ReportKind {
		row = d.pool.QueryRow(ctx, qOwnReportDetail, id, userID)
	} else {
		row = d.pool.QueryRow(ctx, qOwnTicketDetail, id, userID, kind)
	}
	var t OwnTicketDetail
	err := row.Scan(&t.Kind, &t.ID, &t.SiteID, &t.Subject, &t.Status, &t.Reply, &t.CreatedAt, &t.ReviewedAt,
		&t.Body, &t.SourcePage, &t.Messages)
	if errors.Is(err, pgx.ErrNoRows) {
		return OwnTicketDetail{}, ErrNotFound
	}
	if err != nil {
		return OwnTicketDetail{}, fmt.Errorf("read %s %d of user %d: %w", kind, id, userID, err)
	}
	return t, nil
}

type OwnTicketRef struct {
	Kind string
	ID   int64
}

var qHideOwnTickets = register("HideOwnTickets", `
UPDATE web_userticket SET hidden_by_author_at = $3
WHERE author_id = $1 AND id = ANY($2) AND hidden_by_author_at IS NULL`)

var qHideOwnReports = register("HideOwnReports", `
UPDATE web_userreport SET hidden_by_reporter_at = $3
WHERE reporter_id = $1 AND id = ANY($2) AND hidden_by_reporter_at IS NULL`)

var qHideAllOwnTickets = register("HideAllOwnTickets", `
UPDATE web_userticket SET hidden_by_author_at = $2 WHERE author_id = $1 AND hidden_by_author_at IS NULL`)

var qHideAllOwnReports = register("HideAllOwnReports", `
UPDATE web_userreport SET hidden_by_reporter_at = $2 WHERE reporter_id = $1 AND hidden_by_reporter_at IS NULL`)

func (d *DB) HideOwnTickets(ctx context.Context, userID int64, refs []OwnTicketRef, at time.Time) (int64, error) {
	var tickets, reports []int64
	for _, one := range refs {
		if one.Kind == ReportKind {
			reports = append(reports, one.ID)
		} else {
			tickets = append(tickets, one.ID)
		}
	}
	return d.hideOwn(ctx, userID, at, qHideOwnTickets, qHideOwnReports, tickets, reports)
}

func (d *DB) HideAllOwnTickets(ctx context.Context, userID int64, at time.Time) (int64, error) {
	return d.hideOwn(ctx, userID, at, qHideAllOwnTickets, qHideAllOwnReports, nil, nil)
}

func (d *DB) hideOwn(ctx context.Context, userID int64, at time.Time, ticketQuery, reportQuery string, tickets, reports []int64) (int64, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin hiding the tickets of user %d: %w", userID, err)
	}
	defer tx.Rollback(context.WithoutCancel(ctx))
	var hidden int64
	for _, q := range []struct {
		query string
		ids   []int64
	}{{ticketQuery, tickets}, {reportQuery, reports}} {
		args := []any{userID, at}
		if q.query == qHideOwnTickets || q.query == qHideOwnReports {
			if len(q.ids) == 0 {
				continue
			}
			args = []any{userID, q.ids, at}
		}
		tag, err := tx.Exec(ctx, q.query, args...)
		if err != nil {
			return 0, fmt.Errorf("hide the tickets of user %d: %w", userID, err)
		}
		hidden += tag.RowsAffected()
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit hiding the tickets of user %d: %w", userID, err)
	}
	return hidden, nil
}
