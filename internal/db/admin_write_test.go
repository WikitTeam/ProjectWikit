package db

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func scratchImportedUser(t *testing.T, d *DB, name string) int64 {
	t.Helper()
	ctx := context.Background()
	name = name + "-" + time.Now().Format("150405.000000")
	var id int64
	err := d.pool.QueryRow(ctx, `
INSERT INTO web_user (password, is_superuser, first_name, last_name, email, date_joined,
	username, wikidot_username, display_name, type, bio, is_forum_active, is_active,
	can_send_direct_messages, pending_email, previous_email)
VALUES ('!', false, '', '', '', now(), $1, $2, $3, 'wikidot', '', true, false, true, '', '')
RETURNING id`, strings.ToLower(name), name, name).Scan(&id)
	if err != nil {
		t.Fatalf("insert imported user err = %v, want nil", err)
	}
	t.Cleanup(func() {
		if _, err := d.pool.Exec(context.Background(), `DELETE FROM web_user WHERE id = $1`, id); err != nil {
			t.Errorf("delete imported user err = %v, want nil", err)
		}
	})
	return id
}

func TestUserToClaimMatchesTheArchiveSpelling(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	id := scratchImportedUser(t, d, "Probe Claim")

	stored, err := d.UserByID(ctx, id)
	if err != nil {
		t.Fatalf("UserByID() err = %v, want nil", err)
	}
	got, hash, err := d.UserToClaim(ctx, stored.Username, strings.ToLower(stored.WikidotUsername))
	if err != nil {
		t.Fatalf("UserToClaim() err = %v, want nil", err)
	}
	if got.ID != id {
		t.Errorf("UserToClaim().ID = %d, want %d", got.ID, id)
	}
	if hash != "!" {
		t.Errorf("UserToClaim() hash = %q, want %q", hash, "!")
	}
}

func TestUserToClaimIsNotFoundForAnUnknownName(t *testing.T) {
	d := writeTestDB(t)
	_, _, err := d.UserToClaim(context.Background(), "probe-no-such-name", "probe-no-such-name")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("UserToClaim() err = %v, want ErrNotFound", err)
	}
}

func TestSetSuperuserGoesBothWays(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	id := scratchImportedUser(t, d, "Probe Super")

	for _, want := range []bool{true, false} {
		if err := d.SetSuperuser(ctx, id, want); err != nil {
			t.Fatalf("SetSuperuser(%t) err = %v, want nil", want, err)
		}
		got, err := d.UserByID(ctx, id)
		if err != nil {
			t.Fatalf("UserByID() err = %v, want nil", err)
		}
		if got.IsSuperuser != want {
			t.Errorf("UserByID().IsSuperuser = %t, want %t", got.IsSuperuser, want)
		}
	}
}

func TestSetSuperuserIsNotFoundForAnUnknownID(t *testing.T) {
	d := writeTestDB(t)
	if err := d.SetSuperuser(context.Background(), -1, true); !errors.Is(err, ErrNotFound) {
		t.Errorf("SetSuperuser() err = %v, want ErrNotFound", err)
	}
}
