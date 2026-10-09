package db

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

var qSendDirectMessage = register("SendDirectMessage", `
INSERT INTO web_directmessage (sender_id, recipient_id, body, created_at, is_read)
VALUES ($1, $2, $3, $4, false)
RETURNING id`)

func (d *DB) SendDirectMessage(ctx context.Context, senderID, recipientID int64, body string, at time.Time) (DirectMessage, error) {
	out := DirectMessage{SenderID: senderID, RecipientID: recipientID, Body: body, CreatedAt: at}
	err := d.pool.QueryRow(ctx, qSendDirectMessage, senderID, recipientID, body, at).Scan(&out.ID)
	if err != nil {
		return DirectMessage{}, fmt.Errorf("send message from %d to %d: %w", senderID, recipientID, err)
	}
	return out, nil
}

var qMarkConversationRead = register("MarkConversationRead", `
UPDATE web_directmessage
SET is_read = true
WHERE recipient_id = $1 AND sender_id = $2 AND NOT is_read`)

var qMarkMessageNotificationsViewed = register("MarkMessageNotificationsViewed", `
UPDATE web_usernotificationmapping m
SET is_viewed = true
FROM web_usernotification n
WHERE m.notification_id = n.id
	AND m.recipient_id = $1
	AND NOT m.is_viewed
	AND n.type = $3
	AND n.meta->>'sender_id' = $2`)

func (d *DB) MarkConversationRead(ctx context.Context, userID, partnerID int64) (int64, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin conversation read: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, qMarkConversationRead, userID, partnerID)
	if err != nil {
		return 0, fmt.Errorf("mark conversation with %d read: %w", partnerID, err)
	}
	read := tag.RowsAffected()
	if read > 0 {
		partner := strconv.FormatInt(partnerID, 10)
		if _, err := tx.Exec(ctx, qMarkMessageNotificationsViewed, userID, partner, NotifyDirectMessage); err != nil {
			return 0, fmt.Errorf("mark message notifications of %d viewed: %w", userID, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit conversation read: %w", err)
	}
	return read, nil
}

var qBlockUser = register("BlockUser", `
INSERT INTO web_directmessageblock (blocker_id, blocked_id, created_at)
VALUES ($1, $2, $3)
ON CONFLICT (blocker_id, blocked_id) DO NOTHING`)

func (d *DB) BlockUser(ctx context.Context, blockerID, blockedID int64, at time.Time) (bool, error) {
	tag, err := d.pool.Exec(ctx, qBlockUser, blockerID, blockedID, at)
	if err != nil {
		return false, fmt.Errorf("block %d for %d: %w", blockedID, blockerID, err)
	}
	return tag.RowsAffected() > 0, nil
}

var qUnblockUser = register("UnblockUser", `
DELETE FROM web_directmessageblock
WHERE blocker_id = $1 AND blocked_id = $2`)

func (d *DB) UnblockUser(ctx context.Context, blockerID, blockedID int64) (bool, error) {
	tag, err := d.pool.Exec(ctx, qUnblockUser, blockerID, blockedID)
	if err != nil {
		return false, fmt.Errorf("unblock %d for %d: %w", blockedID, blockerID, err)
	}
	return tag.RowsAffected() > 0, nil
}

var qCreateReport = register("CreateReport", `
INSERT INTO web_userreport (reporter_id, reported_id, reason, reported_messages, status, admin_notes, created_at, site_id)
VALUES ($1, $2, $3, $4, $5, '', $6, $7)
RETURNING id`)

func (d *DB) CreateReport(ctx context.Context, siteID, reporterID, reportedID int64, reason, snapshot string, at time.Time) (int64, error) {
	var id int64
	err := d.pool.QueryRow(ctx, qCreateReport, reporterID, reportedID, reason, snapshot, ReportPending, at, siteID).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("write report by %d against %d: %w", reporterID, reportedID, err)
	}
	return id, nil
}

const ReportPending = "pending"

var qClearConversations = register("ClearConversations", `
INSERT INTO pwikit_message_clear (user_id, partner_id, through_id)
SELECT $1, p.partner_id, max(m.id)
FROM unnest($2::bigint[]) AS p(partner_id)
JOIN web_directmessage m
	ON (m.sender_id = $1 AND m.recipient_id = p.partner_id) OR (m.sender_id = p.partner_id AND m.recipient_id = $1)
WHERE p.partner_id <> $1
GROUP BY p.partner_id
ON CONFLICT (user_id, partner_id) DO UPDATE
SET through_id = greatest(pwikit_message_clear.through_id, EXCLUDED.through_id)
RETURNING partner_id`)

var qReadClearedMessages = register("ReadClearedMessages", `
UPDATE web_directmessage m
SET is_read = true
FROM pwikit_message_clear c
WHERE c.user_id = $1
	AND c.partner_id = ANY($2)
	AND m.recipient_id = $1
	AND m.sender_id = c.partner_id
	AND m.id <= c.through_id
	AND NOT m.is_read`)

func (d *DB) ClearConversations(ctx context.Context, userID int64, partnerIDs []int64) ([]int64, error) {
	if len(partnerIDs) == 0 {
		return nil, nil
	}
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin clearing conversations: %w", err)
	}
	defer tx.Rollback(context.WithoutCancel(ctx))

	rows, err := tx.Query(ctx, qClearConversations, userID, partnerIDs)
	if err != nil {
		return nil, fmt.Errorf("clear conversations of %d: %w", userID, err)
	}
	var cleared []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan cleared conversation: %w", err)
		}
		cleared = append(cleared, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("clear conversations of %d: %w", userID, err)
	}
	if len(cleared) == 0 {
		return nil, nil
	}
	if _, err := tx.Exec(ctx, qReadClearedMessages, userID, cleared); err != nil {
		return nil, fmt.Errorf("mark cleared messages of %d read: %w", userID, err)
	}
	for _, partner := range cleared {
		if _, err := tx.Exec(ctx, qMarkMessageNotificationsViewed, userID, strconv.FormatInt(partner, 10), NotifyDirectMessage); err != nil {
			return nil, fmt.Errorf("mark message notifications of %d viewed: %w", userID, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit clearing conversations: %w", err)
	}
	return cleared, nil
}
