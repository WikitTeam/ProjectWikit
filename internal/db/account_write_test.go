package db

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"
)

func scratchName(t *testing.T) string {
	t.Helper()
	return "probe-acct-" + time.Now().Format("150405.000000")
}

func dropUser(t *testing.T, d *DB, id int64) {
	t.Helper()
	t.Cleanup(func() {
		ctx := context.Background()
		for _, sql := range []string{
			`DELETE FROM web_usernotificationmapping WHERE recipient_id = $1`,
			`DELETE FROM web_user_roles WHERE user_id = $1`,
			`DELETE FROM web_userticket WHERE author_id = $1`,
			`DELETE FROM web_invitelink WHERE target_id = $1`,
			`DELETE FROM web_user WHERE id = $1`,
		} {
			if _, err := d.pool.Exec(ctx, sql, id); err != nil {
				t.Errorf("clean up user %d err = %v, want nil", id, err)
			}
		}
	})
}

func TestCreateUser(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	name := scratchName(t)

	id, err := d.CreateUser(ctx, name, "Probe Account", "!", true, time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateUser() err = %v, want nil", err)
	}
	dropUser(t, d, id)

	got, err := d.UserByID(ctx, id)
	if err != nil {
		t.Fatalf("UserByID() err = %v, want nil", err)
	}
	if got.Username != name {
		t.Errorf("UserByID().Username = %q, want %q", got.Username, name)
	}
	if got.DisplayName != "Probe Account" {
		t.Errorf("UserByID().DisplayName = %q, want %q", got.DisplayName, "Probe Account")
	}
	if !got.IsActive {
		t.Errorf("UserByID().IsActive = false, want true")
	}

	taken, err := d.UsernameTaken(ctx, name)
	if err != nil {
		t.Fatalf("UsernameTaken() err = %v, want nil", err)
	}
	if !taken {
		t.Errorf("UsernameTaken(%q) = false, want true", name)
	}
}

func TestCreateInvitedUserIsNamedAfterItsID(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	email := scratchName(t) + "@example.invalid"

	id, err := d.CreateInvitedUser(ctx, email, time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateInvitedUser() err = %v, want nil", err)
	}
	dropUser(t, d, id)

	got, err := d.UserByID(ctx, id)
	if err != nil {
		t.Fatalf("UserByID() err = %v, want nil", err)
	}
	if want := "user-" + strconv.FormatInt(id, 10); got.Username != want {
		t.Errorf("UserByID().Username = %q, want %q", got.Username, want)
	}
	if got.IsActive {
		t.Errorf("UserByID().IsActive = true, want false")
	}

	found, err := d.UserByEmail(ctx, strings.ToUpper(email))
	if err != nil {
		t.Fatalf("UserByEmail() err = %v, want nil", err)
	}
	if found.ID != id {
		t.Errorf("UserByEmail().ID = %d, want %d", found.ID, id)
	}
}

func TestActivateUser(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	id, err := d.CreateInvitedUser(ctx, scratchName(t)+"@example.invalid", time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateInvitedUser() err = %v, want nil", err)
	}
	dropUser(t, d, id)

	name := scratchName(t)
	display := "Probe Display"
	if err := d.ActivateUser(ctx, id, name, &display, "hashed"); err != nil {
		t.Fatalf("ActivateUser() err = %v, want nil", err)
	}
	got, hash, err := d.UserForLogin(ctx, name)
	if err != nil {
		t.Fatalf("UserForLogin() err = %v, want nil", err)
	}
	if !got.IsActive {
		t.Errorf("UserForLogin().IsActive = false, want true")
	}
	if hash != "hashed" {
		t.Errorf("UserForLogin() hash = %q, want %q", hash, "hashed")
	}
	if got.DisplayName != display {
		t.Errorf("UserForLogin().DisplayName = %q, want %q", got.DisplayName, display)
	}
}

func TestActivateUserKeepsTheStoredDisplayName(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	id, err := d.CreateUser(ctx, scratchName(t), "Kept Name", "!", false, time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateUser() err = %v, want nil", err)
	}
	dropUser(t, d, id)

	name := scratchName(t) + "-b"
	if err := d.ActivateUser(ctx, id, name, nil, "hashed"); err != nil {
		t.Fatalf("ActivateUser() err = %v, want nil", err)
	}
	got, err := d.UserByID(ctx, id)
	if err != nil {
		t.Fatalf("UserByID() err = %v, want nil", err)
	}
	if got.DisplayName != "Kept Name" {
		t.Errorf("UserByID().DisplayName = %q, want %q", got.DisplayName, "Kept Name")
	}
}

func TestSetPasswordAndLastLogin(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	id, err := d.CreateUser(ctx, scratchName(t), "", "!", true, time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateUser() err = %v, want nil", err)
	}
	dropUser(t, d, id)

	if err := d.SetPassword(ctx, id, "fresh"); err != nil {
		t.Fatalf("SetPassword() err = %v, want nil", err)
	}
	when := time.Date(2026, 9, 5, 11, 4, 34, 0, time.UTC)
	if err := d.SetLastLogin(ctx, id, when); err != nil {
		t.Fatalf("SetLastLogin() err = %v, want nil", err)
	}
	hash, last, _, err := d.ResetFields(ctx, id)
	if err != nil {
		t.Fatalf("ResetFields() err = %v, want nil", err)
	}
	if hash != "fresh" {
		t.Errorf("ResetFields() hash = %q, want %q", hash, "fresh")
	}
	if last == nil || !last.UTC().Equal(when) {
		t.Errorf("ResetFields() last login = %v, want %v", last, when)
	}
}

func TestSetUserPreference(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	user := scratchUser(t, d, "probe-pref")
	t.Cleanup(func() {
		if _, err := d.pool.Exec(context.Background(),
			`DELETE FROM dynamic_preferences_users_userpreferencemodel WHERE instance_id = $1`, user); err != nil {
			t.Errorf("clean up preference err = %v, want nil", err)
		}
	})

	if _, err := d.UserPreference(ctx, user, "qol", "advanced_source_editor_enabled"); !errors.Is(err, ErrNotFound) {
		t.Errorf("UserPreference(unset) err = %v, want ErrNotFound", err)
	}
	for _, want := range []string{"True", "False"} {
		if err := d.SetUserPreference(ctx, user, "qol", "advanced_source_editor_enabled", want); err != nil {
			t.Fatalf("SetUserPreference(%q) err = %v, want nil", want, err)
		}
		got, err := d.UserPreference(ctx, user, "qol", "advanced_source_editor_enabled")
		if err != nil {
			t.Fatalf("UserPreference() err = %v, want nil", err)
		}
		if got != want {
			t.Errorf("UserPreference() = %q, want %q", got, want)
		}
	}
}

func TestTokenUsed(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	one := "probe-token-" + time.Now().Format("150405.000000")
	t.Cleanup(func() {
		if _, err := d.pool.Exec(context.Background(), `DELETE FROM web_usedtoken WHERE token = $1`, one); err != nil {
			t.Errorf("clean up token err = %v, want nil", err)
		}
	})

	used, err := d.TokenUsed(ctx, one)
	if err != nil {
		t.Fatalf("TokenUsed() err = %v, want nil", err)
	}
	if used {
		t.Errorf("TokenUsed(fresh) = true, want false")
	}
	if err := d.MarkTokenUsed(ctx, one, true); err != nil {
		t.Fatalf("MarkTokenUsed() err = %v, want nil", err)
	}
	used, err = d.TokenUsed(ctx, one)
	if err != nil {
		t.Fatalf("TokenUsed() err = %v, want nil", err)
	}
	if !used {
		t.Errorf("TokenUsed(spent) = false, want true")
	}
	other, err := d.TokenUsed(ctx, strings.ToUpper(one))
	if err != nil {
		t.Fatalf("TokenUsed(uppercased) err = %v, want nil", err)
	}
	if other {
		t.Errorf("TokenUsed(uppercased) = true, want false")
	}
}

func TestCreateTicket(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	author := scratchUser(t, d, "probe-ticket")

	id, err := d.CreateTicket(ctx, TicketKind, "subject", "body", "main", author, time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateTicket() err = %v, want nil", err)
	}
	t.Cleanup(func() {
		if _, err := d.pool.Exec(context.Background(), `DELETE FROM web_userticket WHERE id = $1`, id); err != nil {
			t.Errorf("clean up ticket err = %v, want nil", err)
		}
	})

	var kind, status string
	err = d.pool.QueryRow(ctx, `SELECT kind, status FROM web_userticket WHERE id = $1`, id).Scan(&kind, &status)
	if err != nil {
		t.Fatalf("read ticket err = %v, want nil", err)
	}
	if kind != TicketKind {
		t.Errorf("ticket kind = %q, want %q", kind, TicketKind)
	}
	if status != TicketPending {
		t.Errorf("ticket status = %q, want %q", status, TicketPending)
	}
}
