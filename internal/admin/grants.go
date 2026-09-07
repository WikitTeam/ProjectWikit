package admin

import (
	"slices"

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
