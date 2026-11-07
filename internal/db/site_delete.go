package db

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/jackc/pgx/v5"
)

type SiteSize struct {
	Pages   int
	Posts   int
	Files   int
	Members int
}

var qSiteSize = register("SiteSize", `
SELECT
  (SELECT count(*) FROM web_article WHERE site_id = $1),
  (SELECT count(*) FROM web_forumpost p JOIN web_forumthread t ON t.id = p.thread_id WHERE t.site_id = $1),
  (SELECT count(*) FROM web_file f JOIN web_article a ON a.id = f.article_id WHERE a.site_id = $1),
  (SELECT count(DISTINCT ur.user_id) FROM web_user_roles ur JOIN web_role r ON r.id = ur.role_id WHERE r.site_id = $1)`)

func (d *DB) SiteSize(ctx context.Context, siteID int64) (SiteSize, error) {
	var s SiteSize
	if err := d.pool.QueryRow(ctx, qSiteSize, siteID).Scan(&s.Pages, &s.Posts, &s.Files, &s.Members); err != nil {
		return s, fmt.Errorf("measure site %d: %w", siteID, err)
	}
	return s, nil
}

// The same sets a single site backup is cut by, kept as temporary tables so a
// rule still finds its rows after the rows it was derived from are gone.
var deleteScope = []struct{ name, query string }{
	{"s", `SELECT id FROM web_site WHERE slug = $1`},
	{"art", `SELECT id FROM web_article WHERE site_id = (SELECT id FROM s)`},
	{"cat", `SELECT id FROM web_category WHERE site_id = (SELECT id FROM s)`},
	{"rol", `SELECT id FROM web_role WHERE site_id = (SELECT id FROM s)`},
	{"sec", `SELECT id FROM web_forumsection WHERE site_id = (SELECT id FROM s)`},
	{"fcat", `SELECT id FROM web_forumcategory WHERE section_id IN (SELECT id FROM sec)`},
	{"thr", `SELECT id FROM web_forumthread WHERE site_id = (SELECT id FROM s)`},
	{"pst", `SELECT id FROM web_forumpost WHERE thread_id IN (SELECT id FROM thr)`},
	{"ovr", `SELECT rolepermissionsoverride_id AS id FROM web_category_permissions_override
             WHERE category_id IN (SELECT id FROM cat)
             UNION SELECT rolepermissionsoverride_id FROM web_forumsection_permissions_override
             WHERE forumsection_id IN (SELECT id FROM sec)`},
}

var qDanglingKeys = `
SELECT c.conrelid::regclass::text, a.attname, c.confrelid::regclass::text, pa.attname, NOT a.attnotnull
FROM pg_constraint c
JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = c.conkey[1]
JOIN pg_attribute pa ON pa.attrelid = c.confrelid AND pa.attnum = c.confkey[1]
WHERE c.contype = 'f' AND array_length(c.conkey, 1) = 1 AND c.confdeltype IN ('a', 'r')
  AND c.confrelid::regclass::text = ANY($1)`

var ErrSiteMissing = errors.New("no such site")

// Rows elsewhere that pointed at what went are cleared, or removed when the
// pointer cannot be empty. An account is never removed.
func (d *DB) DeleteSite(ctx context.Context, slug string) ([]string, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.WithoutCancel(ctx))

	if _, err := tx.Exec(ctx, `SET CONSTRAINTS ALL DEFERRED`); err != nil {
		return nil, err
	}
	for _, s := range deleteScope {
		args := []any{}
		if s.name == "s" {
			args = append(args, slug)
		}
		if _, err := tx.Exec(ctx, `CREATE TEMP TABLE `+s.name+` ON COMMIT DROP AS `+s.query, args...); err != nil {
			return nil, fmt.Errorf("gather %s: %w", s.name, err)
		}
	}
	var found int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM s`).Scan(&found); err != nil {
		return nil, err
	}
	if found != 1 {
		return nil, ErrSiteMissing
	}

	var media []string
	rows, err := tx.Query(ctx, `SELECT media_name FROM web_article WHERE id IN (SELECT id FROM art)`)
	if err != nil {
		return nil, err
	}
	if media, err = pgx.CollectRows(rows, pgx.RowTo[string]); err != nil {
		return nil, err
	}

	var tables []string
	for name, r := range rules {
		if r.how == owned && name != "web_site" {
			tables = append(tables, name)
		}
	}
	sort.Strings(tables)
	tables = append(tables, "web_site")

	// A notice with no site shows on every site, so the site's notices go
	// rather than lose their site.
	for _, stmt := range []string{
		`DELETE FROM web_usernotification WHERE site_id = (SELECT id FROM s)`,
		`UPDATE web_externallink SET to_site_id = NULL WHERE to_site_id = (SELECT id FROM s)`,
	} {
		if _, err := tx.Exec(ctx, stmt); err != nil {
			return nil, fmt.Errorf("let go of the site: %w", err)
		}
	}
	emptied, err := clearPointers(ctx, tx, tables)
	if err != nil {
		return nil, err
	}
	for _, name := range tables {
		if _, err := tx.Exec(ctx, `DELETE FROM `+QuoteName(name)+` WHERE `+rules[name].where); err != nil {
			return nil, fmt.Errorf("delete from %s: %w", name, err)
		}
	}

	touched := append(append(slices.Clone(tables), "web_usernotification"), emptied...)
	if err := clearDangling(ctx, tx, touched); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit site deletion: %w", err)
	}
	return media, nil
}

var qBlockingKeys = `
SELECT c.conrelid::regclass::text, a.attname, c.confrelid::regclass::text, pa.attname, NOT a.attnotnull
FROM pg_constraint c
JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = c.conkey[1]
JOIN pg_attribute pa ON pa.attrelid = c.confrelid AND pa.attnum = c.confkey[1]
WHERE c.contype = 'f' AND array_length(c.conkey, 1) = 1 AND c.confdeltype IN ('a', 'r')
  AND NOT c.condeferrable
  AND c.confrelid::regclass::text = ANY($1) AND NOT c.conrelid::regclass::text = ANY($1)`

// A key that is checked at once would stop the deletion halfway, so whatever
// outside the site points at its rows lets go first.
func clearPointers(ctx context.Context, tx pgx.Tx, tables []string) ([]string, error) {
	rows, err := tx.Query(ctx, qBlockingKeys, tables)
	if err != nil {
		return nil, err
	}
	type key struct {
		child, col, parent, pcol string
		nullable                 bool
	}
	keys, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (key, error) {
		var k key
		err := row.Scan(&k.child, &k.col, &k.parent, &k.pcol, &k.nullable)
		return k, err
	})
	if err != nil {
		return nil, err
	}
	var emptied []string
	for _, k := range keys {
		hit := ` WHERE ` + QuoteName(k.col) + ` IN (SELECT ` + QuoteName(k.pcol) + ` FROM ` + k.parent +
			` WHERE ` + rules[k.parent].where + `)`
		stmt := `DELETE FROM ` + k.child + hit
		if k.nullable {
			stmt = `UPDATE ` + k.child + ` SET ` + QuoteName(k.col) + ` = NULL` + hit
		} else if k.child == "web_user" {
			return nil, fmt.Errorf("an account still needs %s.%s; nothing was deleted", k.child, k.col)
		} else {
			emptied = append(emptied, k.child)
		}
		if _, err := tx.Exec(ctx, stmt); err != nil {
			return nil, fmt.Errorf("let go of %s.%s: %w", k.child, k.col, err)
		}
	}
	return emptied, nil
}

// Keys the database cascades by itself are left to it. The rest are followed
// until nothing more points at a removed row.
func clearDangling(ctx context.Context, tx pgx.Tx, touched []string) error {
	seen := map[string]bool{}
	for len(touched) > 0 {
		rows, err := tx.Query(ctx, qDanglingKeys, touched)
		if err != nil {
			return err
		}
		type key struct {
			child, col, parent, pcol string
			nullable                 bool
		}
		keys, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (key, error) {
			var k key
			err := row.Scan(&k.child, &k.col, &k.parent, &k.pcol, &k.nullable)
			return k, err
		})
		if err != nil {
			return err
		}
		touched = nil
		for _, k := range keys {
			gone := ` WHERE ` + QuoteName(k.col) + ` IS NOT NULL AND NOT EXISTS (SELECT 1 FROM ` + k.parent +
				` p WHERE p.` + QuoteName(k.pcol) + ` = ` + k.child + `.` + QuoteName(k.col) + `)`
			if k.nullable {
				if _, err := tx.Exec(ctx, `UPDATE `+k.child+` SET `+QuoteName(k.col)+` = NULL`+gone); err != nil {
					return fmt.Errorf("clear %s.%s: %w", k.child, k.col, err)
				}
				continue
			}
			if k.child == "web_user" {
				return fmt.Errorf("an account still needs %s.%s; nothing was deleted", k.child, k.col)
			}
			tag, err := tx.Exec(ctx, `DELETE FROM `+k.child+gone)
			if err != nil {
				return fmt.Errorf("delete from %s: %w", k.child, err)
			}
			if tag.RowsAffected() > 0 && !seen[k.child] {
				seen[k.child] = true
				touched = append(touched, k.child)
			}
		}
	}
	return nil
}
