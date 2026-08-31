package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type Site struct {
	ID          int64
	Slug        string
	Title       string
	Headline    string
	Domain      string
	MediaDomain string
	HomePage    string
	Icon        string
	ThemeID     *int64

	AuthIcon      string
	FooterLicense string
	SignupNotice  string

	MembershipPasswordEnabled bool
	MembershipPassword        string
	MembershipPasswordRoleID  *int64
}

var qSiteByHost = register("SiteByHost", `
SELECT id, slug, title, headline, domain, media_domain, home_page, COALESCE(icon, ''), active_theme_id,
       COALESCE(auth_icon, ''), footer_license, signup_notice,
       membership_password_enabled, membership_password, membership_password_role_id
FROM web_site
WHERE domain = $1 OR media_domain = $1
ORDER BY id
LIMIT 1`)

// SiteByHosts tries each host in turn and returns the first that matches.
// Callers pass site.LookupHosts, whose ordering carries the host:port round
// that has to run before the bare-host one.
func (d *DB) SiteByHosts(ctx context.Context, hosts []string) (*Site, error) {
	for _, host := range hosts {
		var s Site
		err := d.pool.QueryRow(ctx, qSiteByHost, host).Scan(
			&s.ID, &s.Slug, &s.Title, &s.Headline, &s.Domain, &s.MediaDomain, &s.HomePage,
			&s.Icon, &s.ThemeID,
			&s.AuthIcon, &s.FooterLicense, &s.SignupNotice,
			&s.MembershipPasswordEnabled, &s.MembershipPassword, &s.MembershipPasswordRoleID)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("lookup site by host %q: %w", host, err)
		}
		return &s, nil
	}
	return nil, ErrNotFound
}

var qAnySite = register("AnySite", `SELECT EXISTS(SELECT 1 FROM web_site)`)

// AnySite separates "this domain is not configured" from "nothing is set up
// yet"; the two lead to different responses.
func (d *DB) AnySite(ctx context.Context) (bool, error) {
	var exists bool
	if err := d.pool.QueryRow(ctx, qAnySite).Scan(&exists); err != nil {
		return false, fmt.Errorf("check any site exists: %w", err)
	}
	return exists, nil
}
