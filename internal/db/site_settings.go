package db

import (
	"context"
	"fmt"
)

type SiteSettings struct {
	RatingMode string
	CreateTags string
}

var qSiteSettingsRow = register("SiteSettingsRow", `
SELECT rating_mode, can_user_create_tags
FROM web_settings WHERE site_id = $1 AND category_id IS NULL`)

func (d *DB) SiteSettings(ctx context.Context, siteID int64) (SiteSettings, error) {
	var s SiteSettings
	rows, err := d.pool.Query(ctx, qSiteSettingsRow, siteID)
	if err != nil {
		return s, fmt.Errorf("read settings of site %d: %w", siteID, err)
	}
	defer rows.Close()
	if rows.Next() {
		if err := rows.Scan(&s.RatingMode, &s.CreateTags); err != nil {
			return s, err
		}
	}
	return s, rows.Err()
}

var qUpdateSiteSettings = register("UpdateSiteSettings", `
UPDATE web_settings SET rating_mode = $2, can_user_create_tags = $3
WHERE site_id = $1 AND category_id IS NULL`)

var qUpdateSite = register("UpdateSite", `
UPDATE web_site SET
	slug = $2, title = $3, headline = $4, domain = $5, media_domain = $6, home_page = $7,
	active_theme_id = $8, system_theme_id = $9, icon = $10, auth_icon = $11,
	footer_license = $12, signup_notice = $13, password_help = $14, email_policy = $15,
	language = $16
WHERE id = $1`)

var qUpdateSiteRoles = register("UpdateSiteRoles", `
UPDATE web_site SET
	default_role_id = $2, verified_role_id = $3,
	membership_password_enabled = $4, membership_password = $5, membership_password_role_id = $6
WHERE id = $1`)

func (d *DB) SaveSite(ctx context.Context, s *Site, settings SiteSettings, withRoles bool) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin saving site %d: %w", s.ID, err)
	}
	defer tx.Rollback(context.WithoutCancel(ctx))

	_, err = tx.Exec(ctx, qUpdateSite, s.ID, s.Slug, s.Title, s.Headline, s.Domain, s.MediaDomain,
		s.HomePage, s.ThemeID, s.SystemThemeID, nullable(s.Icon), nullable(s.AuthIcon),
		s.FooterLicense, s.SignupNotice, s.PasswordHelp, s.EmailPolicy, s.Language)
	if err != nil {
		return fmt.Errorf("save site %d: %w", s.ID, err)
	}
	if withRoles {
		_, err = tx.Exec(ctx, qUpdateSiteRoles, s.ID, s.DefaultRoleID, s.VerifiedRoleID,
			s.MembershipPasswordEnabled, s.MembershipPassword, s.MembershipPasswordRoleID)
		if err != nil {
			return fmt.Errorf("save the roles of site %d: %w", s.ID, err)
		}
	}
	if _, err := tx.Exec(ctx, qUpdateSiteSettings, s.ID, settings.RatingMode, settings.CreateTags); err != nil {
		return fmt.Errorf("save the settings of site %d: %w", s.ID, err)
	}
	return tx.Commit(ctx)
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
