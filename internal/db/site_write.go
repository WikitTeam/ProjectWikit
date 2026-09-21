package db

import (
	"context"
	"fmt"
	"slices"

	"github.com/WikitTeam/ProjectWikit/internal/perms"
)

type NewSite struct {
	Slug        string
	Title       string
	Headline    string
	Domain      string
	MediaDomain string
	RoleNames   map[string]string
}

var qInsertSite = register("InsertSite", `
INSERT INTO web_site (slug, title, headline, domain, media_domain, home_page,
                      footer_license, signup_notice, password_help, email_policy,
                      membership_password, membership_password_enabled)
VALUES ($1, $2, $3, $4, $5, 'main', '', '', '', $6, '', false)
RETURNING id`)

var qInsertSiteSettings = register("InsertSiteSettings", `
INSERT INTO web_settings (site_id, category_id, rating_mode, can_user_create_tags)
VALUES ($1, NULL, 'updown', 'disabled')`)

var qInsertBuiltInRole = register("InsertBuiltInRole", `
INSERT INTO web_role (site_id, slug, index, name, short_name, is_staff, group_votes,
                      votes_title, inline_visual_mode, profile_visual_mode, color,
                      icon, badge_text, badge_bg, badge_text_color, badge_show_border)
VALUES ($1, $2, $3, $4, $4, $5, false, '', 'hidden', $6, '#000000',
        '', '', '#808080', '#ffffff', false)
RETURNING id`)

type siteRole struct {
	slug        string
	index       int
	staff       bool
	profile     string
	permissions []string
}

var builtInRoles = []siteRole{
	{slug: "registered", index: 2, profile: "hidden"},
	{slug: "everyone", index: 3, profile: "hidden", permissions: []string{
		perms.ViewArticles,
		perms.ViewArticleComments,
		perms.ViewForumSections,
		perms.ViewForumCategories,
		perms.ViewForumThreads,
		perms.ViewForumPosts,
	}},
}

var starterRoles = []siteRole{
	{slug: "admin", index: 0, staff: true, profile: "status", permissions: catalogNames()},
	{slug: "member", index: 1, profile: "status", permissions: without(
		catalogNames("articles", "forum", "social"),
		perms.LockArticles, perms.ResetArticleVotes, perms.ManageArticleAuthors,
		perms.DeleteForumPosts, perms.CreateForumThreads, perms.EditForumThreads,
		perms.PinForumThreads, perms.LockForumThreads, perms.MoveForumThreads,
	)},
}

func catalogNames(groups ...string) []string {
	var out []string
	for _, g := range perms.Catalog {
		if len(groups) == 0 || slices.Contains(groups, g.Key) {
			out = append(out, g.Names...)
		}
	}
	return out
}

func without(names []string, withheld ...string) []string {
	return slices.DeleteFunc(names, func(name string) bool { return slices.Contains(withheld, name) })
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
	for _, role := range slices.Concat(starterRoles, builtInRoles) {
		var roleID int64
		err := tx.QueryRow(ctx, qInsertBuiltInRole, id, role.slug, role.index,
			s.RoleNames[role.slug], role.staff, role.profile).Scan(&roleID)
		if err != nil {
			return 0, fmt.Errorf("create role %q for site %q: %w", role.slug, s.Slug, err)
		}
		if len(role.permissions) == 0 {
			continue
		}
		if _, err := tx.Exec(ctx, qGrantRolePermission, roleID, role.permissions); err != nil {
			return 0, fmt.Errorf("grant %q on site %q: %w", role.slug, s.Slug, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit site %q: %w", s.Slug, err)
	}
	return id, nil
}
