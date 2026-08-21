package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

const (
	UserTypeNormal  = "normal"
	UserTypeWikidot = "wikidot"
	UserTypeSystem  = "system"
	UserTypeBot     = "bot"
)

type User struct {
	ID              int64
	Type            string
	Username        string
	WikidotUsername string
	DisplayName     string
	Avatar          string
}

const userColumns = `id, type, username, wikidot_username, display_name, avatar`

// Django's QuerySet.first() falls back to ordering by primary key when the
// queryset carries no ordering of its own, so these two are not stray sorts.
var qUserByName = register("UserByName", `
SELECT `+userColumns+`
FROM web_user
WHERE username = $1 OR wikidot_username = $1
ORDER BY id
LIMIT 1`)

var qUserByWikidotName = register("UserByWikidotName", `
SELECT `+userColumns+`
FROM web_user
WHERE type = 'wikidot' AND wikidot_username = $1
ORDER BY id
LIMIT 1`)

// UserByName looks up a canonical username against both the local and the
// Wikidot name. Callers pass wikidot.CanonicalizeUsername output.
func (d *DB) UserByName(ctx context.Context, canonical string) (*User, error) {
	return d.scanUser(ctx, qUserByName, canonical)
}

// UserByWikidotName resolves the wd: prefix, which may only ever match an
// imported Wikidot account.
func (d *DB) UserByWikidotName(ctx context.Context, canonical string) (*User, error) {
	return d.scanUser(ctx, qUserByWikidotName, canonical)
}

func (d *DB) scanUser(ctx context.Context, sql, canonical string) (*User, error) {
	var (
		u                                    User
		wikidotUsername, displayName, avatar *string
	)
	err := d.pool.QueryRow(ctx, sql, canonical).Scan(
		&u.ID, &u.Type, &u.Username, &wikidotUsername, &displayName, &avatar)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lookup user %q: %w", canonical, err)
	}
	u.WikidotUsername = deref(wikidotUsername)
	u.DisplayName = deref(displayName)
	u.Avatar = deref(avatar)
	return &u, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
