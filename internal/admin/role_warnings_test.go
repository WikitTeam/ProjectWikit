package admin

import (
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

func TestRoleTemplatesShowWarnings(t *testing.T) {
	h, err := New(Deps{}, nil)
	if err != nil {
		t.Fatalf("New() err = %v, want nil", err)
	}
	loc := testLocalizer(t)
	tpl, err := h.bind(loc)
	if err != nil {
		t.Fatalf("bind() err = %v, want nil", err)
	}
	warnings := map[int64][]finding{1: {{RoleID: 1, Tone: "deny", Text: "denied <b>x</b>", Href: "/c", Link: "open"}}}

	var list strings.Builder
	err = tpl.ExecuteTemplate(&list, "role_list.html", map[string]any{
		"Roles":     []db.RoleRow{{ID: 1, Slug: "everyone", Users: 7}, {ID: 2, Slug: "member", Users: 5}},
		"Managing":  map[int64]bool{},
		"Warnings":  warnings,
		"Builtin":   map[int64]bool{1: true},
		"New":       "/new",
		"Base":      "/",
		"UsersHref": "/u",
	})
	if err != nil {
		t.Fatalf("ExecuteTemplate(role_list.html) err = %v, want nil", err)
	}
	if got := strings.Count(list.String(), "role-flag"); got != 1 {
		t.Errorf("role_list.html role-flag count = %d, want 1", got)
	}
	if strings.Contains(list.String(), ">7<") {
		t.Errorf("role_list.html = %q, want no member count for everyone", list.String())
	}
	if !strings.Contains(list.String(), ">5</a>") {
		t.Errorf("role_list.html = %q, want the member count for member", list.String())
	}

	var form strings.Builder
	err = tpl.ExecuteTemplate(&form, "role_form.html", map[string]any{
		"Role":     db.RoleRow{ID: 1, Slug: "everyone"},
		"Builtin":  true,
		"Warnings": warnings[1],
		"Action":   "/a",
		"Back":     "/b",
		"CSRF":     "token",
		"Error":    "",
		"Access":   "",
	})
	if err != nil {
		t.Fatalf("ExecuteTemplate(role_form.html) err = %v, want nil", err)
	}
	for _, want := range []string{"role-warnings", loc.T("admin.role-warnings"), "denied <b>x</b>", `href="/c"`} {
		if !strings.Contains(form.String(), want) {
			t.Errorf("role_form.html = %q, want it to contain %q", form.String(), want)
		}
	}
}

func TestAssignableRoles(t *testing.T) {
	got := assignableRoles([]db.RoleChoice{{ID: 1, Slug: "everyone"}, {ID: 2, Slug: "registered"}, {ID: 3, Slug: "member"}})
	if len(got) != 1 || got[0].ID != 3 {
		t.Errorf("assignableRoles() = %v, want only member", got)
	}
}

func TestOffered(t *testing.T) {
	choices := []db.RoleChoice{{ID: 3, Slug: "member"}}
	one, two := int64(1), int64(3)
	if got := offered(&one, choices); got != nil {
		t.Errorf("offered(1) = %d, want nil", *got)
	}
	if got := offered(&two, choices); got == nil || *got != 3 {
		t.Errorf("offered(3) = %v, want 3", got)
	}
	if got := offered(nil, choices); got != nil {
		t.Errorf("offered(nil) = %d, want nil", *got)
	}
}

func TestLayoutListsScreensForSearch(t *testing.T) {
	h, err := New(Deps{}, nil)
	if err != nil {
		t.Fatalf("New() err = %v, want nil", err)
	}
	loc := testLocalizer(t)
	tpl, err := h.bind(loc)
	if err != nil {
		t.Fatalf("bind() err = %v, want nil", err)
	}
	view := layoutView{
		AdminURL:  Prefix,
		loc:       loc,
		Dashboard: navItem{Href: Prefix, Label: "Dashboard", Icon: "fa-tachometer-alt"},
		Groups:    []navGroup{{Key: "people", Label: "People & <Roles>", Items: []navItem{{Href: Prefix + "roles/", Label: "Roles", Icon: "fa-shield-alt"}}}},
	}
	var out strings.Builder
	if err := tpl.ExecuteTemplate(&out, "layout.html", view); err != nil {
		t.Fatalf("ExecuteTemplate(layout.html) err = %v, want nil", err)
	}
	for _, want := range []string{
		`data-search-text="Dashboard"`,
		`data-search-text="Roles People &amp; &lt;Roles&gt;"`,
		loc.T("admin.search-admin"),
		loc.T("admin.search-none"),
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("layout.html = %q, want it to contain %q", out.String(), want)
		}
	}
}
