package db

import (
	"context"
	"fmt"
	"time"
)

type NotifyMute struct {
	ID         int64
	ArticleID  *int64
	ThreadID   *int64
	UserID     *int64
	Title      string
	Path       string
	SiteTitle  string
	SiteDomain string
	CreatedAt  time.Time
}

var (
	qMuteArticle = register("MuteArticle", `
INSERT INTO pwikit_notify_mute (user_id, article_id, created_at) VALUES ($1, $2, $3)
ON CONFLICT (user_id, article_id) WHERE article_id IS NOT NULL DO NOTHING`)

	qMuteThread = register("MuteThread", `
INSERT INTO pwikit_notify_mute (user_id, thread_id, created_at) VALUES ($1, $2, $3)
ON CONFLICT (user_id, thread_id) WHERE thread_id IS NOT NULL DO NOTHING`)

	qMuteUser = register("MuteUser", `
INSERT INTO pwikit_notify_mute (user_id, muted_user, created_at) VALUES ($1, $2, $3)
ON CONFLICT (user_id, muted_user) WHERE muted_user IS NOT NULL DO NOTHING`)

	qUnmute = register("Unmute", `
DELETE FROM pwikit_notify_mute
WHERE user_id = $1 AND (article_id = $2 OR thread_id = $3 OR muted_user = $4)`)

	qMutes = register("Mutes", `
SELECT m.id, m.article_id, m.thread_id, m.muted_user,
       coalesce(nullif(a.title, ''), CASE WHEN a.category = '_default' THEN a.name::text ELSE a.complete_full_name::text END, t.name,
                CASE WHEN u.type = 'wikidot' THEN u.wikidot_username ELSE coalesce(nullif(u.display_name, ''), u.username) END, ''),
       CASE WHEN a.id IS NOT NULL THEN '/' || CASE WHEN a.category = '_default' THEN a.name::text ELSE a.complete_full_name::text END
            WHEN t.id IS NOT NULL THEN '/forum/t-' || t.id
            ELSE '/-/users/' || u.username END,
       coalesce(s.title, ''), coalesce(s.domain, ''), m.created_at
FROM pwikit_notify_mute m
LEFT JOIN web_article a ON a.id = m.article_id
LEFT JOIN web_forumthread t ON t.id = m.thread_id
LEFT JOIN web_user u ON u.id = m.muted_user
LEFT JOIN web_site s ON s.id = coalesce(a.site_id, t.site_id)
WHERE m.user_id = $1
ORDER BY m.created_at DESC`)

	qMutedReaders = register("MutedReaders", `
SELECT user_id FROM pwikit_notify_mute
WHERE user_id = ANY($1::bigint[]) AND (article_id = $2 OR thread_id = $3 OR muted_user = $4)`)
)

func (d *DB) MuteArticle(ctx context.Context, userID, articleID int64) error {
	if _, err := d.pool.Exec(ctx, qMuteArticle, userID, articleID, time.Now().UTC()); err != nil {
		return fmt.Errorf("mute article %d for %d: %w", articleID, userID, err)
	}
	return nil
}

func (d *DB) MuteThread(ctx context.Context, userID, threadID int64) error {
	if _, err := d.pool.Exec(ctx, qMuteThread, userID, threadID, time.Now().UTC()); err != nil {
		return fmt.Errorf("mute thread %d for %d: %w", threadID, userID, err)
	}
	return nil
}

func (d *DB) MuteUser(ctx context.Context, userID, mutedID int64) error {
	if _, err := d.pool.Exec(ctx, qMuteUser, userID, mutedID, time.Now().UTC()); err != nil {
		return fmt.Errorf("mute user %d for %d: %w", mutedID, userID, err)
	}
	return nil
}

func (d *DB) Unmute(ctx context.Context, userID int64, articleID, threadID, mutedID *int64) error {
	if _, err := d.pool.Exec(ctx, qUnmute, userID, articleID, threadID, mutedID); err != nil {
		return fmt.Errorf("unmute for %d: %w", userID, err)
	}
	return nil
}

func (d *DB) Mutes(ctx context.Context, userID int64) ([]NotifyMute, error) {
	rows, err := d.pool.Query(ctx, qMutes, userID)
	if err != nil {
		return nil, fmt.Errorf("list mutes of %d: %w", userID, err)
	}
	defer rows.Close()
	var out []NotifyMute
	for rows.Next() {
		var m NotifyMute
		if err := rows.Scan(&m.ID, &m.ArticleID, &m.ThreadID, &m.UserID, &m.Title, &m.Path, &m.SiteTitle, &m.SiteDomain, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan a mute: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// WithoutMuted drops the readers who muted the page or thread a notice is about,
// or the person whose doing it is. Pass nil for whatever does not apply.
func (d *DB) WithoutMuted(ctx context.Context, readers []int64, articleID, threadID, actorID *int64) ([]int64, error) {
	if len(readers) == 0 {
		return readers, nil
	}
	rows, err := d.pool.Query(ctx, qMutedReaders, readers, articleID, threadID, actorID)
	if err != nil {
		return nil, fmt.Errorf("read mutes: %w", err)
	}
	defer rows.Close()
	muted := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan a mute: %w", err)
		}
		muted[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := readers[:0:0]
	for _, id := range readers {
		if !muted[id] {
			out = append(out, id)
		}
	}
	return out, nil
}
