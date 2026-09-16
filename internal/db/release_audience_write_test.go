package db

import (
	"context"
	"slices"
	"testing"
	"time"
)

func TestReleaseAudienceReachesEveryAdmin(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	site := scratchSite(t, d)
	staffRole := scratchRole(t, d, site)
	if _, err := d.pool.Exec(ctx, `UPDATE web_role SET is_staff = true WHERE id = $1`, staffRole); err != nil {
		t.Fatalf("mark role staff err = %v, want nil", err)
	}

	user := func(name string, active bool) int64 {
		id, err := d.CreateUser(ctx, scratchName(t)+name, name, "!", active, time.Now().UTC())
		if err != nil {
			t.Fatalf("CreateUser(%s) err = %v, want nil", name, err)
		}
		dropUser(t, d, id)
		return id
	}
	super := user("super", true)
	if _, err := d.pool.Exec(ctx, `UPDATE web_user SET is_superuser = true WHERE id = $1`, super); err != nil {
		t.Fatalf("mark superuser err = %v, want nil", err)
	}
	staff := user("staff", true)
	banned := user("banned", false)
	plain := user("plain", true)
	for _, id := range []int64{staff, banned} {
		if err := d.GrantRole(ctx, site, id, staffRole); err != nil {
			t.Fatalf("GrantRole(%d) err = %v, want nil", id, err)
		}
	}

	got, err := d.ReleaseAudience(ctx)
	if err != nil {
		t.Fatalf("ReleaseAudience() err = %v, want nil", err)
	}
	cases := []struct {
		name string
		id   int64
		want bool
	}{
		{"superuser", super, true},
		{"staff on one site", staff, true},
		{"inactive staff", banned, false},
		{"ordinary user", plain, false},
	}
	for _, c := range cases {
		if in := slices.Contains(got, c.id); in != c.want {
			t.Errorf("ReleaseAudience() contains %s = %v, want %v", c.name, in, c.want)
		}
	}
}
