package db

import (
	"context"
	"fmt"
	"time"
)

const (
	NotifyWelcome            = "welcome"
	NotifyNewPostReply       = "new_post_reply"
	NotifyNewThreadPost      = "new_thread_post"
	NotifyNewArticleRevision = "new_article_revision"
	NotifyForumMention       = "forum_mention"
	NotifyDirectMessage      = "direct_message"
)

var (
	qInsertNotification = register("InsertNotification", `
INSERT INTO web_usernotification (type, meta, created_at)
VALUES ($1, $2, $3)
RETURNING id`)

	qInsertNotificationMappings = register("InsertNotificationMappings", `
INSERT INTO web_usernotificationmapping (notification_id, recipient_id, is_viewed)
SELECT $1, recipient, false
FROM unnest($2::bigint[]) AS recipient`)
)

// The notification and the rows naming its readers go in together, so nobody
// ends up with a notification that reaches no one.
func (d *DB) SendNotification(ctx context.Context, kind, meta string, recipients []int64, at time.Time) error {
	if len(recipients) == 0 {
		return nil
	}
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin notification: %w", err)
	}
	defer tx.Rollback(ctx)

	var id int64
	if err := tx.QueryRow(ctx, qInsertNotification, kind, meta, at).Scan(&id); err != nil {
		return fmt.Errorf("write notification %q: %w", kind, err)
	}
	if _, err := tx.Exec(ctx, qInsertNotificationMappings, id, recipients); err != nil {
		return fmt.Errorf("write recipients of notification %d: %w", id, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit notification %q: %w", kind, err)
	}
	return nil
}

var qArticleSubscribers = register("ArticleSubscribers", `
SELECT subscriber_id
FROM web_usernotificationsubscription
WHERE article_id = $1
ORDER BY subscriber_id`)

func (d *DB) ArticleSubscribers(ctx context.Context, articleID int64) ([]int64, error) {
	return d.subscriberIDs(ctx, qArticleSubscribers, articleID)
}

var qThreadSubscribers = register("ThreadSubscribers", `
SELECT subscriber_id
FROM web_usernotificationsubscription
WHERE forum_thread_id = $1
ORDER BY subscriber_id`)

func (d *DB) ThreadSubscribers(ctx context.Context, threadID int64) ([]int64, error) {
	return d.subscriberIDs(ctx, qThreadSubscribers, threadID)
}

func (d *DB) subscriberIDs(ctx context.Context, sql string, id int64) ([]int64, error) {
	rows, err := d.pool.Query(ctx, sql, id)
	if err != nil {
		return nil, fmt.Errorf("list subscribers of %d: %w", id, err)
	}
	defer rows.Close()

	var out []int64
	for rows.Next() {
		var one int64
		if err := rows.Scan(&one); err != nil {
			return nil, fmt.Errorf("scan subscriber: %w", err)
		}
		out = append(out, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list subscribers of %d: %w", id, err)
	}
	return out, nil
}

var qSubscribeToThread = register("SubscribeToThread", `
INSERT INTO web_usernotificationsubscription (subscriber_id, forum_thread_id)
SELECT $1, $2
WHERE NOT EXISTS (
	SELECT 1 FROM web_usernotificationsubscription
	WHERE subscriber_id = $1 AND forum_thread_id = $2)`)

func (d *DB) SubscribeToThread(ctx context.Context, userID, threadID int64) error {
	if _, err := d.pool.Exec(ctx, qSubscribeToThread, userID, threadID); err != nil {
		return fmt.Errorf("subscribe %d to thread %d: %w", userID, threadID, err)
	}
	return nil
}
