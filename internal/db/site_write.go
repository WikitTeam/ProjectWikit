package db

import (
	"context"
	"fmt"
)

type NewSite struct {
	Slug        string
	Title       string
	Headline    string
	Domain      string
	MediaDomain string
}

var qInsertSite = register("InsertSite", `
INSERT INTO web_site (slug, title, headline, domain, media_domain, home_page,
                      footer_license, signup_notice, password_help, email_policy,
                      membership_password, membership_password_enabled)
VALUES ($1, $2, $3, $4, $5, 'main', '', '', '', $6, '', false)
RETURNING id`)

var qInsertSiteSettings = register("InsertSiteSettings", `
INSERT INTO web_settings (site_id, category_id, rating_mode, can_user_create_tags)
VALUES ($1, NULL, 'default', 'default')`)

func (d *DB) CreateSite(ctx context.Context, s NewSite) (int64, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin creating site %q: %w", s.Slug, err)
	}
	defer tx.Rollback(context.WithoutCancel(ctx))

	var id int64
	err = tx.QueryRow(ctx, qInsertSite, s.Slug, s.Title, s.Headline, s.Domain, s.MediaDomain, EmailOptional).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create site %q: %w", s.Slug, err)
	}
	if _, err := tx.Exec(ctx, qInsertSiteSettings, id); err != nil {
		return 0, fmt.Errorf("create settings for site %q: %w", s.Slug, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit site %q: %w", s.Slug, err)
	}
	return id, nil
}
