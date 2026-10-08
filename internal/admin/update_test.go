package admin

import (
	"strings"
	"testing"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

func TestLayoutShowsUpdateNotices(t *testing.T) {
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
		Title:    "Dashboard",
		AdminURL: Prefix,
		loc:      loc,
		Updates: []updateNotice{
			{Text: loc.T("update.notice-scheduled", "version", "v1.1.0", "time", localTime(testTime)), Actions: true, CSRF: "token", Notes: "https://example.org/notes"},
			{Error: true, Text: loc.T("update.notice-rolled-back", "version", "v1.1.0", "from", "v1.0.0", "time", localTime(testTime)), Detail: "health <check> failed"},
		},
	}
	var out strings.Builder
	if err := tpl.ExecuteTemplate(&out, "layout.html", view); err != nil {
		t.Fatalf("ExecuteTemplate(layout.html) err = %v, want nil", err)
	}
	got := out.String()
	for _, want := range []string{
		`action="/-/admin/update/postpone"`,
		`action="/-/admin/update/skip"`,
		`value="token"`,
		`href="https://example.org/notes"`,
		`class="alert alert-error"`,
		`health &lt;check&gt; failed`,
		`data-local-time`,
		"v1.1.0",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("layout.html = %q, want it to contain %q", got, want)
		}
	}
}

var testTime = time.Date(2026, 10, 2, 3, 42, 0, 0, time.UTC)

func TestDashboardOffersAnUpdateCheck(t *testing.T) {
	h, err := New(Deps{}, nil)
	if err != nil {
		t.Fatalf("New() err = %v, want nil", err)
	}
	loc := testLocalizer(t)
	tpl, err := h.bind(loc)
	if err != nil {
		t.Fatalf("bind() err = %v, want nil", err)
	}
	data := map[string]any{
		"Site":        &db.Site{},
		"Settings":    db.SiteSettings{},
		"Version":     "v0.9.0",
		"CheckHref":   Prefix + updateSlug + "/check",
		"ChangesHref": "/system:recent-changes",
		"CSRF":        "token",
		"Checked":     &checkResult{Error: true, Text: loc.T("update.check-failed"), Detail: "dial <tcp>"},
	}
	var out strings.Builder
	if err := tpl.ExecuteTemplate(&out, "dashboard.html", data); err != nil {
		t.Fatalf("ExecuteTemplate(dashboard.html) err = %v, want nil", err)
	}
	got := out.String()
	for _, want := range []string{
		`action="/-/admin/update/check"`,
		`value="token"`,
		"v0.9.0",
		loc.T("update.check-now"),
		`class="alert alert-error"`,
		`dial &lt;tcp&gt;`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("dashboard.html = %q, want it to contain %q", got, want)
		}
	}
}
