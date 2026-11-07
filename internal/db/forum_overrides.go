package db

import (
	"context"
	"fmt"

	"github.com/WikitTeam/ProjectWikit/internal/perms"
)

var qForumSectionOverrides = register("ForumSectionOverrides", `
SELECT spo.forumsection_id, o.id, o.role_id, op.codename, op.restricted
FROM web_forumsection s
JOIN web_forumsection_permissions_override spo ON spo.forumsection_id = s.id
JOIN web_rolepermissionsoverride o ON o.id = spo.rolepermissionsoverride_id
LEFT JOIN (
    SELECT x.rolepermissionsoverride_id AS override_id, p.codename, false AS restricted
    FROM web_rolepermissionsoverride_permissions x
    JOIN auth_permission p ON p.id = x.permission_id
    UNION ALL
    SELECT x.rolepermissionsoverride_id, p.codename, true
    FROM web_rolepermissionsoverride_restrictions x
    JOIN auth_permission p ON p.id = x.permission_id
) op ON op.override_id = o.id
WHERE s.site_id = $1 AND s.id = ANY($2)
ORDER BY spo.forumsection_id, o.id`)

func (d *DB) ForumSectionOverrides(ctx context.Context, siteID int64, sectionIDs []int64) (map[int64][]perms.Override, error) {
	out := map[int64][]perms.Override{}
	if len(sectionIDs) == 0 {
		return out, nil
	}
	rows, err := d.pool.Query(ctx, qForumSectionOverrides, siteID, sectionIDs)
	if err != nil {
		return nil, fmt.Errorf("list permission overrides of forum sections %v: %w", sectionIDs, err)
	}
	defer rows.Close()

	var current int64
	for rows.Next() {
		var section, id, roleID int64
		var codename *string
		var restricted *bool
		if err := rows.Scan(&section, &id, &roleID, &codename, &restricted); err != nil {
			return nil, fmt.Errorf("scan permission override of forum section: %w", err)
		}
		list := out[section]
		if len(list) == 0 || current != id {
			list = append(list, perms.Override{RoleID: roleID})
			current = id
		}
		if codename != nil {
			last := &list[len(list)-1]
			if *restricted {
				last.Restrictions = append(last.Restrictions, *codename)
			} else {
				last.Permissions = append(last.Permissions, *codename)
			}
		}
		out[section] = list
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list permission overrides of forum sections %v: %w", sectionIDs, err)
	}
	return out, nil
}

func (d *DB) withSectionOverrides(ctx context.Context, siteID int64, sections []ForumSection) error {
	ids := make([]int64, len(sections))
	for i := range sections {
		ids[i] = sections[i].ID
	}
	found, err := d.ForumSectionOverrides(ctx, siteID, ids)
	if err != nil {
		return err
	}
	for i := range sections {
		sections[i].Overrides = found[sections[i].ID]
	}
	return nil
}

var qAdminForumSectionOverrides = register("AdminForumSectionOverrides", `
SELECT o.role_id, coalesce(nullif(r.name, ''), r.slug), p.codename, x.restricted
FROM web_forumsection_permissions_override spo
JOIN web_forumsection s ON s.id = spo.forumsection_id
JOIN web_rolepermissionsoverride o ON o.id = spo.rolepermissionsoverride_id
JOIN web_role r ON r.id = o.role_id
LEFT JOIN (
    SELECT rolepermissionsoverride_id AS override_id, permission_id, false AS restricted
    FROM web_rolepermissionsoverride_permissions
    UNION ALL
    SELECT rolepermissionsoverride_id, permission_id, true
    FROM web_rolepermissionsoverride_restrictions
) x ON x.override_id = o.id
LEFT JOIN auth_permission p ON p.id = x.permission_id
WHERE spo.forumsection_id = $1 AND s.site_id = $2
ORDER BY r.index, o.role_id`)

func (d *DB) adminSectionOverrides(ctx context.Context, siteID, sectionID int64) ([]CategoryOverride, error) {
	rows, err := d.pool.Query(ctx, qAdminForumSectionOverrides, sectionID, siteID)
	if err != nil {
		return nil, fmt.Errorf("read the overrides of forum section %d: %w", sectionID, err)
	}
	defer rows.Close()

	var out []CategoryOverride
	at := map[int64]int{}
	for rows.Next() {
		var roleID int64
		var roleName string
		var codename *string
		var restricted *bool
		if err := rows.Scan(&roleID, &roleName, &codename, &restricted); err != nil {
			return nil, err
		}
		i, ok := at[roleID]
		if !ok {
			i = len(out)
			at[roleID] = i
			out = append(out, CategoryOverride{RoleID: roleID, RoleName: roleName})
		}
		if codename == nil || restricted == nil {
			continue
		}
		if *restricted {
			out[i].Deny = append(out[i].Deny, *codename)
		} else {
			out[i].Allow = append(out[i].Allow, *codename)
		}
	}
	return out, rows.Err()
}

var (
	qSectionOverrideIDs = register("SectionOverrideIDs", `
SELECT rolepermissionsoverride_id FROM web_forumsection_permissions_override WHERE forumsection_id = $1`)
	qDropSectionLinks    = register("DropSectionLinks", `DELETE FROM web_forumsection_permissions_override WHERE forumsection_id = $1`)
	qLinkSectionOverride = register("LinkSectionOverride", `
INSERT INTO web_forumsection_permissions_override (forumsection_id, rolepermissionsoverride_id) VALUES ($1, $2)`)
)

func (d *DB) SaveForumSectionOverrides(ctx context.Context, sectionID int64, list []CategoryOverride) error {
	return d.saveOverrides(ctx, overrideLinks{ids: qSectionOverrideIDs, drop: qDropSectionLinks, link: qLinkSectionOverride},
		"forum section", sectionID, list)
}
