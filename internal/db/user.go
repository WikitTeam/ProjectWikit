package db

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	IsActive        bool
	InactiveUntil   *time.Time
	IsSuperuser     bool

	IsForumActive      bool
	ForumInactiveUntil *time.Time
}

// ActiveAt reproduces User.__init__, where a deadline in the future overrides
// the stored flag in both directions: is_active is ignored whenever
// inactive_until is set.
func (u *User) ActiveAt(now time.Time) bool {
	if u.InactiveUntil == nil {
		return u.IsActive
	}
	return now.After(*u.InactiveUntil)
}

func (u *User) ForumActiveAt(now time.Time) bool {
	if u.ForumInactiveUntil == nil {
		return u.IsForumActive
	}
	return now.After(*u.ForumInactiveUntil)
}

func (u *User) DisplayLabel() string {
	if u.Type == UserTypeWikidot {
		return "wd:" + firstNonEmpty(u.DisplayName, u.WikidotUsername, u.Username)
	}
	return firstNonEmpty(u.DisplayName, u.Username)
}

func (u *User) URLName() string {
	if u.Type == UserTypeWikidot {
		return firstNonEmpty(u.WikidotUsername, u.Username)
	}
	return u.Username
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

const userColumns = `id, type, username, wikidot_username, display_name, avatar, is_active, inactive_until, is_superuser, is_forum_active, forum_inactive_until`

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

var qUserByUsername = register("UserByUsername", `
SELECT `+userColumns+`
FROM web_user
WHERE username = $1
ORDER BY id
LIMIT 1`)

// A page list that filters by author names the account as it is spelled here,
// never as it was on the site an imported account came from.
func (d *DB) UserByUsername(ctx context.Context, name string) (*User, error) {
	return d.scanUser(ctx, qUserByUsername, name)
}

var qUserByDisplayName = register("UserByDisplayName", `
SELECT `+userColumns+`
FROM web_user
WHERE lower(display_name) = lower($1)
ORDER BY id
LIMIT 1`)

// The oldest row wins so a later account cannot take over a name a page already
// points at.
func (d *DB) UserByDisplayName(ctx context.Context, name string) (*User, error) {
	return d.scanUser(ctx, qUserByDisplayName, name)
}

// UserByWikidotName resolves the wd: prefix, which may only ever match an
// imported Wikidot account.
func (d *DB) UserByWikidotName(ctx context.Context, canonical string) (*User, error) {
	return d.scanUser(ctx, qUserByWikidotName, canonical)
}

func (d *DB) scanUser(ctx context.Context, sql, canonical string) (*User, error) {
	var u User
	dest, finish := userDest(&u)
	err := d.pool.QueryRow(ctx, sql, canonical).Scan(dest...)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lookup user %q: %w", canonical, err)
	}
	finish()
	return &u, nil
}

// userDest lists the scan targets for userColumns, in order. Three of the
// columns are nullable text, so finish has to run before the user is read.
func userDest(u *User) (dest []any, finish func()) {
	var wikidotUsername, displayName, avatar *string
	dest = []any{
		&u.ID, &u.Type, &u.Username, &wikidotUsername, &displayName, &avatar,
		&u.IsActive, &u.InactiveUntil, &u.IsSuperuser,
		&u.IsForumActive, &u.ForumInactiveUntil,
	}
	return dest, func() {
		u.WikidotUsername = deref(wikidotUsername)
		u.DisplayName = deref(displayName)
		u.Avatar = deref(avatar)
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

var qUserForSession = register("UserForSession", `
SELECT `+userColumns+`, password
FROM web_user
WHERE id = $1`)

// UserForSession returns the password hash alongside the user because the
// session carries a hash of it; a session opened under an older password has to
// stop working. The hash is kept out of User so it cannot travel by accident.
func (d *DB) UserForSession(ctx context.Context, id int64) (*User, string, error) {
	var (
		u        User
		password string
	)
	dest, finish := userDest(&u)
	err := d.pool.QueryRow(ctx, qUserForSession, id).Scan(append(dest, &password)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", ErrNotFound
	}
	if err != nil {
		return nil, "", fmt.Errorf("lookup user %d: %w", id, err)
	}
	finish()
	return &u, password, nil
}
