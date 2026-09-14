package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type ThemeRow struct {
	ID          int64
	Name        string
	Slug        string
	Mode        string
	CSS         string
	ExternalURL string
	UpdatedAt   time.Time
}

var qThemes = register("Themes", `
SELECT id, name, slug, mode, css, external_url, updated_at
FROM web_theme WHERE site_id = $1 ORDER BY name, id`)

func (d *DB) Themes(ctx context.Context, siteID int64) ([]ThemeRow, error) {
	rows, err := d.pool.Query(ctx, qThemes, siteID)
	if err != nil {
		return nil, fmt.Errorf("list themes: %w", err)
	}
	defer rows.Close()

	var out []ThemeRow
	for rows.Next() {
		var t ThemeRow
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.Mode, &t.CSS, &t.ExternalURL, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

var qThemeRow = register("ThemeRow", `
SELECT id, name, slug, mode, css, external_url, updated_at
FROM web_theme WHERE id = $1 AND site_id = $2`)

func (d *DB) Theme(ctx context.Context, siteID, id int64) (ThemeRow, error) {
	var t ThemeRow
	err := d.pool.QueryRow(ctx, qThemeRow, id, siteID).
		Scan(&t.ID, &t.Name, &t.Slug, &t.Mode, &t.CSS, &t.ExternalURL, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ThemeRow{}, ErrNotFound
	}
	if err != nil {
		return ThemeRow{}, fmt.Errorf("read theme %d: %w", id, err)
	}
	return t, nil
}

var qInsertTheme = register("InsertTheme", `
INSERT INTO web_theme (name, slug, mode, css, external_url, updated_at, site_id)
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`)

var qUpdateTheme = register("UpdateTheme", `
UPDATE web_theme SET name = $2, slug = $3, mode = $4, css = $5, external_url = $6, updated_at = $7
WHERE id = $1 AND site_id = $8`)

func (d *DB) SaveTheme(ctx context.Context, siteID int64, t ThemeRow) (int64, error) {
	at := time.Now()
	if t.ID == 0 {
		var id int64
		err := d.pool.QueryRow(ctx, qInsertTheme, t.Name, t.Slug, t.Mode, t.CSS, t.ExternalURL, at, siteID).Scan(&id)
		if err != nil {
			return 0, fmt.Errorf("create theme %q: %w", t.Slug, err)
		}
		return id, nil
	}
	if _, err := d.pool.Exec(ctx, qUpdateTheme, t.ID, t.Name, t.Slug, t.Mode, t.CSS, t.ExternalURL, at, siteID); err != nil {
		return 0, fmt.Errorf("update theme %d: %w", t.ID, err)
	}
	return t.ID, nil
}

var qDeleteTheme = register("DeleteTheme", `DELETE FROM web_theme WHERE id = $1 AND site_id = $2`)

var qDetachTheme = register("DetachTheme", `UPDATE web_site SET
	active_theme_id = CASE WHEN active_theme_id = $1 THEN NULL ELSE active_theme_id END,
	system_theme_id = CASE WHEN system_theme_id = $1 THEN NULL ELSE system_theme_id END`)

func (d *DB) DeleteTheme(ctx context.Context, siteID, id int64) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin deleting theme %d: %w", id, err)
	}
	defer tx.Rollback(context.WithoutCancel(ctx))

	if _, err := tx.Exec(ctx, qDetachTheme, id); err != nil {
		return fmt.Errorf("detach theme %d: %w", id, err)
	}
	if _, err := tx.Exec(ctx, qDeleteTheme, id, siteID); err != nil {
		return fmt.Errorf("delete theme %d: %w", id, err)
	}
	return tx.Commit(ctx)
}
