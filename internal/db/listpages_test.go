package db

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"
)

func int64s(v ...int64) []int64 { return v }

// listFilterVariants covers every branch that puts SQL together, since a
// generated statement escapes the check the registered ones go through.
func listFilterVariants() map[string]ListFilter {
	parent := int64(3)
	return map[string]ListFilter{
		"empty":          {},
		"hidden":         {Hidden: []string{"admin"}},
		"normal-pages":   {PageType: PageTypeNormal},
		"hidden-pages":   {PageType: PageTypeHidden},
		"name":           {Name: "main", HasName: true},
		"name-prefix":    {NamePrefix: "scp-", HasNamePrefix: true},
		"no-tags":        {NoTags: true},
		"exact-tags":     {ExactTags: int64s(1, 2)},
		"required-tags":  {RequiredTags: int64s(1, 2)},
		"present-tags":   {PresentTags: int64s(3)},
		"absent-tags":    {AbsentTags: int64s(4)},
		"categories":     {Categories: []string{"scp"}, NotCategories: []string{"meta"}},
		"no-parent":      {HasParent: true},
		"parent":         {HasParent: true, ParentID: &parent},
		"not-parent":     {HasNotParent: true, NotParentID: &parent},
		"any-parent":     {HasNotParent: true},
		"author":         {AuthorID: &parent},
		"created-range":  {CreatedAt: &TimeFilter{Op: TimeRange, Start: time.Unix(0, 0), End: time.Unix(1, 0)}},
		"created-out":    {CreatedAt: &TimeFilter{Op: TimeExcludeRange, Start: time.Unix(0, 0), End: time.Unix(1, 0)}},
		"created-lt":     {CreatedAt: &TimeFilter{Op: TimeLT, Start: time.Unix(0, 0)}},
		"created-lte":    {CreatedAt: &TimeFilter{Op: TimeLTE, Start: time.Unix(0, 0)}},
		"created-gt":     {CreatedAt: &TimeFilter{Op: TimeGT, End: time.Unix(0, 0)}},
		"created-gte":    {CreatedAt: &TimeFilter{Op: TimeGTE, End: time.Unix(0, 0)}},
		"updated-range":  {UpdatedAt: &TimeFilter{Op: TimeRange, Start: time.Unix(0, 0), End: time.Unix(1, 0)}},
		"updated-lt":     {UpdatedAt: &TimeFilter{Op: TimeLT, Start: time.Unix(0, 0)}},
		"link-to":        {LinkTo: "theme:black", HasLinkTo: true},
		"link-to-bare":   {LinkTo: "start", HasLinkTo: true},
		"rating-updown":  {RatingMode: "updown", Rating: &NumFilter{Op: NumGTE, Value: 3}},
		"rating-stars":   {RatingMode: "stars", Rating: &NumFilter{Op: NumLT, Value: 3}},
		"rating-off":     {RatingMode: "disabled", Rating: &NumFilter{Op: NumNE, Value: 0}},
		"votes":          {Votes: &NumFilter{Op: NumEQ, Value: 2}},
		"popularity":     {RatingMode: "updown", Popularity: &NumFilter{Op: NumGT, Value: 50}},
		"sort-created":   {Sort: Sort{Column: SortCreatedAt}},
		"sort-author":    {Sort: Sort{Column: SortCreatedBy, Ascending: true}},
		"sort-name":      {Sort: Sort{Column: SortName, Ascending: true}},
		"sort-title":     {Sort: Sort{Column: SortTitle}},
		"sort-updated":   {Sort: Sort{Column: SortUpdatedAt}},
		"sort-fullname":  {Sort: Sort{Column: SortFullName}},
		"sort-rating":    {RatingMode: "stars", Sort: Sort{Column: SortRating}},
		"sort-votes":     {Sort: Sort{Column: SortVotes}},
		"sort-pop":       {RatingMode: "updown", Sort: Sort{Column: SortPopularity}},
		"sort-random":    {Sort: Sort{Column: SortRandom}},
		"sort-size":      {Sort: Sort{Column: SortSize}},
		"sort-revisions": {Sort: Sort{Column: SortRevisions}},
		"sort-comments":  {Sort: Sort{Column: SortComments}},
	}
}

// TestListFilterSQLMatchesSchema sends every shape the builder can produce to
// Postgres. The registered statements get this from TestQueriesMatchSchema, and
// without it the built ones would only fail on a page load.
func TestListFilterSQLMatchesSchema(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()
	limit := 20

	for name, filter := range listFilterVariants() {
		for _, window := range []struct {
			label  string
			offset int
			limit  *int
		}{{"plain", 0, nil}, {"windowed", 5, &limit}} {
			if _, err := d.ListArticles(ctx, filter, window.offset, window.limit); err != nil {
				t.Errorf("ListArticles(%s, %s) err = %v, want nil", name, window.label, err)
			}
			if _, err := d.CountArticles(ctx, filter, window.offset, window.limit); err != nil {
				t.Errorf("CountArticles(%s, %s) err = %v, want nil", name, window.label, err)
			}
		}
	}
}

func TestLikePrefixEscapesWildcards(t *testing.T) {
	cases := map[string]string{
		"scp-": "scp-%",
		"a_b":  `a\_b%`,
		"a%b":  `a\%b%`,
		`a\b`:  `a\\b%`,
		"":     "%",
	}
	for in, want := range cases {
		if got := likePrefix(in); got != want {
			t.Errorf("likePrefix(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSelectSQLNamesEveryArgument(t *testing.T) {
	for name, filter := range listFilterVariants() {
		limit := 10
		sql, args := filter.SelectSQL(3, &limit)
		for i := range args {
			if !strings.Contains(sql, placeholder(i+1)) {
				t.Errorf("SelectSQL(%s) leaves %s unused", name, placeholder(i+1))
			}
		}
	}
}

func placeholder(n int) string {
	return "$" + strconv.Itoa(n)
}
