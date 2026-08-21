package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

var qSessionByKey = register("SessionByKey", `
SELECT session_data, expire_date
FROM django_session
WHERE session_key = $1 AND expire_date > now()`)

func (d *DB) SessionByKey(ctx context.Context, key string) (string, time.Time, error) {
	var (
		data    string
		expires time.Time
	)
	err := d.pool.QueryRow(ctx, qSessionByKey, key).Scan(&data, &expires)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", time.Time{}, ErrNotFound
	}
	if err != nil {
		return "", time.Time{}, fmt.Errorf("lookup session: %w", err)
	}
	return data, expires, nil
}

var qSaveSession = register("SaveSession", `
INSERT INTO django_session (session_key, session_data, expire_date)
VALUES ($1, $2, $3)
ON CONFLICT (session_key) DO UPDATE
SET session_data = EXCLUDED.session_data, expire_date = EXCLUDED.expire_date`)

func (d *DB) SaveSession(ctx context.Context, key, data string, expires time.Time) error {
	if _, err := d.pool.Exec(ctx, qSaveSession, key, data, expires); err != nil {
		return fmt.Errorf("save session: %w", err)
	}
	return nil
}

var qDeleteSession = register("DeleteSession", `DELETE FROM django_session WHERE session_key = $1`)

func (d *DB) DeleteSession(ctx context.Context, key string) error {
	if _, err := d.pool.Exec(ctx, qDeleteSession, key); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
