package db

import (
	"context"
	"fmt"

	"github.com/WikitTeam/ProjectWikit/internal/perms"
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

var qInsertBuiltInRole = register("InsertBuiltInRole", `
INSERT INTO web_role (site_id, slug, index, name, short_name, is_staff, group_votes,
                      votes_title, inline_visual_mode, profile_visual_mode, color,
                      icon, badge_text, badge_bg, badge_text_color, badge_show_border)
VALUES ($1, $2, $3, '', '', false, false, '', 'hidden', 'hidden', '#000000',
        '', '', '#808080', '#ffffff', false)
RETURNING id`)

// The permission resolver looks both of these up by slug, so a site without
// them answers every question with no rights at all.
var builtInRoles = []struct {
	slug        string
	index       int
	permissions []string
}{
	{slug: "registered", index: 1},
	{slug: "everyone", index: 2, permissions: []string{
		perms.ViewArticles,
		perms.ViewArticleComments,
		perms.ViewForumSections,
		perms.ViewForumCategories,
		perms.ViewForumThreads,
		perms.ViewForumPosts,
	}},
}

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
	for _, built := range builtInRoles {
		var roleID int64
		err := tx.QueryRow(ctx, qInsertBuiltInRole, id, built.slug, built.index).Scan(&roleID)
		if err != nil {
			return 0, fmt.Errorf("create role %q for site %q: %w", built.slug, s.Slug, err)
		}
		if len(built.permissions) == 0 {
			continue
		}
		if _, err := tx.Exec(ctx, qGrantRolePermission, roleID, built.permissions); err != nil {
			return 0, fmt.Errorf("grant %q on site %q: %w", built.slug, s.Slug, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit site %q: %w", s.Slug, err)
	}
	return id, nil
}
