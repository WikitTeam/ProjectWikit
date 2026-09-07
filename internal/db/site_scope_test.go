package db

import (
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
)

var perSiteTables = []string{
	"web_article",
	"web_category",
	"web_tag",
	"web_tagscategory",
	"web_role",
	"web_rolecategory",
	"web_forumsection",
	"web_forumthread",
	"web_theme",
	"web_invitelink",
	"web_userreport",
	"web_userticket",
	"pwikit_admin_log",
}

func touchedTables(sql string) []string {
	var out []string
	for _, table := range perSiteTables {
		pattern := regexp.MustCompile(`\b` + table + `\b`)
		if pattern.MatchString(sql) {
			out = append(out, table)
		}
	}
	return out
}

func TestEveryPerSiteQueryNamesTheSite(t *testing.T) {
	pending := map[string]bool{}
	for _, name := range siteScopePending {
		pending[name] = true
	}

	var missing, stale []string
	for _, q := range queries {
		if len(touchedTables(q.sql)) == 0 {
			if pending[q.name] {
				stale = append(stale, q.name)
			}
			continue
		}
		if strings.Contains(q.sql, "site_id") {
			if pending[q.name] {
				stale = append(stale, q.name)
			}
			continue
		}
		if !pending[q.name] {
			missing = append(missing, q.name)
		}
	}

	for _, name := range missing {
		t.Errorf("query %q reads a per-site table without site_id, want the filter or an entry in siteScopePending", name)
	}
	for _, name := range stale {
		t.Errorf("query %q is in siteScopePending but no longer needs to be, want it removed", name)
	}

	if os.Getenv("PWIKIT_SITE_SCOPE_LIST") != "" {
		var left []string
		for _, q := range queries {
			if pending[q.name] {
				left = append(left, q.name)
			}
		}
		sort.Strings(left)
		t.Logf("%d queries still to scope:\n%s", len(left), strings.Join(left, "\n"))
	}
}

func TestSiteScopePendingHasNoUnknownNames(t *testing.T) {
	known := make([]string, 0, len(queries))
	for _, q := range queries {
		known = append(known, q.name)
	}
	for _, name := range siteScopePending {
		if !slices.Contains(known, name) {
			t.Errorf("siteScopePending has %q, want a registered query name", name)
		}
	}
}
