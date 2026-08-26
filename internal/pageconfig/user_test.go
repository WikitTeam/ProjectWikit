package pageconfig

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/roles"
)

var update = flag.Bool("update", false, "rewrite the golden file and the corpus the oracle reads")

const (
	goldenPath = "testdata/pageconfig.golden"
	corpusPath = "testdata/login_corpus.json"
)

type roleSpec struct {
	Slug        string `json:"slug"`
	Index       int    `json:"index"`
	IsStaff     bool   `json:"is_staff"`
	GroupVotes  bool   `json:"group_votes"`
	InlineMode  string `json:"inline_visual_mode"`
	ProfileMode string `json:"profile_visual_mode"`
	CanEdit     bool   `json:"can_edit_articles"`
}

type userSpec struct {
	Username        string `json:"username"`
	Type            string `json:"type"`
	DisplayName     string `json:"display_name"`
	WikidotUsername string `json:"wikidot_username"`
	Avatar          string `json:"avatar"`
	IsActive        bool   `json:"is_active"`
	IsSuperuser     bool   `json:"is_superuser"`
}

type caseSpec struct {
	Name              string     `json:"name"`
	Kind              string     `json:"kind"`
	User              *userSpec  `json:"user,omitempty"`
	Roles             []roleSpec `json:"roles,omitempty"`
	Editor            bool       `json:"editor"`
	NotificationCount int        `json:"notification_count"`
}

func corpus() []caseSpec {
	normal := func() *userSpec {
		return &userSpec{Username: "loginprobe", Type: db.UserTypeNormal, IsActive: true}
	}
	badge := roleSpec{Slug: "probebadged", Index: 1, InlineMode: "badge", ProfileMode: "hidden"}
	hidden := roleSpec{Slug: "probeinvisible", Index: 2, InlineMode: "hidden", ProfileMode: "hidden"}
	staff := roleSpec{Slug: "probemoderator", Index: 3, IsStaff: true, InlineMode: "hidden", ProfileMode: "hidden"}
	grouped := roleSpec{Slug: "probegrouped", Index: 4, GroupVotes: true, InlineMode: "hidden", ProfileMode: "hidden"}
	profile := roleSpec{Slug: "probeprofiled", Index: 5, InlineMode: "hidden", ProfileMode: "status"}

	return []caseSpec{
		{Name: "anonymous", Kind: "anonymous"},
		{Name: "system", Kind: "system"},
		{Name: "plain", Kind: "user", User: normal()},
		{
			Name: "with display name and avatar",
			Kind: "user",
			User: &userSpec{
				Username: "loginprobe", Type: db.UserTypeNormal, IsActive: true,
				DisplayName: "Login Probe", Avatar: "-/users/loginprobe.png",
			},
		},
		{
			Name: "wikidot with display name",
			Kind: "user",
			User: &userSpec{
				Username: "loginprobe", Type: db.UserTypeWikidot, IsActive: true,
				DisplayName: "Wd Display", WikidotUsername: "Wd Original",
			},
		},
		{
			Name: "wikidot without display name",
			Kind: "user",
			User: &userSpec{
				Username: "loginprobe", Type: db.UserTypeWikidot, IsActive: true,
				WikidotUsername: "Wd Original",
			},
		},
		{
			Name: "inactive",
			Kind: "user",
			User: &userSpec{Username: "loginprobe", Type: db.UserTypeNormal, IsActive: false},
		},
		{
			Name:   "superuser",
			Kind:   "user",
			User:   &userSpec{Username: "loginprobe", Type: db.UserTypeNormal, IsActive: true, IsSuperuser: true},
			Editor: true,
		},
		{Name: "staff by role", Kind: "user", User: normal(), Roles: []roleSpec{staff}},
		{Name: "visual roles", Kind: "user", User: normal(), Roles: []roleSpec{badge, hidden, grouped, profile}},
		{Name: "only hidden roles", Kind: "user", User: normal(), Roles: []roleSpec{hidden}},
		{
			Name:   "editor",
			Kind:   "user",
			User:   normal(),
			Roles:  []roleSpec{{Slug: "probeeditor", Index: 6, InlineMode: "hidden", ProfileMode: "hidden", CanEdit: true}},
			Editor: true,
		},
		{Name: "with notifications", Kind: "user", User: normal(), NotificationCount: 7},
		{
			Name: "unicode display name",
			Kind: "user",
			User: &userSpec{
				Username: "loginprobe", Type: db.UserTypeNormal, IsActive: true, DisplayName: "中文 名字",
			},
		},
	}
}

func localizer(t *testing.T) *i18n.Localizer {
	t.Helper()
	b, err := i18n.Load("")
	if err != nil {
		t.Fatalf("i18n.Load() err = %v, want nil", err)
	}
	return b.Localizer(i18n.DefaultLanguage)
}

func toRoles(specs []roleSpec) []roles.Role {
	out := make([]roles.Role, len(specs))
	for i, spec := range specs {
		out[i] = roles.Role{
			Slug:              spec.Slug,
			Index:             spec.Index,
			IsStaff:           spec.IsStaff,
			GroupVotes:        spec.GroupVotes,
			InlineVisualMode:  roles.InlineVisualMode(spec.InlineMode),
			ProfileVisualMode: roles.ProfileVisualMode(spec.ProfileMode),
		}
	}
	return out
}

func toUser(spec *userSpec) *db.User {
	return &db.User{
		ID:              424242,
		Type:            spec.Type,
		Username:        spec.Username,
		WikidotUsername: spec.WikidotUsername,
		DisplayName:     spec.DisplayName,
		Avatar:          spec.Avatar,
		IsActive:        spec.IsActive,
		IsSuperuser:     spec.IsSuperuser,
	}
}

func render(t *testing.T, loc *i18n.Localizer, c caseSpec) string {
	t.Helper()
	if c.Kind == "system" {
		got, err := SystemUserJSON().JSON()
		if err != nil {
			t.Fatalf("SystemUserJSON().JSON() err = %v, want nil", err)
		}
		return got
	}
	status := LoginStatus{Roles: toRoles(c.Roles), CanEditArticles: c.Editor, NotificationCount: c.NotificationCount}
	if c.Kind == "user" {
		status.User = toUser(c.User)
	}
	got, err := status.JSON(loc)
	if err != nil {
		t.Fatalf("LoginStatus(%s).JSON() err = %v, want nil", c.Name, err)
	}
	return got
}

func TestLoginStatusMatchesGolden(t *testing.T) {
	cases := corpus()
	loc := localizer(t)

	var b strings.Builder
	for _, c := range cases {
		fmt.Fprintf(&b, "=== %s\n%s\n", c.Name, render(t, loc, c))
	}
	got := b.String()

	if *update {
		writeCorpus(t, cases)
		if err := os.WriteFile(filepath.FromSlash(goldenPath), []byte(got), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) err = %v, want nil", goldenPath, err)
		}
		return
	}

	want, err := os.ReadFile(filepath.FromSlash(goldenPath))
	if err != nil {
		t.Fatalf("ReadFile(%s) err = %v, want nil", goldenPath, err)
	}
	if got != string(want) {
		t.Errorf("LoginStatus(corpus) = %q, want %q", got, string(want))
	}
}

func writeCorpus(t *testing.T, cases []caseSpec) {
	t.Helper()
	data, err := json.MarshalIndent(cases, "", "  ")
	if err != nil {
		t.Fatalf("Marshal(corpus) err = %v, want nil", err)
	}
	if err := os.WriteFile(filepath.FromSlash(corpusPath), append(data, '\n'), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) err = %v, want nil", corpusPath, err)
	}
}

func TestIsStaffOfSuperuserWithoutRoles(t *testing.T) {
	if !IsStaff(&db.User{IsSuperuser: true}, nil) {
		t.Error("IsStaff(superuser, nil) = false, want true")
	}
}

func TestIsStaffOfPlainUser(t *testing.T) {
	if IsStaff(&db.User{}, []roles.Role{{Slug: "reader"}}) {
		t.Error("IsStaff(user, reader) = true, want false")
	}
}
