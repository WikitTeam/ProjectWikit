package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

var (
	qTagRule = register("TagRule", `
SELECT groups FROM pwikit_category_tag_rule WHERE category_id = $1`)

	qTagExemptRoles = register("TagExemptRoles", `
SELECT role_id FROM pwikit_category_tag_exempt WHERE category_id = $1 ORDER BY role_id`)

	qSaveTagRule = register("SaveTagRule", `
INSERT INTO pwikit_category_tag_rule (category_id, groups) VALUES ($1, $2)
ON CONFLICT (category_id) DO UPDATE SET groups = EXCLUDED.groups`)

	qDropTagExempt = register("DropTagExempt", `
DELETE FROM pwikit_category_tag_exempt WHERE category_id = $1`)

	qAddTagExempt = register("AddTagExempt", `
INSERT INTO pwikit_category_tag_exempt (category_id, role_id) VALUES ($1, $2)`)

	qRequiredTags = register("RequiredTags", `
SELECT r.groups,
       EXISTS (SELECT 1 FROM pwikit_category_tag_exempt e WHERE e.category_id = c.id AND e.role_id = ANY($3))
FROM pwikit_category_tag_rule r
JOIN web_category c ON c.id = r.category_id
WHERE c.site_id = $1 AND c.name = $2`)
)

// ParseTagGroups reads one group per line, the tags of a group split by
// spaces. A page needs at least one tag out of every group.
func ParseTagGroups(raw string) [][]string {
	var out [][]string
	for _, line := range strings.Split(raw, "\n") {
		var group []string
		for _, tag := range strings.Fields(line) {
			group = append(group, TagFullName(tag))
		}
		if len(group) > 0 {
			out = append(out, group)
		}
	}
	return out
}

func TagFullName(raw string) string {
	return tagFullName(splitTagName(strings.ToLower(strings.TrimSpace(raw))))
}

func (d *DB) TagRule(ctx context.Context, categoryID int64) (string, []int64, error) {
	var groups string
	err := d.pool.QueryRow(ctx, qTagRule, categoryID).Scan(&groups)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", nil, fmt.Errorf("read the tag rule of category %d: %w", categoryID, err)
	}
	rows, err := d.pool.Query(ctx, qTagExemptRoles, categoryID)
	if err != nil {
		return "", nil, fmt.Errorf("read the tag exemptions of category %d: %w", categoryID, err)
	}
	defer rows.Close()
	var roles []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return "", nil, fmt.Errorf("scan a tag exemption: %w", err)
		}
		roles = append(roles, id)
	}
	return groups, roles, rows.Err()
}

func (d *DB) SaveTagRule(ctx context.Context, categoryID int64, groups string, exempt []int64) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin the tag rule of category %d: %w", categoryID, err)
	}
	defer tx.Rollback(context.WithoutCancel(ctx))
	if _, err := tx.Exec(ctx, qSaveTagRule, categoryID, groups); err != nil {
		return fmt.Errorf("save the tag rule of category %d: %w", categoryID, err)
	}
	if _, err := tx.Exec(ctx, qDropTagExempt, categoryID); err != nil {
		return fmt.Errorf("clear the tag exemptions of category %d: %w", categoryID, err)
	}
	for _, id := range exempt {
		if _, err := tx.Exec(ctx, qAddTagExempt, categoryID, id); err != nil {
			return fmt.Errorf("exempt role %d in category %d: %w", id, categoryID, err)
		}
	}
	return tx.Commit(ctx)
}

// RequiredTags is nil when the category asks for nothing or one of the roles
// is exempt.
func (d *DB) RequiredTags(ctx context.Context, siteID int64, category string, roleIDs []int64) ([][]string, error) {
	var groups string
	var exempt bool
	err := d.pool.QueryRow(ctx, qRequiredTags, siteID, category, roleIDs).Scan(&groups, &exempt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read the required tags of %s: %w", category, err)
	}
	if exempt {
		return nil, nil
	}
	return ParseTagGroups(groups), nil
}

func MissingTagGroups(groups [][]string, tags []string) [][]string {
	held := map[string]bool{}
	for _, tag := range tags {
		held[TagFullName(tag)] = true
	}
	var out [][]string
	for _, group := range groups {
		found := false
		for _, tag := range group {
			if held[tag] {
				found = true
				break
			}
		}
		if !found {
			out = append(out, group)
		}
	}
	return out
}
