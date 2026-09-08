package db

import (
	"context"
	"testing"
	"time"
)

func TestRolesByUserWithoutRoles(t *testing.T) {
	d := newTestDB(t)

	got, err := d.RolesByUser(context.Background(), seedSiteID(t, d), 1)
	if err != nil {
		t.Fatalf("RolesByUser(1) err = %v, want nil", err)
	}
	if len(got) != 0 {
		t.Errorf("len(RolesByUser(1)) = %d, want 0", len(got))
	}
}

func TestActiveAt(t *testing.T) {
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	cases := []struct {
		name string
		user User
		want bool
	}{
		{"flag only", User{IsActive: true}, true},
		{"flag only, off", User{IsActive: false}, false},
		{"deadline passed overrides a false flag", User{IsActive: false, InactiveUntil: &past}, true},
		{"deadline ahead overrides a true flag", User{IsActive: true, InactiveUntil: &future}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.user.ActiveAt(now); got != c.want {
				t.Errorf("ActiveAt() = %t, want %t", got, c.want)
			}
		})
	}
}

func TestRolesByUserSkipsAnotherSite(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()
	here := seedSiteID(t, d)

	roleList, err := d.AllRoles(ctx, here)
	if err != nil {
		t.Fatalf("AllRoles() err = %v, want nil", err)
	}
	if len(roleList) == 0 {
		t.Skip("the seed site has no roles to ask about")
	}

	elsewhere, err := d.AllRoles(ctx, here+1000)
	if err != nil {
		t.Fatalf("AllRoles(unknown site) err = %v, want nil", err)
	}
	if len(elsewhere) != 0 {
		t.Errorf("len(AllRoles(unknown site)) = %d, want 0", len(elsewhere))
	}

	bySlug, err := d.RoleIDsBySlug(ctx, here+1000, []string{"everyone", "registered"})
	if err != nil {
		t.Fatalf("RoleIDsBySlug(unknown site) err = %v, want nil", err)
	}
	if len(bySlug) != 0 {
		t.Errorf("len(RoleIDsBySlug(unknown site)) = %d, want 0", len(bySlug))
	}
}
