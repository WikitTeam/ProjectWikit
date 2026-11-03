package db

import (
	"context"
	"fmt"
	"time"
)

type ProfileField struct {
	Label string
	Value string
}

type ProfileLink struct {
	Platform string
	Value    string
}

type AuthoredPage struct {
	ArticleID  int64
	Title      string
	Path       string
	SiteID     int64
	Category   string
	SiteTitle  string
	SiteDomain string
	CreatedAt  time.Time
	Pinned     bool
}

const MaxPins = 6

var ProfileFieldKeys = []string{"region", "gender", "birthday", "website"}

var (
	qProfileFields = register("ProfileFields", `
SELECT label, value FROM pwikit_profile_field WHERE user_id = $1 ORDER BY position`)
	qDropProfileFields = register("DropProfileFields", `DELETE FROM pwikit_profile_field WHERE user_id = $1`)
	qAddProfileField   = register("AddProfileField", `
INSERT INTO pwikit_profile_field (user_id, position, label, value) VALUES ($1, $2, $3, $4)`)

	qProfileLinks = register("ProfileLinks", `
SELECT platform, value FROM pwikit_profile_link WHERE user_id = $1 ORDER BY position`)
	qDropProfileLinks = register("DropProfileLinks", `DELETE FROM pwikit_profile_link WHERE user_id = $1`)
	qAddProfileLink   = register("AddProfileLink", `
INSERT INTO pwikit_profile_link (user_id, position, platform, value) VALUES ($1, $2, $3, $4)`)

	qAuthoredPages = register("AuthoredPages", `
SELECT a.id, coalesce(nullif(a.title, ''), CASE WHEN a.category = '_default' THEN a.name::text ELSE a.complete_full_name::text END), '/' || CASE WHEN a.category = '_default' THEN a.name::text ELSE a.complete_full_name::text END,
       coalesce(a.site_id, 0), a.category, coalesce(s.title, ''), coalesce(s.domain, ''), a.created_at,
       EXISTS (SELECT 1 FROM pwikit_profile_pin p WHERE p.user_id = $1 AND p.article_id = a.id)
FROM web_article_authors aa
JOIN web_article a ON a.id = aa.article_id
LEFT JOIN web_site s ON s.id = a.site_id
WHERE aa.user_id = $1 AND ($2 = false OR EXISTS (SELECT 1 FROM pwikit_profile_pin p WHERE p.user_id = $1 AND p.article_id = a.id))
  AND ($4::timestamptz IS NULL OR a.created_at >= $4)
  AND ($5::timestamptz IS NULL OR a.created_at < $5)
ORDER BY a.created_at DESC, a.id DESC
LIMIT $3`)

	qPin = register("PinPage", `
INSERT INTO pwikit_profile_pin (user_id, article_id, pinned_at)
SELECT $1, $2, $3
WHERE EXISTS (SELECT 1 FROM web_article_authors WHERE user_id = $1 AND article_id = $2)
  AND (SELECT count(*) FROM pwikit_profile_pin WHERE user_id = $1) < $4
ON CONFLICT DO NOTHING`)
	qUnpin = register("UnpinPage", `DELETE FROM pwikit_profile_pin WHERE user_id = $1 AND article_id = $2`)
)

func (d *DB) ProfileExtras(ctx context.Context, userID int64) ([]ProfileField, []ProfileLink, error) {
	var fields []ProfileField
	rows, err := d.pool.Query(ctx, qProfileFields, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("read profile fields of %d: %w", userID, err)
	}
	for rows.Next() {
		var f ProfileField
		if err := rows.Scan(&f.Label, &f.Value); err != nil {
			rows.Close()
			return nil, nil, err
		}
		fields = append(fields, f)
	}
	rows.Close()
	var links []ProfileLink
	rows, err = d.pool.Query(ctx, qProfileLinks, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("read profile links of %d: %w", userID, err)
	}
	defer rows.Close()
	for rows.Next() {
		var l ProfileLink
		if err := rows.Scan(&l.Platform, &l.Value); err != nil {
			return nil, nil, err
		}
		links = append(links, l)
	}
	return fields, links, rows.Err()
}

func (d *DB) SaveProfileExtras(ctx context.Context, userID int64, fields []ProfileField, links []ProfileLink) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin profile extras of %d: %w", userID, err)
	}
	defer tx.Rollback(context.WithoutCancel(ctx))
	if _, err := tx.Exec(ctx, qDropProfileFields, userID); err != nil {
		return err
	}
	for i, f := range fields {
		if _, err := tx.Exec(ctx, qAddProfileField, userID, i, f.Label, f.Value); err != nil {
			return fmt.Errorf("save profile field of %d: %w", userID, err)
		}
	}
	if _, err := tx.Exec(ctx, qDropProfileLinks, userID); err != nil {
		return err
	}
	for i, l := range links {
		if _, err := tx.Exec(ctx, qAddProfileLink, userID, i, l.Platform, l.Value); err != nil {
			return fmt.Errorf("save profile link of %d: %w", userID, err)
		}
	}
	return tx.Commit(ctx)
}

func (d *DB) AuthoredPages(ctx context.Context, userID int64, pinnedOnly bool, from, before *time.Time, limit int) ([]AuthoredPage, error) {
	rows, err := d.pool.Query(ctx, qAuthoredPages, userID, pinnedOnly, limit, from, before)
	if err != nil {
		return nil, fmt.Errorf("list pages of %d: %w", userID, err)
	}
	defer rows.Close()
	var out []AuthoredPage
	for rows.Next() {
		var p AuthoredPage
		if err := rows.Scan(&p.ArticleID, &p.Title, &p.Path, &p.SiteID, &p.Category, &p.SiteTitle, &p.SiteDomain, &p.CreatedAt, &p.Pinned); err != nil {
			return nil, fmt.Errorf("scan a page of %d: %w", userID, err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Pin only takes a page the user is an author of, and stops at MaxPins.
func (d *DB) Pin(ctx context.Context, userID, articleID int64, on bool) error {
	var err error
	if on {
		_, err = d.pool.Exec(ctx, qPin, userID, articleID, time.Now().UTC(), MaxPins)
	} else {
		_, err = d.pool.Exec(ctx, qUnpin, userID, articleID)
	}
	if err != nil {
		return fmt.Errorf("pin %d for %d: %w", articleID, userID, err)
	}
	return nil
}
