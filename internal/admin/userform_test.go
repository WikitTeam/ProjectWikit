package admin

import (
	"strings"
	"testing"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

func TestPickUnclaimed(t *testing.T) {
	waiting := []db.UserChoice{{ID: 3, Name: "Alice"}, {ID: 7, Name: "bob"}}
	cases := []struct {
		in   string
		id   int64
		want bool
	}{
		{"Alice", 3, true},
		{"alice", 3, true},
		{"bob", 7, true},
		{"#7", 7, true},
		{"7", 7, true},
		{"#9", 0, false},
		{"carol", 0, false},
		{"", 0, false},
	}
	for _, c := range cases {
		id, ok := pickUnclaimed(waiting, c.in)
		if id != c.id || ok != c.want {
			t.Errorf("pickUnclaimed(%q) = %d, %v, want %d, %v", c.in, id, ok, c.id, c.want)
		}
	}
}

func TestUntilUnlessDropsDeadlineWhenOn(t *testing.T) {
	if got := untilUnless(true, "2030-01-02T03:04", time.UTC); got != nil {
		t.Errorf("untilUnless(true, ...) = %v, want nil", got)
	}
}

func TestUntilUnlessKeepsDeadlineWhenOff(t *testing.T) {
	got := untilUnless(false, "2030-01-02T03:04", time.UTC)
	want := time.Date(2030, 1, 2, 3, 4, 0, 0, time.UTC)
	if got == nil || !got.Equal(want) {
		t.Errorf("untilUnless(false, ...) = %v, want %v", got, want)
	}
}

func TestUntilUnlessBlankIsOpenEnded(t *testing.T) {
	if got := untilUnless(false, "", time.UTC); got != nil {
		t.Errorf("untilUnless(false, \"\") = %v, want nil", got)
	}
}

func renderAdmin(t *testing.T, name string, data map[string]any) string {
	t.Helper()
	h, err := New(Deps{}, nil)
	if err != nil {
		t.Fatalf("New() err = %v, want nil", err)
	}
	tpl, err := h.bind(testLocalizer(t))
	if err != nil {
		t.Fatalf("bind() err = %v, want nil", err)
	}
	var out strings.Builder
	if err := tpl.ExecuteTemplate(&out, name, data); err != nil {
		t.Fatalf("ExecuteTemplate(%s) err = %v, want nil", name, err)
	}
	return out.String()
}

func TestUserFormShowsTheEffectiveState(t *testing.T) {
	later := testTime.Add(time.Hour)
	got := renderAdmin(t, "user_form.html", map[string]any{
		"User":       db.AdminUserRow{ID: 4, Username: "a", IsActive: true, InactiveUntil: &later, CanSendDM: false, DMUntil: &later},
		"Now":        testTime,
		"Zone":       time.UTC,
		"MayAccount": true,
		"Action":     "/-/admin/users/4",
		"CSRF":       "token",
		"ZoneName":   "UTC",
		"Error":      "",
		"Back":       "/-/admin/users/",
		"Activate":   "",
		"Activity":   "",
		"ResetVotes": "",
	})
	for _, want := range []string{`name="direct_messages_until"`, `data-value="2026-10-02T04:42"`} {
		if !strings.Contains(got, want) {
			t.Errorf("user_form.html = %q, want it to contain %q", got, want)
		}
	}
	if strings.Contains(got, ` value="2026-10-02T04:42"`) {
		t.Errorf("user_form.html = %q, want no deadline in a value attribute", got)
	}
	if strings.Contains(got, `name="is_active" value="1" checked`) {
		t.Errorf("user_form.html = %q, want is_active unchecked", got)
	}
}

func TestUserListMarksBannedUsers(t *testing.T) {
	got := renderAdmin(t, "user_list.html", map[string]any{
		"Users":  []db.AdminUserRow{{ID: 1, Username: "a", IsActive: true, Banned: true}},
		"Now":    testTime,
		"Banned": stateBanned,
		"Here":   stateSite,
		"State":  stateBanned,
		"Action": "/-/admin/users/",
		"Base":   "/-/admin/users/",
		"Query":  "",
		"Kind":   "",
		"Role":   int64(0),
		"Page":   1,
		"Pages":  1,
		"Total":  1,
	})
	for _, want := range []string{`class="state bad"`, `?state=banned`, `name="state" value="banned"`} {
		if !strings.Contains(got, want) {
			t.Errorf("user_list.html = %q, want it to contain %q", got, want)
		}
	}
}

func TestClaimFormOffersASearchableList(t *testing.T) {
	got := renderAdmin(t, "user_claim.html", map[string]any{
		"Waiting": []db.UserChoice{{ID: 3, Name: "Alice"}},
		"Picked":  "Ali",
		"Action":  "/-/admin/users/claim-link",
		"CSRF":    "token",
		"Error":   "",
		"Link":    "",
		"Back":    "/-/admin/invites/",
	})
	for _, want := range []string{`list="unclaimed-users"`, `<option value="Alice">`, `value="Ali"`} {
		if !strings.Contains(got, want) {
			t.Errorf("user_claim.html = %q, want it to contain %q", got, want)
		}
	}
}

func TestDashboardLinksUsers(t *testing.T) {
	got := renderAdmin(t, "dashboard.html", map[string]any{
		"Site":        &db.Site{},
		"Settings":    db.SiteSettings{},
		"Changes":     []changeRow{{Title: "x", Href: "/x", User: "Bob", UserHref: profileHref("Bob B"), CreatedAt: testTime}},
		"ChangesHref": recentChangesPage,
	})
	for _, want := range []string{`href="/-/users/Bob%20B"`, `href="/system:recent-changes"`} {
		if !strings.Contains(got, want) {
			t.Errorf("dashboard.html = %q, want it to contain %q", got, want)
		}
	}
}
