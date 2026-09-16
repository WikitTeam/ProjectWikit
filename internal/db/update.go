package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type UpdateState struct {
	CheckedAt         *time.Time
	CheckError        string
	NextCheckAt       *time.Time
	LatestVersion     string
	LatestPublishedAt *time.Time
	LatestPostgres    string
	LatestNotes       string
	ScheduledVersion  string
	ScheduledAt       *time.Time
	ScheduledByHand   bool
	PostponedUntil    *time.Time
	SkippedVersion    string
	FailedVersions    []string
	PinnedVersion     string
	LastFrom          string
	LastTo            string
	LastOutcome       string
	LastError         string
	LastAt            *time.Time
	RollbackVersion   string
	RollbackKind      string
	RollbackExpiresAt *time.Time
}

const updateColumns = `checked_at, check_error, next_check_at, latest_version, latest_published_at,
latest_postgres, latest_notes, scheduled_version, scheduled_at, postponed_until, skipped_version,
failed_versions, pinned_version, last_from, last_to, last_outcome, last_error, last_at,
rollback_version, rollback_kind, rollback_expires_at, scheduled_by_hand`

var (
	qUpdateState = register("UpdateState", `SELECT `+updateColumns+` FROM pwikit_update WHERE id = 1`)

	qSaveUpdateState = register("SaveUpdateState", `
INSERT INTO pwikit_update (id, `+updateColumns+`)
VALUES (1, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
ON CONFLICT (id) DO UPDATE SET
	checked_at = EXCLUDED.checked_at, check_error = EXCLUDED.check_error,
	next_check_at = EXCLUDED.next_check_at, latest_version = EXCLUDED.latest_version,
	latest_published_at = EXCLUDED.latest_published_at, latest_postgres = EXCLUDED.latest_postgres,
	latest_notes = EXCLUDED.latest_notes, scheduled_version = EXCLUDED.scheduled_version,
	scheduled_at = EXCLUDED.scheduled_at, postponed_until = EXCLUDED.postponed_until,
	skipped_version = EXCLUDED.skipped_version, failed_versions = EXCLUDED.failed_versions,
	pinned_version = EXCLUDED.pinned_version, last_from = EXCLUDED.last_from,
	last_to = EXCLUDED.last_to, last_outcome = EXCLUDED.last_outcome,
	last_error = EXCLUDED.last_error, last_at = EXCLUDED.last_at,
	rollback_version = EXCLUDED.rollback_version, rollback_kind = EXCLUDED.rollback_kind,
	rollback_expires_at = EXCLUDED.rollback_expires_at,
	scheduled_by_hand = EXCLUDED.scheduled_by_hand`)

	qSiteDomains = register("SiteDomains", `SELECT domain FROM web_site ORDER BY id`)

	qSuperuserEmails = register("SuperuserEmails", `
SELECT email FROM web_user
WHERE is_superuser AND is_active AND email <> ''
ORDER BY id`)
)

func (d *DB) UpdateState(ctx context.Context) (UpdateState, error) {
	var s UpdateState
	err := d.pool.QueryRow(ctx, qUpdateState).Scan(
		&s.CheckedAt, &s.CheckError, &s.NextCheckAt, &s.LatestVersion, &s.LatestPublishedAt,
		&s.LatestPostgres, &s.LatestNotes, &s.ScheduledVersion, &s.ScheduledAt, &s.PostponedUntil,
		&s.SkippedVersion, &s.FailedVersions, &s.PinnedVersion, &s.LastFrom, &s.LastTo,
		&s.LastOutcome, &s.LastError, &s.LastAt, &s.RollbackVersion, &s.RollbackKind, &s.RollbackExpiresAt,
		&s.ScheduledByHand)
	var pgErr *pgconn.PgError
	if errors.Is(err, pgx.ErrNoRows) || (errors.As(err, &pgErr) && pgErr.Code == "42P01") {
		return UpdateState{}, nil
	}
	if err != nil {
		return UpdateState{}, fmt.Errorf("read the update state: %w", err)
	}
	return s, nil
}

func (d *DB) SaveUpdateState(ctx context.Context, s UpdateState) error {
	if s.FailedVersions == nil {
		s.FailedVersions = []string{}
	}
	_, err := d.pool.Exec(ctx, qSaveUpdateState,
		s.CheckedAt, s.CheckError, s.NextCheckAt, s.LatestVersion, s.LatestPublishedAt,
		s.LatestPostgres, s.LatestNotes, s.ScheduledVersion, s.ScheduledAt, s.PostponedUntil,
		s.SkippedVersion, s.FailedVersions, s.PinnedVersion, s.LastFrom, s.LastTo,
		s.LastOutcome, s.LastError, s.LastAt, s.RollbackVersion, s.RollbackKind, s.RollbackExpiresAt,
		s.ScheduledByHand)
	if err != nil {
		return fmt.Errorf("write the update state: %w", err)
	}
	return nil
}

func (d *DB) SiteDomains(ctx context.Context) ([]string, error) {
	return d.strings(ctx, qSiteDomains, "list site domains")
}

func (d *DB) SuperuserEmails(ctx context.Context) ([]string, error) {
	return d.strings(ctx, qSuperuserEmails, "list superuser addresses")
}

func (d *DB) strings(ctx context.Context, query, what string) ([]string, error) {
	rows, err := d.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, rows.Err()
}
