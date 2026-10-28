package admin

import (
	"context"
	"slices"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
)

func TestSeesSensitive(t *testing.T) {
	for _, tc := range []struct {
		name string
		user *db.User
		want bool
	}{
		{"anonymous", nil, false},
		{"staff", &db.User{ID: 2, IsActive: true}, false},
		{"instance superuser", &db.User{ID: 1, IsActive: true, IsSuperuser: true}, true},
		{"inactive superuser", &db.User{ID: 1, IsSuperuser: true}, false},
	} {
		ctx := context.Background()
		if tc.user != nil {
			ctx = auth.NewContext(ctx, tc.user)
		}
		if got := seesSensitive(ctx); got != tc.want {
			t.Errorf("seesSensitive(%s) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestCatalogLeavesOutRetired(t *testing.T) {
	for _, g := range perms.Catalog {
		for _, name := range perms.Retired {
			if slices.Contains(g.Names, name) {
				t.Errorf("perms.Catalog[%q] has %q, want it left out", g.Key, name)
			}
		}
	}
}
