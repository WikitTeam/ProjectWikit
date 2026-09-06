package admin

import (
	"slices"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
)

func testLocalizer(t *testing.T) *i18n.Localizer {
	t.Helper()
	bundle, err := i18n.Load("")
	if err != nil {
		t.Fatalf("Load() err = %v, want nil", err)
	}
	return bundle.Localizer(i18n.DefaultLanguage)
}

func TestRegisteredScreensAreReachable(t *testing.T) {
	slugs := make([]string, 0, len(screens))
	for _, s := range screens {
		slugs = append(slugs, s.slug)
	}
	if !slices.Contains(slugs, themeSlug) {
		t.Errorf("screens = %v, want it to contain %q", slugs, themeSlug)
	}
	for _, s := range screens {
		if s.serve == nil {
			t.Errorf("screen %q has no handler, want one", s.slug)
		}
		if s.label == "" {
			t.Errorf("screen %q has no label, want one", s.slug)
		}
	}
}

func TestTemplatesParse(t *testing.T) {
	h, err := New(Deps{}, nil)
	if err != nil {
		t.Fatalf("New() err = %v, want nil", err)
	}
	for _, name := range []string{"index.html", "theme_list.html", "theme_form.html"} {
		if h.templates.Lookup(name) == nil {
			t.Errorf("Lookup(%q) = nil, want the template", name)
		}
	}
}

func TestCheckTheme(t *testing.T) {
	loc := testLocalizer(t)
	h := &Handler{}
	cases := map[string]struct {
		row  db.ThemeRow
		want bool
	}{
		"inline is fine":         {db.ThemeRow{Name: "a", Slug: "a", Mode: db.ThemeInline}, true},
		"external with a url":    {db.ThemeRow{Name: "a", Slug: "a", Mode: db.ThemeExternal, ExternalURL: "https://x/y.css"}, true},
		"no name":                {db.ThemeRow{Slug: "a", Mode: db.ThemeInline}, false},
		"empty slug":             {db.ThemeRow{Name: "a", Mode: db.ThemeInline}, false},
		"slug with a space":      {db.ThemeRow{Name: "a", Slug: "a b", Mode: db.ThemeInline}, false},
		"unknown mode":           {db.ThemeRow{Name: "a", Slug: "a", Mode: "other"}, false},
		"external without a url": {db.ThemeRow{Name: "a", Slug: "a", Mode: db.ThemeExternal}, false},
	}
	for name, c := range cases {
		got := h.checkTheme(loc, c.row) == ""
		if got != c.want {
			t.Errorf("checkTheme(%s) accepted = %t, want %t", name, got, c.want)
		}
	}
}

func TestCheckSite(t *testing.T) {
	loc := testLocalizer(t)
	ok := db.Site{Slug: "wikit", Title: "T", Domain: "a.test", MediaDomain: "b.test",
		HomePage: "main", EmailPolicy: db.EmailOptional}
	fine := db.SiteSettings{RatingMode: "default", CreateTags: "default"}

	if got := checkSite(loc, ok, fine); got != "" {
		t.Errorf("checkSite(complete) = %q, want \"\"", got)
	}

	bad := map[string]struct {
		site     db.Site
		settings db.SiteSettings
	}{
		"slug with a space": {withSlug(ok, "a b"), fine},
		"empty slug":        {withSlug(ok, ""), fine},
		"no title":          {func() db.Site { s := ok; s.Title = ""; return s }(), fine},
		"no domain":         {func() db.Site { s := ok; s.Domain = ""; return s }(), fine},
		"no media domain":   {func() db.Site { s := ok; s.MediaDomain = ""; return s }(), fine},
		"no home page":      {func() db.Site { s := ok; s.HomePage = ""; return s }(), fine},
		"unknown policy":    {func() db.Site { s := ok; s.EmailPolicy = "later"; return s }(), fine},
		"unknown rating":    {ok, db.SiteSettings{RatingMode: "vibes", CreateTags: "default"}},
		"unknown tag mode":  {ok, db.SiteSettings{RatingMode: "default", CreateTags: "vibes"}},
	}
	for name, c := range bad {
		if checkSite(loc, c.site, c.settings) == "" {
			t.Errorf("checkSite(%s) = \"\", want a complaint", name)
		}
	}
}

func withSlug(s db.Site, slug string) db.Site {
	s.Slug = slug
	return s
}

func TestOptionalID(t *testing.T) {
	for raw, want := range map[string]int64{"": 0, "0": 0, " 7 ": 7, "12": 12, "x": 0} {
		got := optionalID(raw)
		if want == 0 {
			if got != nil {
				t.Errorf("optionalID(%q) = %d, want nil", raw, *got)
			}
			continue
		}
		if got == nil || *got != want {
			t.Errorf("optionalID(%q) = %v, want %d", raw, got, want)
		}
	}
}

func TestCheckRole(t *testing.T) {
	loc := testLocalizer(t)
	ok := db.RoleRow{Slug: "helper", InlineVisualMode: "badge", ProfileVisualMode: "status"}
	if got := checkRole(loc, ok); got != "" {
		t.Errorf("checkRole(complete) = %q, want \"\"", got)
	}
	bad := map[string]db.RoleRow{
		"empty slug":        {InlineVisualMode: "badge", ProfileVisualMode: "status"},
		"slug with a space": {Slug: "a b", InlineVisualMode: "badge", ProfileVisualMode: "status"},
		"unknown inline":    {Slug: "a", InlineVisualMode: "glow", ProfileVisualMode: "status"},
		"unknown profile":   {Slug: "a", InlineVisualMode: "badge", ProfileVisualMode: "glow"},
		"profile as inline": {Slug: "a", InlineVisualMode: "status", ProfileVisualMode: "status"},
		"inline as profile": {Slug: "a", InlineVisualMode: "badge", ProfileVisualMode: "icon"},
	}
	for name, row := range bad {
		if checkRole(loc, row) == "" {
			t.Errorf("checkRole(%s) = \"\", want a complaint", name)
		}
	}
}

func TestBuiltinRolesAreNamed(t *testing.T) {
	for _, want := range []string{"everyone", "registered"} {
		if !slices.Contains(builtinRoles, want) {
			t.Errorf("builtinRoles = %v, want it to contain %q", builtinRoles, want)
		}
	}
}

func TestEveryScreenNeedsAPermission(t *testing.T) {
	for _, s := range screens {
		if s.need == "" {
			t.Errorf("screen %q needs no permission, want one", s.slug)
		}
	}
}

func TestMaySetRoles(t *testing.T) {
	super := &db.User{ID: 1, IsSuperuser: true}
	plain := &db.User{ID: 2}
	none := perms.Resolve(perms.Subject{Active: true}, nil)
	granting := perms.Resolve(perms.Subject{
		Active: true,
		Roles:  []perms.Role{{ID: 1, Permissions: []string{perms.ManagePermissions}}},
	}, nil)

	cases := map[string]struct {
		mine    *db.User
		granted perms.Set
		target  db.AdminUserRow
		want    bool
	}{
		"anonymous":                     {nil, granting, db.AdminUserRow{}, false},
		"superuser over anyone":         {super, none, db.AdminUserRow{IsSuperuser: true}, true},
		"granted over a plain target":   {plain, granting, db.AdminUserRow{}, true},
		"granted over a superuser":      {plain, granting, db.AdminUserRow{IsSuperuser: true}, false},
		"ungranted over a plain target": {plain, none, db.AdminUserRow{}, false},
	}
	h := &Handler{}
	for name, c := range cases {
		if got := h.maySetRoles(c.mine, c.granted, c.target); got != c.want {
			t.Errorf("maySetRoles(%s) = %t, want %t", name, got, c.want)
		}
	}
}

func TestOptionalTime(t *testing.T) {
	if optionalTime("") != nil || optionalTime("  ") != nil || optionalTime("nonsense") != nil {
		t.Error("optionalTime over an unparseable value = non-nil, want nil")
	}
	got := optionalTime("2026-09-06T12:30")
	if got == nil {
		t.Fatal("optionalTime(\"2026-09-06T12:30\") = nil, want a time")
	}
	if got.Year() != 2026 || got.Month() != 9 || got.Day() != 6 || got.Hour() != 12 {
		t.Errorf("optionalTime(\"2026-09-06T12:30\") = %v, want 2026-09-06 12:30", got)
	}
}

func TestAllSixteenScreensAreRegistered(t *testing.T) {
	want := []string{
		"themes", "site", "roles", "role-categories", "users", "reports",
		"forum-sections", "forum-categories", "action-log",
		"tags", "tag-categories", "page-categories",
		"tickets", "membership-applications", "invite-links",
	}
	got := make([]string, 0, len(screens))
	for _, s := range screens {
		got = append(got, s.slug)
	}
	for _, slug := range want {
		if !slices.Contains(got, slug) {
			t.Errorf("screens = %v, want it to contain %q", got, slug)
		}
	}
	if len(screens) != len(want) {
		t.Errorf("len(screens) = %d, want %d", len(screens), len(want))
	}
}

func TestEveryScreenHasItsTemplates(t *testing.T) {
	h, err := New(Deps{}, nil)
	if err != nil {
		t.Fatalf("New() err = %v, want nil", err)
	}
	for _, name := range []string{
		"index.html", "theme_list.html", "theme_form.html", "site_form.html",
		"role_list.html", "role_form.html", "role_category.html",
		"user_list.html", "user_form.html", "report_list.html", "report_form.html",
		"forum_section_list.html", "forum_section_form.html",
		"forum_category_list.html", "forum_category_form.html", "action_log.html",
		"tag_list.html", "tag_form.html", "tag_category_list.html", "tag_category_form.html",
		"page_category_list.html", "page_category_form.html",
		"ticket_list.html", "ticket_form.html", "invite_list.html",
	} {
		if h.templates.Lookup(name) == nil {
			t.Errorf("Lookup(%q) = nil, want the template", name)
		}
	}
}

func TestOptionalInt(t *testing.T) {
	if optionalInt("") != nil || optionalInt("  ") != nil || optionalInt("x") != nil {
		t.Error("optionalInt over an empty or unparseable value = non-nil, want nil")
	}
	got := optionalInt(" 7 ")
	if got == nil || *got != 7 {
		t.Errorf("optionalInt(\" 7 \") = %v, want 7", got)
	}
}
