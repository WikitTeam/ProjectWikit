package db

import (
	"context"
	"testing"
	"time"
)

func TestRolesByUserWithoutRoles(t *testing.T) {
	d := newTestDB(t)

	got, err := d.RolesByUser(context.Background(), 1)
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
