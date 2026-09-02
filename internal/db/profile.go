package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type Profile struct {
	User
	Bio        string
	FirstName  string
	LastName   string
	DateJoined time.Time
}

const profileColumns = `id, type, username, wikidot_username, display_name, avatar,
       is_active, inactive_until, is_superuser, is_forum_active, forum_inactive_until,
       bio, first_name, last_name, date_joined`

var qProfileByID = register("ProfileByID", `
SELECT `+profileColumns+`
FROM web_user
WHERE id = $1`)

func (d *DB) ProfileByID(ctx context.Context, id int64) (*Profile, error) {
	return d.profile(ctx, qProfileByID, id)
}

var qProfileByName = register("ProfileByName", `
SELECT `+profileColumns+`
FROM web_user
WHERE username = $1 OR wikidot_username = $1
LIMIT 1`)

func (d *DB) ProfileByName(ctx context.Context, name string) (*Profile, error) {
	return d.profile(ctx, qProfileByName, name)
}

func (d *DB) profile(ctx context.Context, query string, arg any) (*Profile, error) {
	var p Profile
	dest, finish := userDest(&p.User)
	dest = append(dest, &p.Bio, &p.FirstName, &p.LastName, &p.DateJoined)

	err := d.pool.QueryRow(ctx, query, arg).Scan(dest...)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lookup profile %v: %w", arg, err)
	}
	finish()
	return &p, nil
}

var qDirectMessageBlocked = register("DirectMessageBlocked", `
SELECT EXISTS (SELECT 1 FROM web_directmessageblock
               WHERE blocker_id = $1 AND blocked_id = $2)`)

func (d *DB) DirectMessageBlocked(ctx context.Context, blockerID, blockedID int64) (bool, error) {
	var blocked bool
	if err := d.pool.QueryRow(ctx, qDirectMessageBlocked, blockerID, blockedID).Scan(&blocked); err != nil {
		return false, fmt.Errorf("check block of %d by %d: %w", blockedID, blockerID, err)
	}
	return blocked, nil
}
