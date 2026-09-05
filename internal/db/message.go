package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type DirectMessage struct {
	ID          int64
	SenderID    int64
	RecipientID int64
	Body        string
	CreatedAt   time.Time
	IsRead      bool
}

type Conversation struct {
	PartnerID int64
	Last      DirectMessage
	Unread    int
}

var qConversations = register("Conversations", `
WITH mine AS (
	SELECT id,
		CASE WHEN sender_id = $1 THEN recipient_id ELSE sender_id END AS partner_id
	FROM web_directmessage
	WHERE sender_id = $1 OR recipient_id = $1
), newest AS (
	SELECT partner_id, max(id) AS last_id
	FROM mine
	GROUP BY partner_id
)
SELECT n.partner_id, m.id, m.sender_id, m.recipient_id, m.body, m.created_at, m.is_read,
	(SELECT count(*) FROM web_directmessage u
		WHERE u.sender_id = n.partner_id AND u.recipient_id = $1 AND NOT u.is_read)
FROM newest n
JOIN web_directmessage m ON m.id = n.last_id
ORDER BY m.created_at DESC`)

func (d *DB) Conversations(ctx context.Context, userID int64) ([]Conversation, error) {
	rows, err := d.pool.Query(ctx, qConversations, userID)
	if err != nil {
		return nil, fmt.Errorf("list conversations of %d: %w", userID, err)
	}
	defer rows.Close()

	var out []Conversation
	for rows.Next() {
		var one Conversation
		m := &one.Last
		if err := rows.Scan(&one.PartnerID, &m.ID, &m.SenderID, &m.RecipientID,
			&m.Body, &m.CreatedAt, &m.IsRead, &one.Unread); err != nil {
			return nil, fmt.Errorf("scan conversation: %w", err)
		}
		out = append(out, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list conversations of %d: %w", userID, err)
	}
	return out, nil
}

var qConversationBefore = register("ConversationBefore", `
SELECT id, sender_id, recipient_id, body, created_at, is_read
FROM web_directmessage
WHERE ((sender_id = $1 AND recipient_id = $2) OR (sender_id = $2 AND recipient_id = $1))
	AND ($3::bigint IS NULL OR id < $3)
ORDER BY id DESC
LIMIT $4`)

func (d *DB) ConversationBefore(ctx context.Context, userID, partnerID int64, before *int64, limit int) ([]DirectMessage, error) {
	return d.messages(ctx, qConversationBefore, userID, partnerID, before, limit)
}

var qConversationAfter = register("ConversationAfter", `
SELECT id, sender_id, recipient_id, body, created_at, is_read
FROM web_directmessage
WHERE ((sender_id = $1 AND recipient_id = $2) OR (sender_id = $2 AND recipient_id = $1))
	AND id > $3
ORDER BY id
LIMIT $4`)

func (d *DB) ConversationAfter(ctx context.Context, userID, partnerID, after int64, limit int) ([]DirectMessage, error) {
	return d.messages(ctx, qConversationAfter, userID, partnerID, after, limit)
}

func (d *DB) messages(ctx context.Context, sql string, userID, partnerID int64, bound any, limit int) ([]DirectMessage, error) {
	rows, err := d.pool.Query(ctx, sql, userID, partnerID, bound, limit)
	if err != nil {
		return nil, fmt.Errorf("list messages between %d and %d: %w", userID, partnerID, err)
	}
	defer rows.Close()
	return scanMessages(rows, userID, partnerID)
}

var qMessagesBetween = register("MessagesBetween", `
SELECT id, sender_id, recipient_id, body, created_at, is_read
FROM web_directmessage
WHERE (sender_id = $1 AND recipient_id = $2) OR (sender_id = $2 AND recipient_id = $1)
ORDER BY created_at`)

func (d *DB) MessagesBetween(ctx context.Context, userID, partnerID int64) ([]DirectMessage, error) {
	rows, err := d.pool.Query(ctx, qMessagesBetween, userID, partnerID)
	if err != nil {
		return nil, fmt.Errorf("list messages between %d and %d: %w", userID, partnerID, err)
	}
	defer rows.Close()
	return scanMessages(rows, userID, partnerID)
}

var qMessagesByIDs = register("MessagesByIDs", `
SELECT id, sender_id, recipient_id, body, created_at, is_read
FROM web_directmessage
WHERE id = ANY($3)
	AND ((sender_id = $1 AND recipient_id = $2) OR (sender_id = $2 AND recipient_id = $1))
ORDER BY created_at`)

func (d *DB) MessagesByIDs(ctx context.Context, userID, partnerID int64, ids []int64) ([]DirectMessage, error) {
	rows, err := d.pool.Query(ctx, qMessagesByIDs, userID, partnerID, ids)
	if err != nil {
		return nil, fmt.Errorf("list messages by id: %w", err)
	}
	defer rows.Close()
	return scanMessages(rows, userID, partnerID)
}

func scanMessages(rows pgx.Rows, userID, partnerID int64) ([]DirectMessage, error) {
	var out []DirectMessage
	for rows.Next() {
		var m DirectMessage
		if err := rows.Scan(&m.ID, &m.SenderID, &m.RecipientID, &m.Body, &m.CreatedAt, &m.IsRead); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list messages between %d and %d: %w", userID, partnerID, err)
	}
	return out, nil
}

var qIsBlocked = register("IsBlocked", `
SELECT EXISTS (
	SELECT 1 FROM web_directmessageblock
	WHERE blocker_id = $1 AND blocked_id = $2)`)

func (d *DB) IsBlocked(ctx context.Context, blockerID, blockedID int64) (bool, error) {
	var blocked bool
	if err := d.pool.QueryRow(ctx, qIsBlocked, blockerID, blockedID).Scan(&blocked); err != nil {
		return false, fmt.Errorf("check block of %d by %d: %w", blockedID, blockerID, err)
	}
	return blocked, nil
}

var qUnreadMessages = register("UnreadMessages", `
SELECT count(*)
FROM web_directmessage
WHERE recipient_id = $1 AND NOT is_read`)

func (d *DB) UnreadMessages(ctx context.Context, userID int64) (int, error) {
	var count int
	if err := d.pool.QueryRow(ctx, qUnreadMessages, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count unread messages of %d: %w", userID, err)
	}
	return count, nil
}

type Report struct {
	ID         int64
	ReporterID *int64
	ReportedID *int64
	Reason     string
	Messages   string
	Status     string
	CreatedAt  time.Time
}

var qReport = register("Report", `
SELECT id, reporter_id, reported_id, reason, reported_messages, status, created_at
FROM web_userreport
WHERE id = $1`)

func (d *DB) Report(ctx context.Context, id int64) (*Report, error) {
	var r Report
	err := d.pool.QueryRow(ctx, qReport, id).Scan(&r.ID, &r.ReporterID, &r.ReportedID,
		&r.Reason, &r.Messages, &r.Status, &r.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read report %d: %w", id, err)
	}
	return &r, nil
}

var qReportsSince = register("ReportsSince", `
SELECT count(*)
FROM web_userreport
WHERE reporter_id = $1 AND reported_id = $2 AND created_at >= $3`)

func (d *DB) ReportsSince(ctx context.Context, reporterID, reportedID int64, since time.Time) (int, error) {
	var count int
	if err := d.pool.QueryRow(ctx, qReportsSince, reporterID, reportedID, since).Scan(&count); err != nil {
		return 0, fmt.Errorf("count reports by %d against %d: %w", reporterID, reportedID, err)
	}
	return count, nil
}

type SuspiciousUser struct {
	UserID   int64
	Username string
	IP       *string
}

var qSuspiciousUsers = register("SuspiciousUsers", `
SELECT DISTINCT ON (l.user_id, l.origin_ip) l.user_id, u.username, host(l.origin_ip)
FROM web_actionlogentry l
JOIN web_user u ON u.id = l.user_id
ORDER BY l.user_id, l.origin_ip`)

func (d *DB) SuspiciousUsers(ctx context.Context) ([]SuspiciousUser, error) {
	rows, err := d.pool.Query(ctx, qSuspiciousUsers)
	if err != nil {
		return nil, fmt.Errorf("list addresses per user: %w", err)
	}
	defer rows.Close()

	var out []SuspiciousUser
	for rows.Next() {
		var one SuspiciousUser
		if err := rows.Scan(&one.UserID, &one.Username, &one.IP); err != nil {
			return nil, fmt.Errorf("scan address per user: %w", err)
		}
		out = append(out, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list addresses per user: %w", err)
	}
	return out, nil
}
