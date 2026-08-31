package db

import (
	"context"
	"strings"
	"testing"
)

func siteChangeFilterVariants() map[string]SiteChangeFilter {
	return map[string]SiteChangeFilter{
		"empty":      {},
		"hidden":     {Hidden: []string{"admin"}},
		"one-type":   {Types: []string{"source"}},
		"many-types": {Types: []string{"source", "title", "revert"}},
		"category":   {Category: "probe", HasCategory: true},
		"users":      {HasUser: true, UserIDs: int64s(1, 2)},
		"users-none": {HasUser: true},
		"system":     {HasUser: true, UserIDs: int64s(1), WithSystem: true},
		"everything": {Hidden: []string{"admin"}, Types: []string{"tags"}, Category: "probe", HasCategory: true, HasUser: true, UserIDs: int64s(1), WithSystem: true},
	}
}

func TestSiteChangeFilterSQLMatchesSchema(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()

	for name, filter := range siteChangeFilterVariants() {
		if _, err := d.SiteChanges(ctx, filter, 5, 20); err != nil {
			t.Errorf("SiteChanges(%s) err = %v, want nil", name, err)
		}
		if _, err := d.SiteChangeCount(ctx, filter); err != nil {
			t.Errorf("SiteChangeCount(%s) err = %v, want nil", name, err)
		}
	}
}

func TestSiteChangeSelectSQLNamesEveryArgument(t *testing.T) {
	for name, filter := range siteChangeFilterVariants() {
		sql, args := filter.SelectSQL(3, 10)
		for i := range args {
			if !strings.Contains(sql, placeholder(i+1)) {
				t.Errorf("SelectSQL(%s) leaves %s unused", name, placeholder(i+1))
			}
		}
	}
}

func TestLikeContainsEscapesWildcards(t *testing.T) {
	cases := map[string]string{
		"probe": "%probe%",
		"a_b":   `%a\_b%`,
		"a%b":   `%a\%b%`,
		`a\b`:   `%a\\b%`,
		"":      "%%",
	}
	for in, want := range cases {
		if got := likeContains(in); got != want {
			t.Errorf("likeContains(%q) = %q, want %q", in, got, want)
		}
	}
}
