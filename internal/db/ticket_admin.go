package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	TicketApproved = "approved"
	TicketRejected = "rejected"
	TicketClosed   = "closed"
)

type TicketRow struct {
	ID         int64
	Kind       string
	Author     string
	Subject    string
	Body       string
	SourcePage string
	Status     string
	AdminNotes string
	CreatedAt  time.Time
	ReviewedAt *time.Time
	ReviewedBy string
	GrantedID  *int64
}

const ticketColumns = `t.id, t.kind, coalesce(a.username, ''), t.subject, t.body, t.source_page,
	t.status, t.admin_notes, t.created_at, t.reviewed_at, coalesce(rev.username, ''), t.granted_role_id`

const ticketJoins = `
FROM web_userticket t
LEFT JOIN web_user a ON a.id = t.author_id
LEFT JOIN web_user rev ON rev.id = t.reviewed_by_id`

var qAdminTickets = register("AdminTickets", `
SELECT `+ticketColumns+ticketJoins+`
WHERE t.kind = $1 AND ($2 = '' OR t.status = $2)
ORDER BY t.created_at DESC, t.id DESC
LIMIT $3 OFFSET $4`)

var qAdminTicketCount = register("AdminTicketCount", `
SELECT count(*) FROM web_userticket t WHERE t.kind = $1 AND ($2 = '' OR t.status = $2)`)

func scanTicket(row pgx.Row, t *TicketRow) error {
	return row.Scan(&t.ID, &t.Kind, &t.Author, &t.Subject, &t.Body, &t.SourcePage,
		&t.Status, &t.AdminNotes, &t.CreatedAt, &t.ReviewedAt, &t.ReviewedBy, &t.GrantedID)
}

func (d *DB) AdminTickets(ctx context.Context, kind, status string, limit, offset int) ([]TicketRow, int, error) {
	var total int
	if err := d.pool.QueryRow(ctx, qAdminTicketCount, kind, status).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tickets: %w", err)
	}
	rows, err := d.pool.Query(ctx, qAdminTickets, kind, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list tickets: %w", err)
	}
	defer rows.Close()

	var out []TicketRow
	for rows.Next() {
		var t TicketRow
		if err := scanTicket(rows, &t); err != nil {
			return nil, 0, err
		}
		out = append(out, t)
	}
	return out, total, rows.Err()
}

var qAdminTicket = register("AdminTicket", `SELECT `+ticketColumns+ticketJoins+` WHERE t.id = $1`)

func (d *DB) AdminTicket(ctx context.Context, id int64) (TicketRow, error) {
	var t TicketRow
	err := scanTicket(d.pool.QueryRow(ctx, qAdminTicket, id), &t)
	if errors.Is(err, pgx.ErrNoRows) {
		return TicketRow{}, ErrNotFound
	}
	if err != nil {
		return TicketRow{}, fmt.Errorf("read ticket %d: %w", id, err)
	}
	return t, nil
}

var qReviewTicket = register("ReviewTicket", `
UPDATE web_userticket SET status=$2, admin_notes=$3, reviewed_at=$4, reviewed_by_id=$5, granted_role_id=$6
WHERE id=$1`)

var qGrantTicketRole = register("GrantTicketRole", `
INSERT INTO web_user_roles (user_id, role_id)
SELECT t.author_id, $2 FROM web_userticket t WHERE t.id = $1 AND t.author_id IS NOT NULL
ON CONFLICT DO NOTHING`)

func (d *DB) ReviewTicket(ctx context.Context, id int64, status, notes string, by int64, role *int64, at time.Time) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin reviewing ticket %d: %w", id, err)
	}
	defer tx.Rollback(context.WithoutCancel(ctx))

	var reviewedAt *time.Time
	var reviewer *int64
	if status != TicketPending {
		reviewedAt, reviewer = &at, &by
	}
	if _, err := tx.Exec(ctx, qReviewTicket, id, status, notes, reviewedAt, reviewer, role); err != nil {
		return fmt.Errorf("review ticket %d: %w", id, err)
	}
	if status == TicketApproved && role != nil {
		if _, err := tx.Exec(ctx, qGrantTicketRole, id, *role); err != nil {
			return fmt.Errorf("grant the role of ticket %d: %w", id, err)
		}
	}
	return tx.Commit(ctx)
}

type InviteRow struct {
	ID                int64
	Kind              string
	Delivery          string
	Email             string
	WikidotUsername   string
	Token             string
	UIDB64            string
	CreatedAt         time.Time
	ActivatedAt       *time.Time
	ActivatedUsername string
}

var qAdminInvites = register("AdminInvites", `
SELECT id, kind, delivery, email, wikidot_username, token, uidb64,
       created_at, activated_at, activated_username
FROM web_invitelink ORDER BY created_at DESC, id DESC LIMIT $1 OFFSET $2`)

var qAdminInviteCount = register("AdminInviteCount", `SELECT count(*) FROM web_invitelink`)

func (d *DB) AdminInvites(ctx context.Context, limit, offset int) ([]InviteRow, int, error) {
	var total int
	if err := d.pool.QueryRow(ctx, qAdminInviteCount).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count invite links: %w", err)
	}
	rows, err := d.pool.Query(ctx, qAdminInvites, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list invite links: %w", err)
	}
	defer rows.Close()

	var out []InviteRow
	for rows.Next() {
		var i InviteRow
		err := rows.Scan(&i.ID, &i.Kind, &i.Delivery, &i.Email, &i.WikidotUsername,
			&i.Token, &i.UIDB64, &i.CreatedAt, &i.ActivatedAt, &i.ActivatedUsername)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, i)
	}
	return out, total, rows.Err()
}

var qDeleteInvite = register("DeleteInvite", `DELETE FROM web_invitelink WHERE id = $1 AND activated_at IS NULL`)

func (d *DB) DeleteInvite(ctx context.Context, id int64) error {
	if _, err := d.pool.Exec(ctx, qDeleteInvite, id); err != nil {
		return fmt.Errorf("delete invite link %d: %w", id, err)
	}
	return nil
}
