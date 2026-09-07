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
	"web_forumcategory",
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

	byKey := map[string]bool{}
	for _, name := range siteScopeByKey {
		byKey[name] = true
	}

	var missing, stale, both []string
	for _, q := range queries {
		if pending[q.name] && byKey[q.name] {
			both = append(both, q.name)
		}
		settled := len(touchedTables(q.sql)) == 0 ||
			strings.Contains(q.sql, "site_id") ||
			byKey[q.name]
		if settled {
			if pending[q.name] {
				stale = append(stale, q.name)
			}
			continue
		}
		if !pending[q.name] {
			missing = append(missing, q.name)
		}
	}

	for _, name := range both {
		t.Errorf("query %q is in both siteScopeByKey and siteScopePending, want one", name)
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

func TestSiteScopeListsHoldOnlyKnownNames(t *testing.T) {
	known := make([]string, 0, len(queries))
	for _, q := range queries {
		known = append(known, q.name)
	}
	for list, names := range map[string][]string{
		"siteScopePending": siteScopePending,
		"siteScopeByKey":   siteScopeByKey,
	} {
		for _, name := range names {
			if !slices.Contains(known, name) {
				t.Errorf("%s has %q, want a registered query name", list, name)
			}
		}
	}
}

func TestBuiltFiltersNameTheSite(t *testing.T) {
	list, _ := ListFilter{SiteID: 7}.SelectSQL(0, nil)
	if !strings.Contains(list, "a.site_id") {
		t.Errorf("ListFilter.SelectSQL() = %q, want it to name a.site_id", list)
	}
	changes, _ := SiteChangeFilter{SiteID: 7}.SelectSQL(0, 10)
	if !strings.Contains(changes, "a.site_id") {
		t.Errorf("SiteChangeFilter.SelectSQL() = %q, want it to name a.site_id", changes)
	}
	b := &listBuilder{}
	search := SearchFilter{SiteID: 7}.where(b)
	if !strings.Contains(search, "a.site_id") {
		t.Errorf("SearchFilter.where() = %q, want it to name a.site_id", search)
	}
}
