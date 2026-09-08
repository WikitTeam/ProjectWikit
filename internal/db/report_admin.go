package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	ReportReviewed  = "reviewed"
	ReportDismissed = "dismissed"
)

type ReportRow struct {
	ID         int64
	Reporter   string
	Reported   string
	Reason     string
	Messages   string
	Status     string
	AdminNotes string
	CreatedAt  time.Time
	ReviewedAt *time.Time
	ReviewedBy string
}

const reportColumns = `r.id, coalesce(rep.username, ''), coalesce(tgt.username, ''),
	r.reason, r.reported_messages::text, r.status, r.admin_notes, r.created_at,
	r.reviewed_at, coalesce(rev.username, '')`

const reportJoins = `
FROM web_userreport r
LEFT JOIN web_user rep ON rep.id = r.reporter_id
LEFT JOIN web_user tgt ON tgt.id = r.reported_id
LEFT JOIN web_user rev ON rev.id = r.reviewed_by_id`

var qAdminReports = register("AdminReports", `
SELECT `+reportColumns+reportJoins+`
WHERE ($1 = '' OR r.status = $1) AND r.site_id = $4
ORDER BY r.created_at DESC, r.id DESC
LIMIT $2 OFFSET $3`)

var qAdminReportCount = register("AdminReportCount", `
SELECT count(*) FROM web_userreport r WHERE ($1 = '' OR r.status = $1) AND r.site_id = $2`)

func scanReport(row pgx.Row, r *ReportRow) error {
	return row.Scan(&r.ID, &r.Reporter, &r.Reported, &r.Reason, &r.Messages,
		&r.Status, &r.AdminNotes, &r.CreatedAt, &r.ReviewedAt, &r.ReviewedBy)
}

func (d *DB) AdminReports(ctx context.Context, siteID int64, status string, limit, offset int) ([]ReportRow, int, error) {
	var total int
	if err := d.pool.QueryRow(ctx, qAdminReportCount, status, siteID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count reports: %w", err)
	}
	rows, err := d.pool.Query(ctx, qAdminReports, status, limit, offset, siteID)
	if err != nil {
		return nil, 0, fmt.Errorf("list reports: %w", err)
	}
	defer rows.Close()

	var out []ReportRow
	for rows.Next() {
		var r ReportRow
		if err := scanReport(rows, &r); err != nil {
			return nil, 0, err
		}
		out = append(out, r)
	}
	return out, total, rows.Err()
}

var qAdminReport = register("AdminReport", `SELECT `+reportColumns+reportJoins+` WHERE r.id = $1 AND r.site_id = $2`)

func (d *DB) AdminReport(ctx context.Context, siteID, id int64) (ReportRow, error) {
	var r ReportRow
	err := scanReport(d.pool.QueryRow(ctx, qAdminReport, id, siteID), &r)
	if errors.Is(err, pgx.ErrNoRows) {
		return ReportRow{}, ErrNotFound
	}
	if err != nil {
		return ReportRow{}, fmt.Errorf("read report %d: %w", id, err)
	}
	return r, nil
}

var qReviewReport = register("ReviewReport", `
UPDATE web_userreport SET status = $2, admin_notes = $3, reviewed_at = $4, reviewed_by_id = $5
WHERE id = $1 AND site_id = $6`)

func (d *DB) ReviewReport(ctx context.Context, id int64, status, notes string, by int64, at time.Time) error {
	var reviewedAt *time.Time
	var reviewer *int64
	if status != ReportPending {
		reviewedAt, reviewer = &at, &by
	}
	if _, err := d.pool.Exec(ctx, qReviewReport, id, status, notes, reviewedAt, reviewer); err != nil {
		return fmt.Errorf("review report %d: %w", id, err)
	}
	return nil
}
