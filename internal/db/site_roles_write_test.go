package db

import (
	"context"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/perms"
)

type seededRole struct {
	name        string
	index       int
	staff       bool
	profile     string
	permissions []string
}

func seededRoles(t *testing.T, d *DB, names map[string]string) map[string]seededRole {
	t.Helper()
	ctx := context.Background()
	stamp := strconv.FormatInt(time.Now().UnixNano(), 36)
	siteID, err := d.CreateSite(ctx, NewSite{
		Slug: "probe-" + stamp, Title: "Probe", Headline: "Probe",
		Domain: "probe-" + stamp + ".test", MediaDomain: "probe-" + stamp + ".test",
		RoleNames: names,
	})
	if err != nil {
		t.Fatalf("CreateSite() err = %v, want nil", err)
	}
	t.Cleanup(func() {
		for _, sql := range []string{
			`DELETE FROM web_role_permissions WHERE role_id IN (SELECT id FROM web_role WHERE site_id = $1)`,
			`DELETE FROM web_role WHERE site_id = $1`,
			`DELETE FROM web_settings WHERE site_id = $1`,
			`DELETE FROM web_site WHERE id = $1`,
		} {
			if _, err := d.pool.Exec(context.Background(), sql, siteID); err != nil {
				t.Errorf("clean up site %d err = %v, want nil", siteID, err)
			}
		}
	})

	rows, err := d.pool.Query(ctx, `
SELECT r.slug, r.name, r.index, r.is_staff, r.profile_visual_mode,
       coalesce(array_agg(p.codename ORDER BY p.codename) FILTER (WHERE p.codename IS NOT NULL), '{}')
FROM web_role r
LEFT JOIN web_role_permissions rp ON rp.role_id = r.id
LEFT JOIN auth_permission p ON p.id = rp.permission_id
WHERE r.site_id = $1
GROUP BY r.id`, siteID)
	if err != nil {
		t.Fatalf("read roles err = %v, want nil", err)
	}
	defer rows.Close()
	out := map[string]seededRole{}
	for rows.Next() {
		var slug string
		var role seededRole
		if err := rows.Scan(&slug, &role.name, &role.index, &role.staff, &role.profile, &role.permissions); err != nil {
			t.Fatalf("scan role err = %v, want nil", err)
		}
		out[slug] = role
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read roles err = %v, want nil", err)
	}
	return out
}

func sortedNames(names []string) []string {
	out := slices.Clone(names)
	slices.Sort(out)
	return out
}

func TestCreateSiteSeedsFourRoles(t *testing.T) {
	d := writeTestDB(t)
	got := seededRoles(t, d, nil)

	var slugs []string
	for slug := range got {
		slugs = append(slugs, slug)
	}
	slices.Sort(slugs)
	if want := []string{"admin", "everyone", "member", "registered"}; !slices.Equal(slugs, want) {
		t.Errorf("CreateSite() roles = %q, want %q", slugs, want)
	}
}

func TestCreateSiteNamesTheStarterRoles(t *testing.T) {
	d := writeTestDB(t)
	got := seededRoles(t, d, map[string]string{"admin": "Admins", "member": "Members"})

	for slug, want := range map[string]string{"admin": "Admins", "member": "Members", "registered": "", "everyone": ""} {
		if got[slug].name != want {
			t.Errorf("CreateSite() role %q name = %q, want %q", slug, got[slug].name, want)
		}
	}
}

func TestCreateSiteRanksAdminAboveMember(t *testing.T) {
	d := writeTestDB(t)
	got := seededRoles(t, d, nil)

	order := []string{"admin", "member", "registered", "everyone"}
	for i := 1; i < len(order); i++ {
		if got[order[i-1]].index >= got[order[i]].index {
			t.Errorf("CreateSite() index %s = %d, %s = %d, want %s lower",
				order[i-1], got[order[i-1]].index, order[i], got[order[i]].index, order[i-1])
		}
	}
}

func TestCreateSiteMakesOnlyAdminStaff(t *testing.T) {
	d := writeTestDB(t)
	got := seededRoles(t, d, nil)

	for slug, want := range map[string]bool{"admin": true, "member": false, "registered": false, "everyone": false} {
		if got[slug].staff != want {
			t.Errorf("CreateSite() role %q is_staff = %t, want %t", slug, got[slug].staff, want)
		}
	}
}

func TestCreateSiteShowsStarterRolesAsStatus(t *testing.T) {
	d := writeTestDB(t)
	got := seededRoles(t, d, nil)

	for slug, want := range map[string]string{"admin": "status", "member": "status", "registered": "hidden", "everyone": "hidden"} {
		if got[slug].profile != want {
			t.Errorf("CreateSite() role %q profile_visual_mode = %q, want %q", slug, got[slug].profile, want)
		}
	}
}

func TestCreateSiteGivesAdminEveryPermission(t *testing.T) {
	d := writeTestDB(t)
	got := seededRoles(t, d, nil)

	var want []string
	for _, g := range perms.Catalog {
		want = append(want, g.Names...)
	}
	if want := sortedNames(want); !slices.Equal(got["admin"].permissions, want) {
		t.Errorf("CreateSite() admin permissions = %q, want %q", got["admin"].permissions, want)
	}
}

func TestCreateSiteGivesMemberItsPermissions(t *testing.T) {
	d := writeTestDB(t)
	got := seededRoles(t, d, nil)

	want := sortedNames([]string{
		perms.ViewArticles, perms.RateArticles, perms.CreateArticles, perms.EditArticles,
		perms.TagArticles, perms.MoveArticles, perms.ManageArticleFiles, perms.DeleteArticles,
		perms.CommentArticles, perms.ViewArticleComments,
		perms.ViewForumPosts, perms.CreateForumPosts, perms.EditForumPosts,
		perms.ViewForumThreads, perms.ViewForumSections, perms.ViewHiddenForumSections,
		perms.ViewForumCategories,
		perms.SendDirectMessage,
	})
	if !slices.Equal(got["member"].permissions, want) {
		t.Errorf("CreateSite() member permissions = %q, want %q", got["member"].permissions, want)
	}
}
