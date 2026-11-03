package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

var qEnabledPlugins = register("EnabledPlugins", `
SELECT plugin FROM pwikit_plugin_site WHERE site_id = $1 ORDER BY plugin`)

func (d *DB) EnabledPlugins(ctx context.Context, siteID int64) ([]string, error) {
	rows, err := d.pool.Query(ctx, qEnabledPlugins, siteID)
	if err != nil {
		return nil, fmt.Errorf("enabled plugins of site %d: %w", siteID, err)
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

var (
	qEnablePlugin = register("EnablePlugin", `
INSERT INTO pwikit_plugin_site (site_id, plugin) VALUES ($1, $2) ON CONFLICT DO NOTHING`)
	qDisablePlugin = register("DisablePlugin", `
DELETE FROM pwikit_plugin_site WHERE site_id = $1 AND plugin = $2`)
)

func (d *DB) SetPluginEnabled(ctx context.Context, siteID int64, plugin string, on bool) error {
	q := qDisablePlugin
	if on {
		q = qEnablePlugin
	}
	if _, err := d.pool.Exec(ctx, q, siteID, plugin); err != nil {
		return fmt.Errorf("switch plugin %q on site %d: %w", plugin, siteID, err)
	}
	return nil
}

var (
	qPluginKVGet = register("PluginKVGet", `
SELECT value FROM pwikit_plugin_kv WHERE site_id = $1 AND plugin = $2 AND key = $3`)
	qPluginKVSet = register("PluginKVSet", `
INSERT INTO pwikit_plugin_kv (site_id, plugin, key, value) VALUES ($1, $2, $3, $4)
ON CONFLICT (site_id, plugin, key) DO UPDATE SET value = excluded.value, updated_at = now()`)
	qPluginKVDelete = register("PluginKVDelete", `
DELETE FROM pwikit_plugin_kv WHERE site_id = $1 AND plugin = $2 AND key = $3`)
	qPluginKVCount = register("PluginKVCount", `
SELECT count(*) FROM pwikit_plugin_kv WHERE site_id = $1 AND plugin = $2`)
)

func (d *DB) PluginKVGet(ctx context.Context, siteID int64, plugin, key string) ([]byte, error) {
	var value []byte
	err := d.pool.QueryRow(ctx, qPluginKVGet, siteID, plugin, key).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return value, err
}

func (d *DB) PluginKVSet(ctx context.Context, siteID int64, plugin, key string, value []byte) error {
	_, err := d.pool.Exec(ctx, qPluginKVSet, siteID, plugin, key, value)
	return err
}

func (d *DB) PluginKVDelete(ctx context.Context, siteID int64, plugin, key string) error {
	_, err := d.pool.Exec(ctx, qPluginKVDelete, siteID, plugin, key)
	return err
}

func (d *DB) PluginKVCount(ctx context.Context, siteID int64, plugin string) (int, error) {
	var n int
	err := d.pool.QueryRow(ctx, qPluginKVCount, siteID, plugin).Scan(&n)
	return n, err
}
