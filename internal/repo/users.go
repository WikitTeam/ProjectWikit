package repo

import (
	"context"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/pageconfig"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/roles"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

func UsersJSON(ctx context.Context, d *db.DB, users []db.User, now time.Time) (wikijson.Array, error) {
	ids := make([]int64, 0, len(users))
	for i := range users {
		ids = append(ids, users[i].ID)
	}
	byUser, err := d.RolesByUsers(ctx, ids)
	if err != nil {
		return nil, err
	}
	bySlug, err := d.RoleIDsBySlug(ctx, []string{slugEveryone, slugRegistered})
	if err != nil {
		return nil, err
	}

	wanted := map[int64]bool{}
	for _, id := range bySlug {
		wanted[id] = true
	}
	for _, list := range byUser {
		for _, role := range list {
			wanted[role.ID] = true
		}
	}
	roleIDs := make([]int64, 0, len(wanted))
	for id := range wanted {
		roleIDs = append(roleIDs, id)
	}
	loaded, err := d.RolePermissions(ctx, roleIDs)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]perms.Role, len(loaded))
	for _, role := range loaded {
		byID[role.ID] = role
	}

	base := make([]perms.Role, 0, 2)
	for _, slug := range []string{slugEveryone, slugRegistered} {
		if id, ok := bySlug[slug]; ok {
			base = append(base, byID[id])
		}
	}

	out := make(wikijson.Array, 0, len(users))
	for i := range users {
		u := &users[i]
		own := byUser[u.ID]
		subject := perms.Subject{
			Roles:       append(append([]perms.Role{}, base...), rolesOf(byID, own)...),
			Active:      u.ActiveAt(now),
			ForumActive: u.ForumActiveAt(now),
			Superuser:   u.IsSuperuser,
		}
		editor := perms.Resolve(subject, nil).Has(perms.EditArticles)
		out = append(out, pageconfig.SignedInUserJSON(u, own, true, editor).Object())
	}
	return out, nil
}

func rolesOf(byID map[int64]perms.Role, own []roles.Role) []perms.Role {
	out := make([]perms.Role, 0, len(own))
	for _, role := range own {
		out = append(out, byID[role.ID])
	}
	return out
}
