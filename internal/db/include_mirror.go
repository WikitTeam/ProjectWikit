package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	MirrorOK      = "ok"
	MirrorMissing = "missing"
	MirrorFailed  = "failed"
)

type Mirror struct {
	ID        int64
	Wiki      string
	Page      string
	RemoteID  *int64
	Source    string
	Status    string
	Problem   string
	FetchedAt time.Time
	Refs      int
}

type IncludeRef struct {
	ArticleID int64
	Wiki      string
	Page      string
}

type MissingInclude struct {
	Wiki    string
	Page    string
	Status  string
	Problem string
	Pages   []string
}

var (
	qSiteSources = register("SiteSources", `
SELECT DISTINCT ON (a.id) a.id, v.source
FROM web_articleversion v
JOIN web_article a ON a.id = v.article_id
WHERE a.site_id = $1
ORDER BY a.id, v.created_at DESC`)
	qDropIncludeRefs = register("DropIncludeRefs", `
DELETE FROM pwikit_include_ref WHERE site_id = $1`)
	qAddIncludeRefs = register("AddIncludeRefs", `
INSERT INTO pwikit_include_ref (site_id, article_id, wiki, page)
SELECT $1, a.id, r.wiki, r.page
FROM unnest($2::bigint[], $3::text[], $4::text[]) AS r(article_id, wiki, page)
JOIN web_article a ON a.id = r.article_id AND a.site_id = $1
ON CONFLICT DO NOTHING`)
	qMirrors = register("Mirrors", `
SELECT m.id, m.wiki, m.page, m.remote_id, m.status, m.problem, m.fetched_at,
	(SELECT count(*) FROM pwikit_include_ref r WHERE r.site_id = $1 AND r.wiki = m.wiki AND r.page = m.page)
FROM pwikit_include_mirror m
ORDER BY m.wiki, m.page`)
	qMirror = register("Mirror", `
SELECT id, wiki, page, remote_id, source, status, problem, fetched_at
FROM pwikit_include_mirror WHERE id = $1`)
	qMirrorByName = register("MirrorByName", `
SELECT id, wiki, page, remote_id, source, status, problem, fetched_at
FROM pwikit_include_mirror WHERE wiki = $1 AND page = $2`)
	qMirrorSource = register("MirrorSource", `
SELECT source FROM pwikit_include_mirror WHERE wiki = $1 AND page = $2 AND status = 'ok'`)
	qMirrorReferrers = register("MirrorReferrers", `
SELECT regexp_replace(a.complete_full_name, '^_default:', '') AS name
FROM pwikit_include_ref r
JOIN web_article a ON a.id = r.article_id
WHERE r.site_id = $1 AND r.wiki = $2 AND r.page = $3
ORDER BY name`)
	qMissingIncludes = register("MissingIncludes", `
SELECT r.wiki, r.page, coalesce(m.status, ''), coalesce(m.problem, ''),
	array_agg(DISTINCT regexp_replace(a.complete_full_name, '^_default:', '')
		ORDER BY regexp_replace(a.complete_full_name, '^_default:', ''))
FROM pwikit_include_ref r
JOIN web_article a ON a.id = r.article_id
LEFT JOIN pwikit_include_mirror m ON m.wiki = r.wiki AND m.page = r.page
WHERE ($1::bigint IS NULL OR r.site_id = $1)
  AND NOT EXISTS (SELECT 1 FROM web_site s WHERE lower(s.slug) = r.wiki)
  AND (m.id IS NULL OR m.status <> 'ok')
GROUP BY 1, 2, 3, 4
ORDER BY 1, 2`)
	qSaveMirror = register("SaveMirror", `
INSERT INTO pwikit_include_mirror (wiki, page, remote_id, source, status, problem, fetched_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (wiki, page) DO UPDATE SET
	remote_id = coalesce(EXCLUDED.remote_id, pwikit_include_mirror.remote_id),
	source = CASE WHEN EXCLUDED.status = 'ok' THEN EXCLUDED.source ELSE pwikit_include_mirror.source END,
	status = CASE WHEN EXCLUDED.status <> 'ok' AND pwikit_include_mirror.status = 'ok'
		THEN pwikit_include_mirror.status ELSE EXCLUDED.status END,
	problem = EXCLUDED.problem,
	fetched_at = CASE WHEN EXCLUDED.status = 'ok' OR pwikit_include_mirror.status <> 'ok'
		THEN EXCLUDED.fetched_at ELSE pwikit_include_mirror.fetched_at END`)
	qDeleteMirror = register("DeleteMirror", `
DELETE FROM pwikit_include_mirror WHERE id = $1`)
	qInstanceSiteSlugs = register("InstanceSiteSlugs", `
SELECT lower(slug) FROM web_site`)
)

func (d *DB) SiteSources(ctx context.Context, siteID int64) (map[int64]string, error) {
	rows, err := d.pool.Query(ctx, qSiteSources, siteID)
	if err != nil {
		return nil, fmt.Errorf("read the sources of site %d: %w", siteID, err)
	}
	defer rows.Close()
	out := map[int64]string{}
	for rows.Next() {
		var id int64
		var source string
		if err := rows.Scan(&id, &source); err != nil {
			return nil, err
		}
		out[id] = source
	}
	return out, rows.Err()
}

func (d *DB) ReplaceIncludeRefs(ctx context.Context, siteID int64, refs []IncludeRef) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin the include refs of site %d: %w", siteID, err)
	}
	defer tx.Rollback(context.WithoutCancel(ctx))

	if _, err := tx.Exec(ctx, qDropIncludeRefs, siteID); err != nil {
		return fmt.Errorf("clear the include refs of site %d: %w", siteID, err)
	}
	ids := make([]int64, len(refs))
	wikis := make([]string, len(refs))
	pages := make([]string, len(refs))
	for i, r := range refs {
		ids[i], wikis[i], pages[i] = r.ArticleID, r.Wiki, r.Page
	}
	if _, err := tx.Exec(ctx, qAddIncludeRefs, siteID, ids, wikis, pages); err != nil {
		return fmt.Errorf("store the include refs of site %d: %w", siteID, err)
	}
	return tx.Commit(ctx)
}

func (d *DB) Mirrors(ctx context.Context, siteID int64) ([]Mirror, error) {
	rows, err := d.pool.Query(ctx, qMirrors, siteID)
	if err != nil {
		return nil, fmt.Errorf("list mirrors: %w", err)
	}
	defer rows.Close()
	var out []Mirror
	for rows.Next() {
		var m Mirror
		if err := rows.Scan(&m.ID, &m.Wiki, &m.Page, &m.RemoteID, &m.Status, &m.Problem, &m.FetchedAt, &m.Refs); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func scanMirror(row pgx.Row) (Mirror, error) {
	var m Mirror
	err := row.Scan(&m.ID, &m.Wiki, &m.Page, &m.RemoteID, &m.Source, &m.Status, &m.Problem, &m.FetchedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Mirror{}, ErrNotFound
	}
	return m, err
}

func (d *DB) Mirror(ctx context.Context, id int64) (Mirror, error) {
	return scanMirror(d.pool.QueryRow(ctx, qMirror, id))
}

func (d *DB) MirrorByName(ctx context.Context, wiki, page string) (Mirror, error) {
	return scanMirror(d.pool.QueryRow(ctx, qMirrorByName, wiki, page))
}

func (d *DB) MirrorSource(ctx context.Context, wiki, page string) (string, bool, error) {
	var source string
	err := d.pool.QueryRow(ctx, qMirrorSource, wiki, page).Scan(&source)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read the mirror of %s:%s: %w", wiki, page, err)
	}
	return source, true, nil
}

func (d *DB) MirrorReferrers(ctx context.Context, siteID int64, wiki, page string) ([]string, error) {
	rows, err := d.pool.Query(ctx, qMirrorReferrers, siteID, wiki, page)
	if err != nil {
		return nil, fmt.Errorf("list the pages including %s:%s: %w", wiki, page, err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

func (d *DB) MissingIncludes(ctx context.Context, siteID *int64) ([]MissingInclude, error) {
	rows, err := d.pool.Query(ctx, qMissingIncludes, siteID)
	if err != nil {
		return nil, fmt.Errorf("list missing includes: %w", err)
	}
	defer rows.Close()
	var out []MissingInclude
	for rows.Next() {
		var m MissingInclude
		if err := rows.Scan(&m.Wiki, &m.Page, &m.Status, &m.Problem, &m.Pages); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (d *DB) SaveMirror(ctx context.Context, m Mirror) error {
	if _, err := d.pool.Exec(ctx, qSaveMirror, m.Wiki, m.Page, m.RemoteID, m.Source, m.Status, m.Problem, m.FetchedAt); err != nil {
		return fmt.Errorf("save the mirror of %s:%s: %w", m.Wiki, m.Page, err)
	}
	return nil
}

func (d *DB) DeleteMirror(ctx context.Context, id int64) error {
	if _, err := d.pool.Exec(ctx, qDeleteMirror, id); err != nil {
		return fmt.Errorf("delete mirror %d: %w", id, err)
	}
	return nil
}

func (d *DB) InstanceSiteSlugs(ctx context.Context) (map[string]bool, error) {
	rows, err := d.pool.Query(ctx, qInstanceSiteSlugs)
	if err != nil {
		return nil, fmt.Errorf("list site slugs: %w", err)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, err
		}
		out[slug] = true
	}
	return out, rows.Err()
}
