package admin

import (
	"context"
	"net/http"
	"slices"
	"strconv"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
)

type grantRow struct {
	Name  string
	Allow bool
	Deny  bool
}

type grantGroup struct {
	Key  string
	Rows []grantRow
}

func grantGroups(catalog, allow, deny []string) []grantGroup {
	row := func(name string) grantRow {
		return grantRow{
			Name:  name,
			Allow: slices.Contains(allow, name),
			Deny:  slices.Contains(deny, name),
		}
	}
	known := make(map[string]bool, len(catalog))
	for _, name := range catalog {
		known[name] = true
	}

	groups := make([]grantGroup, 0, len(perms.Catalog)+1)
	placed := make(map[string]bool, len(catalog))
	for _, g := range perms.Catalog {
		group := grantGroup{Key: g.Key}
		for _, name := range g.Names {
			if !known[name] {
				continue
			}
			group.Rows = append(group.Rows, row(name))
			placed[name] = true
		}
		if len(group.Rows) > 0 {
			groups = append(groups, group)
		}
	}

	rest := grantGroup{}
	for _, name := range catalog {
		if !placed[name] {
			rest.Rows = append(rest.Rows, row(name))
		}
	}
	if len(rest.Rows) > 0 {
		groups = append(groups, rest)
	}
	return groups
}

// A category override is only read when the question is about a page or the
// comments under one, so the site-wide groups would change nothing there.
func pageScoped(catalog []string) []string {
	return slices.DeleteFunc(slices.Clone(catalog), func(name string) bool {
		group := perms.GroupOf(name)
		return group != "articles" && group != "forum"
	})
}

// A section override reaches the section and the threads and posts under it,
// and nothing else asks about a section.
func sectionScoped(catalog []string) []string {
	return slices.DeleteFunc(slices.Clone(catalog), func(name string) bool {
		return perms.GroupOf(name) != "forum"
	})
}

type overrideCell struct {
	Role    db.RoleChoice
	Grants  []grantGroup
	Present bool
}

func (h *Handler) overrideCells(ctx context.Context, scope func([]string) []string, overrides []db.CategoryOverride) ([]overrideCell, error) {
	roleList, err := h.deps.DB.AllRoles(ctx, siteID(ctx))
	if err != nil {
		return nil, err
	}
	catalog, err := h.deps.DB.PermissionCatalog(ctx)
	if err != nil {
		return nil, err
	}
	catalog = scope(catalog)
	byRole := map[int64]db.CategoryOverride{}
	for _, one := range overrides {
		byRole[one.RoleID] = one
	}
	cells := make([]overrideCell, 0, len(roleList))
	for _, one := range roleList {
		override := byRole[one.ID]
		cells = append(cells, overrideCell{
			Role:    one,
			Grants:  grantGroups(catalog, override.Allow, override.Deny),
			Present: len(override.Allow)+len(override.Deny) > 0,
		})
	}
	return cells, nil
}

func (h *Handler) postedOverrides(r *http.Request, scope func([]string) []string) ([]db.CategoryOverride, error) {
	ctx := r.Context()
	roleList, err := h.deps.DB.AllRoles(ctx, siteID(ctx))
	if err != nil {
		return nil, err
	}
	catalog, err := h.deps.DB.PermissionCatalog(ctx)
	if err != nil {
		return nil, err
	}
	catalog = scope(catalog)
	var out []db.CategoryOverride
	for _, one := range roleList {
		id := strconv.FormatInt(one.ID, 10)
		if r.PostFormValue("override_"+id) == "" {
			continue
		}
		next := db.CategoryOverride{RoleID: one.ID}
		for _, name := range catalog {
			switch r.PostFormValue("perm_" + id + "_" + name) {
			case "allow":
				next.Allow = append(next.Allow, name)
			case "deny":
				next.Deny = append(next.Deny, name)
			}
		}
		if len(next.Allow)+len(next.Deny) > 0 {
			out = append(out, next)
		}
	}
	return out, nil
}
