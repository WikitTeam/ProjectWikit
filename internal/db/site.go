package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

	SystemThemeID *int64

	AuthIcon      string
	FooterLicense string
	SignupNotice  string
	PasswordHelp  string

	MembershipPasswordEnabled bool
	MembershipPassword        string
	MembershipPasswordRoleID  *int64

	DefaultRoleID  *int64
	VerifiedRoleID *int64

	EmailPolicy string
}

var qSiteByHost = register("SiteByHost", `
SELECT id, slug, title, headline, domain, media_domain, home_page, COALESCE(icon, ''), active_theme_id,
       system_theme_id, COALESCE(auth_icon, ''), footer_license, signup_notice, password_help,
       membership_password_enabled, membership_password, membership_password_role_id,
       default_role_id, verified_role_id, email_policy
FROM web_site
WHERE domain = $1 OR media_domain = $1
ORDER BY id
LIMIT 1`)

func scanSite(row pgx.Row, s *Site) error {
	return row.Scan(
		&s.ID, &s.Slug, &s.Title, &s.Headline, &s.Domain, &s.MediaDomain, &s.HomePage,
		&s.Icon, &s.ThemeID, &s.SystemThemeID,
		&s.AuthIcon, &s.FooterLicense, &s.SignupNotice, &s.PasswordHelp,
		&s.MembershipPasswordEnabled, &s.MembershipPassword, &s.MembershipPasswordRoleID,
		&s.DefaultRoleID, &s.VerifiedRoleID, &s.EmailPolicy)
}

var qSiteBySlug = register("SiteBySlug", `
SELECT id, slug, title, headline, domain, media_domain, home_page, COALESCE(icon, ''), active_theme_id,
       system_theme_id, COALESCE(auth_icon, ''), footer_license, signup_notice, password_help,
       membership_password_enabled, membership_password, membership_password_role_id,
       default_role_id, verified_role_id, email_policy
FROM web_site
WHERE slug = $1`)

var qSiteSlugs = register("SiteSlugs", `SELECT slug FROM web_site ORDER BY id`)

func (d *DB) SiteBySlug(ctx context.Context, slug string) (*Site, error) {
	var s Site
	err := scanSite(d.pool.QueryRow(ctx, qSiteBySlug, slug), &s)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("look up site %q: %w", slug, err)
	}
	return &s, nil
}

func (d *DB) SiteSlugs(ctx context.Context) ([]string, error) {
	rows, err := d.pool.Query(ctx, qSiteSlugs)
	if err != nil {
		return nil, fmt.Errorf("list sites: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, err
		}
		out = append(out, slug)
	}
	return out, rows.Err()
}

// SiteByHosts tries each host in turn and returns the first that matches.
// Callers pass site.LookupHosts, whose ordering carries the host:port round
// that has to run before the bare-host one.
func (d *DB) SiteByHosts(ctx context.Context, hosts []string) (*Site, error) {
	for _, host := range hosts {
		var s Site
		err := scanSite(d.pool.QueryRow(ctx, qSiteByHost, host), &s)
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

var qSiteHostExists = register("SiteHostExists", `
SELECT EXISTS(SELECT 1 FROM web_site WHERE lower(domain) = $1 OR lower(media_domain) = $1)`)

func (d *DB) SiteHostExists(ctx context.Context, host string) (bool, error) {
	var exists bool
	if err := d.pool.QueryRow(ctx, qSiteHostExists, strings.ToLower(host)).Scan(&exists); err != nil {
		return false, fmt.Errorf("check host %q belongs to a site: %w", host, err)
	}
	return exists, nil
}

const (
	EmailAtSignup = "at_signup"
	EmailRequired = "required"
	EmailOptional = "optional"
)
