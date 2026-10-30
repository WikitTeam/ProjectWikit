package admin

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
)

func TestParseTarget(t *testing.T) {
	cases := []struct {
		raw  string
		kind string
		id   int64
		name string
	}{
		{"", "", 0, ""},
		{"scp-173", "page", 0, "scp-173"},
		{"/SCP 173", "page", 0, "scp-173"},
		{"https://wiki.example.org/fragment:abc/noredirect/true", "page", 0, "fragment:abc"},
		{"t-12", "thread", 12, ""},
		{"https://wiki.example.org/forum/t-12/some-name", "thread", 12, ""},
		{"https://wiki.example.org/forum/t-12/some-name#post-99", "post", 99, ""},
		{"post-99", "post", 99, ""},
		{"#post-99", "post", 99, ""},
		{"/forum/s-3/general", "section", 3, ""},
	}
	for _, c := range cases {
		kind, id, name := parseTarget(c.raw)
		if kind != c.kind || id != c.id || name != c.name {
			t.Errorf("parseTarget(%q) = %q, %d, %q, want %q, %d, %q", c.raw, kind, id, name, c.kind, c.id, c.name)
		}
	}
}

func testExplainer(loc *i18n.Localizer) *explainer {
	member := db.RoleChoice{ID: 2, Slug: "member", Name: "<b>Member</b>"}
	return &explainer{
		loc:     loc,
		site:    &db.Site{},
		roles:   map[int64]db.RoleChoice{1: {ID: 1, Slug: "everyone"}, 2: member},
		choices: []db.RoleChoice{{ID: 1, Slug: "everyone"}, member},
		cats:    map[string]int64{"fragment": 7},
	}
}

func TestExplainerEscapesRoleNames(t *testing.T) {
	x := testExplainer(testLocalizer(t))
	got := x.cause(perms.Cause{Reason: perms.ReasonRoleDeny, RoleID: 2})
	if strings.Contains(got, "<b>") {
		t.Errorf("cause(role-deny) = %q, want the role name escaped", got)
	}
}

func TestExplainerVerdict(t *testing.T) {
	x := testExplainer(testLocalizer(t))
	until := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	x.user = &db.User{ID: 5, IsActive: true}
	x.sanctions = map[perms.Sanction]*time.Time{perms.SanctionMute: &until}
	subject := perms.Subject{Active: true, ForumActive: true, Roles: []perms.Role{{ID: 1, Permissions: []string{perms.CreateForumPosts}}}, Sanctions: []perms.Sanction{perms.SanctionMute}}
	v := x.verdict(perms.Explain(subject, nil, perms.CreateForumPosts), true)
	if v.Allowed {
		t.Fatal("verdict(muted).Allowed = true, want false")
	}
	if !strings.Contains(v.Summary, "2026-11-03") {
		t.Errorf("verdict(muted).Summary = %q, want the end time", v.Summary)
	}
	if v.Fix.Href != Prefix+userSlug+"/5" {
		t.Errorf("verdict(muted).Fix.Href = %q, want %q", v.Fix.Href, Prefix+userSlug+"/5")
	}
	if len(v.Steps) != len(perms.Steps) {
		t.Errorf("len(verdict(muted).Steps) = %d, want %d", len(v.Steps), len(perms.Steps))
	}
}

func TestAccessTemplateRenders(t *testing.T) {
	for _, lang := range []string{i18n.DefaultLanguage, "en"} {
		t.Run(lang, func(t *testing.T) {
			h, err := New(Deps{}, nil)
			if err != nil {
				t.Fatal(err)
			}
			loc := testBundle(t).Localizer(lang)
			tpl, err := h.bind(loc)
			if err != nil {
				t.Fatal(err)
			}
			x := testExplainer(loc)
			x.user = &db.User{ID: 5, IsActive: true}
			x.target = &accessTarget{kind: "page", category: "fragment", label: "fragment:a", link: "/fragment:a", article: &db.Article{Locked: true}}
			object := &perms.Object{Locked: true, Overrides: []perms.Override{{RoleID: 2, Restrictions: []string{perms.EditArticles}}}}
			subject := perms.Subject{Active: true, Roles: []perms.Role{{ID: 1, Permissions: []string{perms.ViewArticles, perms.EditArticles}}, {ID: 2}}}
			base := func(tab string) map[string]any {
				return map[string]any{"Base": Prefix + accessSlug + "/", "Tab": tab, "Perms": permOptions(loc), "Roles": x.choices, "Form": accessForm{User: "someone"}}
			}
			tabs := map[string]map[string]any{}

			single := base("")
			single["Who"] = []fact{{Label: "a", Value: "b", Tone: "deny"}}
			single["What"] = x.whatFacts(object)
			single["Verdict"] = x.verdict(perms.Explain(subject, object, perms.EditArticles), true)
			tabs["single"] = single

			table := base("")
			table["Who"] = single["Who"]
			table["What"] = single["What"]
			table["Table"] = x.table(subject, object, url.Values{"user": {"someone"}})
			tabs["table"] = table

			role := base(accessRoleTab)
			role["RoleTable"] = []roleGroup{{Label: "g", Rows: []roleRow{{Name: "edit_articles", Label: "Edit", Base: "Allow", Override: "—", Result: "Allow", Tone: "allow"}}}}
			role["RoleName"], role["RoleHref"], role["Everyone"], role["Categories"] = "member", "/x", true, []string{"fragment"}
			tabs["role"] = role

			holders := base(accessHoldersTab)
			holders["Checked"] = true
			holders["Anonymous"] = x.verdict(perms.Explain(perms.Subject{Anonymous: true}, nil, perms.EditArticles), false)
			holders["Member"] = x.verdict(perms.Explain(subject, nil, perms.EditArticles), false)
			holders["Holders"] = []holderRow{{Name: "a", Href: "/a", Check: "/c", Allowed: true, Summary: "s"}}
			holders["Refused"] = []holderRow{}
			holders["Truncated"] = accessLimit
			tabs["holders"] = holders

			lint := base(accessLintTab)
			lint["Findings"] = []finding{{Tone: "deny", Text: "t", Href: "/h", Link: "l"}, {Tone: "warn", Text: "t"}, {Tone: "info", Text: "t"}}
			tabs["lint"] = lint
			tabs["lint-clean"] = base(accessLintTab)

			for name, data := range tabs {
				var out strings.Builder
				if err := tpl.ExecuteTemplate(&out, "access.html", data); err != nil {
					t.Errorf("ExecuteTemplate(access.html, %s) err = %v, want nil", name, err)
					continue
				}
				if strings.Contains(out.String(), "admin.") {
					t.Errorf("ExecuteTemplate(access.html, %s) left an untranslated key in %q", name, out.String())
				}
			}
		})
	}
}
