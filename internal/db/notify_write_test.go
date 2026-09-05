package db

import (
	"context"
	"testing"
	"time"
)

func scratchUser(t *testing.T, d *DB, name string) int64 {
	t.Helper()
	ctx := context.Background()
	name = name + "-" + time.Now().Format("150405.000000")
	var id int64
	err := d.pool.QueryRow(ctx, `
INSERT INTO web_user (password, is_superuser, first_name, last_name, email, date_joined,
	username, type, bio, is_forum_active, is_active, can_send_direct_messages)
VALUES ('!', false, '', '', $1, now(), $2, 'normal', '', true, true, true)
RETURNING id`, name+"@example.invalid", name).Scan(&id)
	if err != nil {
		t.Fatalf("insert scratch user err = %v, want nil", err)
	}
	t.Cleanup(func() {
		clean := context.Background()
		for _, sql := range []string{
			`DELETE FROM web_usernotificationmapping WHERE recipient_id = $1`,
			`DELETE FROM web_usernotificationsubscription WHERE subscriber_id = $1`,
			`DELETE FROM web_forumpostlike WHERE user_id = $1`,
			`DELETE FROM web_articlefavourite WHERE user_id = $1`,
			`DELETE FROM web_user WHERE id = $1`,
		} {
			if _, err := d.pool.Exec(clean, sql, id); err != nil {
				t.Errorf("clean up scratch user err = %v, want nil", err)
			}
		}
	})
	return id
}

func dropNotification(t *testing.T, d *DB, after int64) {
	t.Helper()
	t.Cleanup(func() {
		clean := context.Background()
		for _, sql := range []string{
			`DELETE FROM web_usernotificationmapping WHERE notification_id > $1`,
			`DELETE FROM web_usernotification WHERE id > $1`,
		} {
			if _, err := d.pool.Exec(clean, sql, after); err != nil {
				t.Errorf("clean up notification err = %v, want nil", err)
			}
		}
	})
}

func highestNotification(t *testing.T, d *DB) int64 {
	t.Helper()
	var id int64
	err := d.pool.QueryRow(context.Background(),
		`SELECT COALESCE(MAX(id), 0) FROM web_usernotification`).Scan(&id)
	if err != nil {
		t.Fatalf("read highest notification err = %v, want nil", err)
	}
	return id
}

func TestSendNotificationReachesEveryRecipient(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	dropNotification(t, d, highestNotification(t, d))
	first := scratchUser(t, d, "probe-notify-a")
	second := scratchUser(t, d, "probe-notify-b")

	err := d.SendNotification(ctx, NotifyNewArticleRevision, `{"probe": true}`,
		[]int64{first, second}, time.Now().UTC())
	if err != nil {
		t.Fatalf("SendNotification() err = %v, want nil", err)
	}

	var count int
	err = d.pool.QueryRow(ctx, `
SELECT count(*) FROM web_usernotificationmapping m
JOIN web_usernotification n ON n.id = m.notification_id
WHERE n.type = $1 AND m.recipient_id = ANY($2) AND m.is_viewed = false`,
		NotifyNewArticleRevision, []int64{first, second}).Scan(&count)
	if err != nil {
		t.Fatalf("count recipients err = %v, want nil", err)
	}
	if count != 2 {
		t.Errorf("SendNotification() reached %d recipients, want 2", count)
	}
}

func TestSendNotificationToNobodyWritesNothing(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	before := highestNotification(t, d)

	if err := d.SendNotification(ctx, NotifyWelcome, `{}`, nil, time.Now().UTC()); err != nil {
		t.Fatalf("SendNotification() err = %v, want nil", err)
	}
	if got := highestNotification(t, d); got != before {
		t.Errorf("highest notification = %d, want %d", got, before)
	}
}

func TestArticleSubscribersListsWhoAsked(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	article := scratchArticle(t, d)
	first := scratchUser(t, d, "probe-sub-a")
	second := scratchUser(t, d, "probe-sub-b")

	for _, id := range []int64{first, second} {
		if err := d.SubscribeToArticle(ctx, id, article); err != nil {
			t.Fatalf("SubscribeToArticle(%d) err = %v, want nil", id, err)
		}
	}
	got, err := d.ArticleSubscribers(ctx, article)
	if err != nil {
		t.Fatalf("ArticleSubscribers() err = %v, want nil", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(ArticleSubscribers()) = %d, want 2", len(got))
	}
	if got[0] != first || got[1] != second {
		t.Errorf("ArticleSubscribers() = %v, want [%d %d]", got, first, second)
	}
}

func TestSubscribeToThreadOnlyOnce(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	user := scratchUser(t, d, "probe-sub-thread")
	var thread int64
	if err := d.pool.QueryRow(ctx, `SELECT id FROM web_forumthread ORDER BY id LIMIT 1`).Scan(&thread); err != nil {
		t.Skipf("no forum thread in the write database, skipping")
	}

	for i := 0; i < 2; i++ {
		if err := d.SubscribeToThread(ctx, user, thread); err != nil {
			t.Fatalf("SubscribeToThread() err = %v, want nil", err)
		}
	}
	got, err := d.ThreadSubscribers(ctx, thread)
	if err != nil {
		t.Fatalf("ThreadSubscribers() err = %v, want nil", err)
	}
	seen := 0
	for _, id := range got {
		if id == user {
			seen++
		}
	}
	if seen != 1 {
		t.Errorf("ThreadSubscribers() holds the subscriber %d times, want 1", seen)
	}
}

func TestLogEntryByIDReadsWhatWasWritten(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	article := scratchArticle(t, d)

	rev, err := d.AddArticleLogEntry(ctx, article, nil, LogTitle, "why", `{"title": "After"}`, time.Now().UTC())
	if err != nil {
		t.Fatalf("AddArticleLogEntry() err = %v, want nil", err)
	}
	if rev.EntryID == 0 {
		t.Fatal("AddArticleLogEntry().EntryID = 0, want the row id")
	}

	entry, err := d.LogEntryByID(ctx, rev.EntryID)
	if err != nil {
		t.Fatalf("LogEntryByID() err = %v, want nil", err)
	}
	if entry.Type != LogTitle {
		t.Errorf("LogEntryByID().Type = %q, want %q", entry.Type, LogTitle)
	}
	if entry.Comment != "why" {
		t.Errorf("LogEntryByID().Comment = %q, want %q", entry.Comment, "why")
	}
	if entry.RevNumber != rev.RevNumber {
		t.Errorf("LogEntryByID().RevNumber = %d, want %d", entry.RevNumber, rev.RevNumber)
	}
}
