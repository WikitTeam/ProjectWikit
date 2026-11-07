package modules

import (
	"slices"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

const (
	memberRole = 1
	staffRole  = 2
)

type sectionData struct {
	module.Data
	staff  bool
	asked  db.ForumSearchFilter
	search bool
}

func (d *sectionData) Subject(*db.User) (perms.Subject, error) {
	role := perms.Role{ID: memberRole}
	if d.staff {
		role.ID = staffRole
	}
	role.Permissions = []string{perms.ViewForumSections, perms.ViewForumCategories, perms.ViewForumThreads, perms.ViewForumPosts}
	return perms.Subject{Active: true, Roles: []perms.Role{role}}, nil
}

func (d *sectionData) ForumSectionObject(s *db.ForumSection) *perms.Object {
	return &perms.Object{Kind: perms.KindForumSection, Overrides: s.Overrides}
}

var probeSections = []db.ForumSection{
	{ID: 1, Name: "Open", IsIndexed: true},
	{ID: 2, Name: "Hidden", IsHidden: true, IsIndexed: true},
	{ID: 3, Name: "Staff", IsIndexed: true, Overrides: []perms.Override{
		{RoleID: memberRole, Restrictions: []string{perms.ViewForumSections, perms.ViewForumCategories, perms.ViewForumThreads}},
	}},
	{ID: 4, Name: "Unindexed"},
}

func (d *sectionData) ForumSections() ([]db.ForumSection, error) { return probeSections, nil }

func (d *sectionData) ForumSection(id int64) (*db.ForumSection, error) {
	for i := range probeSections {
		if probeSections[i].ID == id {
			return &probeSections[i], nil
		}
	}
	return nil, db.ErrNotFound
}

func (d *sectionData) ForumCategories() ([]db.ForumCategory, error) {
	return []db.ForumCategory{
		{ID: 11, Name: "open-chat", SectionID: 1},
		{ID: 12, Name: "hidden-chat", SectionID: 2},
		{ID: 13, Name: "staff-chat", SectionID: 3},
		{ID: 14, Name: "unindexed-chat", SectionID: 4},
	}, nil
}

func (d *sectionData) ForumCategory(id int64) (*db.ForumCategory, error) {
	all, _ := d.ForumCategories()
	for i := range all {
		if all[i].ID == id {
			return &all[i], nil
		}
	}
	return nil, db.ErrNotFound
}

func (d *sectionData) SearchForum(f db.ForumSearchFilter, _, _ int) ([]db.ForumSearchHit, error) {
	d.asked, d.search = f, true
	return nil, nil
}

func sectionEnv(t *testing.T, staff bool) (module.Env, *sectionData) {
	t.Helper()
	env := forumEnv(t)
	data := &sectionData{staff: staff}
	env.Data = data
	env.Page = &page.Context{}
	return env, data
}

func TestForumSearchAPISections(t *testing.T) {
	cases := map[bool][]int64{false: {1, 2}, true: {1, 2, 3}}
	for staff, want := range cases {
		env, data := sectionEnv(t, staff)
		if _, err := forumSearchAPI(env, map[string]string{"q": "x"}); err != nil {
			t.Fatalf("forumSearchAPI(staff=%t) err = %v, want nil", staff, err)
		}
		if !data.search {
			t.Fatalf("forumSearchAPI(staff=%t) searched = false, want true", staff)
		}
		if !slices.Equal(data.asked.Sections, want) {
			t.Errorf("forumSearchAPI(staff=%t) Sections = %v, want %v", staff, data.asked.Sections, want)
		}
	}
}

func placeNames(t *testing.T, staff bool) map[string]bool {
	t.Helper()
	env, _ := sectionEnv(t, staff)
	out, err := forumPlacesAPI(env, nil)
	if err != nil {
		t.Fatalf("forumPlacesAPI(staff=%t) err = %v, want nil", staff, err)
	}
	names := map[string]bool{}
	for _, field := range out {
		if field.Key != "places" {
			continue
		}
		for _, place := range field.Value.(wikijson.Array) {
			for _, f := range place.(wikijson.Object) {
				if f.Key == "name" {
					names[f.Value.(string)] = true
				}
			}
		}
	}
	return names
}

func TestForumPlacesAPIMember(t *testing.T) {
	names := placeNames(t, false)
	for name, want := range map[string]bool{"open-chat": true, "hidden-chat": true, "staff-chat": false, "unindexed-chat": false} {
		if names[name] != want {
			t.Errorf("forumPlacesAPI(staff=false) has %q = %t, want %t", name, names[name], want)
		}
	}
}

func TestForumPlacesAPIStaff(t *testing.T) {
	names := placeNames(t, true)
	for name, want := range map[string]bool{"open-chat": true, "hidden-chat": true, "staff-chat": true, "unindexed-chat": false} {
		if names[name] != want {
			t.Errorf("forumPlacesAPI(staff=true) has %q = %t, want %t", name, names[name], want)
		}
	}
}

func TestForumThreadCategoryNoIndex(t *testing.T) {
	cases := map[int64]bool{11: false, 12: false, 13: false, 14: true}
	for category, want := range cases {
		env, data := sectionEnv(t, true)
		subject, _ := data.Subject(nil)
		id := category
		view := forumThreadView{thread: db.ForumThread{CategoryID: &id}}
		if err := forumThreadCategory(env, &view, subject, perms.Resolve(subject, nil)); err != nil {
			t.Fatalf("forumThreadCategory(category=%d) err = %v, want nil", category, err)
		}
		if env.Page.NoIndex != want {
			t.Errorf("forumThreadCategory(category=%d) NoIndex = %t, want %t", category, env.Page.NoIndex, want)
		}
	}
}

func TestCategoryAllows(t *testing.T) {
	env, data := sectionEnv(t, false)
	subject, _ := data.Subject(nil)
	grants, err := sectionGrants(env, subject)
	if err != nil {
		t.Fatalf("sectionGrants() err = %v, want nil", err)
	}
	sitewide := perms.Resolve(subject, nil)
	categories, _ := data.ForumCategories()
	for _, c := range categories {
		want := c.SectionID != 3
		if got := categoryAllows(grants, sitewide, &c, perms.ViewForumThreads); got != want {
			t.Errorf("categoryAllows(%q) = %t, want %t", c.Name, got, want)
		}
	}
}
