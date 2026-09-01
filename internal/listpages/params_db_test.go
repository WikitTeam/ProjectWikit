package listpages

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/form"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

const (
	paramsGolden = "testdata/params.golden"
	paramsCorpus = "testdata/params_corpus.json"
)

type paramCase struct {
	Name   string            `json:"name"`
	Page   string            `json:"page"`
	Viewer string            `json:"viewer"`
	Params map[string]string `json:"params"`
	Path   map[string]string `json:"path"`
}

func paramCases() []paramCase {
	on := func(name string, params map[string]string) paramCase {
		return paramCase{Name: name, Page: "main", Params: params}
	}
	return []paramCase{
		{Name: "no-page", Params: map[string]string{"category": "*"}},
		{Name: "no-page-dot-category", Params: map[string]string{}},
		on("bare", nil),
		on("whole-site", map[string]string{"category": "*"}),
		on("this-page", map[string]string{"name": "."}),
		on("this-range", map[string]string{"range": "."}),
		on("full-name", map[string]string{"fullname": "nav:side"}),
		on("full-name-missing", map[string]string{"fullname": "no-such-page"}),
		on("hidden-pages", map[string]string{"pagetype": "hidden", "category": "*"}),
		on("bogus-pagetype", map[string]string{"pagetype": "sideways", "category": "*"}),
		on("name-prefix", map[string]string{"name": "NAV%", "category": "*"}),
		on("name-star-prefix", map[string]string{"name": "nav*", "category": "*"}),
		on("name-exact", map[string]string{"name": "Main", "category": "*"}),
		on("name-equals", map[string]string{"name": "=", "category": "*"}),
		on("name-star", map[string]string{"name": "*", "category": "*"}),
		on("category-list", map[string]string{"category": "forum, nav"}),
		on("category-negated", map[string]string{"category": "* -forum"}),
		on("category-with-colon", map[string]string{"category": "forum:thing"}),
		on("category-dot-in-list", map[string]string{"category": ". forum"}),
		on("tags-none", map[string]string{"tags": "-", "category": "*"}),
		on("tags-own", map[string]string{"tags": "=", "category": "*"}),
		on("tags-own-exact", map[string]string{"tags": "==", "category": "*"}),
		on("tags-unknown", map[string]string{"tags": "nosuchtag", "category": "*"}),
		on("tags-unknown-required", map[string]string{"tags": "+nosuchtag", "category": "*"}),
		on("tags-unknown-absent", map[string]string{"tags": "-nosuchtag", "category": "*"}),
		on("parent-none", map[string]string{"parent": "-", "category": "*"}),
		on("parent-own", map[string]string{"parent": "=", "category": "*"}),
		on("parent-not-own", map[string]string{"parent": "-=", "category": "*"}),
		on("parent-self", map[string]string{"parent": ".", "category": "*"}),
		on("parent-named", map[string]string{"parent": "NAV:side", "category": "*"}),
		on("parent-missing", map[string]string{"parent": "no-such-page", "category": "*"}),
		on("created-by-missing", map[string]string{"created_by": "nobody", "category": "*"}),
		on("created-by-anonymous", map[string]string{"created_by": ".", "category": "*"}),
		on("created-by-wikidot", map[string]string{"created_by": "wd:nobody", "category": "*"}),
		on("created-at-year", map[string]string{"created_at": "2021", "category": "*"}),
		on("created-at-month", map[string]string{"created_at": "2021-02", "category": "*"}),
		on("created-at-leap", map[string]string{"created_at": "2020-02", "category": "*"}),
		on("created-at-day", map[string]string{"created_at": "2021-02-09", "category": "*"}),
		on("created-at-clamped", map[string]string{"created_at": "2021-13-99", "category": "*"}),
		on("created-at-zeroes", map[string]string{"created_at": "2021-00-00", "category": "*"}),
		on("created-at-gt", map[string]string{"created_at": ">2021", "category": "*"}),
		on("created-at-gte", map[string]string{"created_at": ">=2021", "category": "*"}),
		on("created-at-lt", map[string]string{"created_at": "<2021", "category": "*"}),
		on("created-at-lte", map[string]string{"created_at": "<=2021", "category": "*"}),
		on("created-at-outside", map[string]string{"created_at": "<>2021", "category": "*"}),
		on("created-at-own-day", map[string]string{"created_at": "=", "category": "*"}),
		on("created-at-junk", map[string]string{"created_at": "twenty", "category": "*"}),
		on("created-at-trailing-dash", map[string]string{"created_at": "2021-", "category": "*"}),
		on("created-at-negative", map[string]string{"created_at": "-5", "category": "*"}),
		on("created-at-zero-year", map[string]string{"created_at": "0", "category": "*"}),
		on("created-at-too-far", map[string]string{"created_at": "10000", "category": "*"}),
		on("created-at-leading-space", map[string]string{"created_at": " >2021", "category": "*"}),
		on("legacy-date", map[string]string{"date": "2021", "category": "*"}),
		on("rating-int", map[string]string{"rating": "5", "category": "*"}),
		on("rating-float", map[string]string{"rating": "3.5", "category": "*"}),
		on("rating-negative", map[string]string{"rating": "-2", "category": "*"}),
		on("rating-gte", map[string]string{"rating": ">=5", "category": "*"}),
		on("rating-ne", map[string]string{"rating": "<>5", "category": "*"}),
		on("rating-junk", map[string]string{"rating": "high", "category": "*"}),
		on("rating-own", map[string]string{"rating": "=", "category": "*"}),
		on("votes-int", map[string]string{"votes": "2", "category": "*"}),
		on("votes-float", map[string]string{"votes": "2.5", "category": "*"}),
		on("votes-own", map[string]string{"votes": "=", "category": "*"}),
		on("popularity-gt", map[string]string{"popularity": ">50", "category": "*"}),
		on("popularity-own", map[string]string{"popularity": "=", "category": "*"}),
		on("order-name", map[string]string{"order": "name", "category": "*"}),
		on("order-name-desc", map[string]string{"order": "name desc", "category": "*"}),
		on("order-name-asc", map[string]string{"order": "name asc", "category": "*"}),
		on("order-three-words", map[string]string{"order": "name desc extra", "category": "*"}),
		on("order-unknown", map[string]string{"order": "nosuchcolumn", "category": "*"}),
		on("order-empty", map[string]string{"order": "", "category": "*"}),
		on("order-rating", map[string]string{"order": "rating", "category": "*"}),
		on("window", map[string]string{"offset": "5", "limit": "40", "perpage": "300", "category": "*"}),
		on("window-junk", map[string]string{"offset": "x", "limit": "y", "perpage": "z", "category": "*"}),
		{Name: "window-page", Page: "main", Params: map[string]string{"category": "*"}, Path: map[string]string{"p": "3"}},
		{Name: "window-page-zero", Page: "main", Params: map[string]string{"category": "*"}, Path: map[string]string{"p": "0"}},
		{Name: "window-page-junk", Page: "main", Params: map[string]string{"category": "*"}, Path: map[string]string{"p": "x"}},
	}
}

type dbSource struct {
	ctx    context.Context
	d      *db.DB
	siteID int64
}

func (s dbSource) LatestSource(articleID int64) (string, error) {
	return s.d.LatestSource(s.ctx, articleID)
}

func (s dbSource) CategoryForm(category string) (*form.Definition, error) {
	return nil, nil
}

func (s dbSource) TagIDsByName(categorySlug, name string) ([]int64, error) {
	return s.d.TagIDsByName(s.ctx, categorySlug, name)
}

func (s dbSource) ArticleTagIDs(articleID int64) ([]int64, error) {
	return s.d.ArticleTagIDs(s.ctx, articleID)
}

func (s dbSource) ArticleByRef(ref string) (*db.Article, error) {
	return s.d.ArticleByName(s.ctx, ref)
}

func (s dbSource) UserByUsername(name string) (*db.User, error) {
	return s.d.UserByUsername(s.ctx, name)
}

func (s dbSource) UserByWikidotName(name string) (*db.User, error) {
	return s.d.UserByWikidotName(s.ctx, name)
}

func (s dbSource) SiteRatingMode() (string, error) {
	return s.d.SiteRatingMode(s.ctx, s.siteID)
}

func (s dbSource) CategoryRatingMode(category string) (string, error) {
	return s.d.CategoryRatingMode(s.ctx, category)
}

func (s dbSource) VoteStats(articleID int64) (db.VoteStats, error) {
	return s.d.VoteStats(s.ctx, articleID)
}

func (s dbSource) HiddenCategories(*db.User) ([]string, error) { return nil, nil }

func (s dbSource) ListArticles(f db.ListFilter, offset int, limit *int) ([]db.Article, error) {
	return s.d.ListArticles(s.ctx, f, offset, limit)
}

func (s dbSource) CountArticles(f db.ListFilter, offset int, limit *int) (int, error) {
	return s.d.CountArticles(s.ctx, f, offset, limit)
}

func testSource(t *testing.T) dbSource {
	t.Helper()
	dsn := os.Getenv(db.EnvDSN)
	if dsn == "" {
		t.Skipf("%s not set, skipping the database test", db.EnvDSN)
	}
	ctx := context.Background()
	d, err := db.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("db.Open() err = %v, want nil", err)
	}
	t.Cleanup(d.Close)
	site, err := d.SiteByHosts(ctx, []string{"localhost"})
	if err != nil {
		t.Fatalf("SiteByHosts([localhost]) err = %v, want nil", err)
	}
	return dbSource{ctx: ctx, d: d, siteID: site.ID}
}

// TestParseMatchesGolden records what every parameter resolves to against the
// live database. The row ids in it are the database's own, which is what lets
// the oracle print the same thing without a fixture of its own.
func TestParseMatchesGolden(t *testing.T) {
	src := testSource(t)
	cases := paramCases()

	var b strings.Builder
	for _, c := range cases {
		var host *db.Article
		if c.Page != "" {
			found, err := src.ArticleByRef(c.Page)
			if err != nil {
				t.Fatalf("ArticleByRef(%q) err = %v, want nil", c.Page, err)
			}
			host = found
		}
		var viewer *db.User
		if c.Viewer != "" {
			found, err := src.UserByUsername(c.Viewer)
			if err != nil {
				t.Fatalf("UserByUsername(%q) err = %v, want nil", c.Viewer, err)
			}
			viewer = found
		}
		q, err := Parse(src, host, viewer, copyParams(c.Params), pathOf(c.Path))
		if err != nil {
			t.Fatalf("Parse(%s) err = %v, want nil", c.Name, err)
		}
		fmt.Fprintf(&b, "=== %s\n%s", c.Name, dumpQuery(q))
	}
	compareGolden(t, paramsGolden, b.String(), paramsCorpus, cases)
}

func copyParams(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func pathOf(in map[string]string) page.PathParams {
	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make(page.PathParams, 0, len(keys))
	for _, k := range keys {
		out = append(out, page.PathParam{Key: k, Value: in[k]})
	}
	return out
}

func dumpQuery(q Query) string {
	f := q.Filter
	var b strings.Builder
	line := func(key, value string) { fmt.Fprintf(&b, "%s=%s\n", key, value) }

	line("invalid", strconv.FormatBool(q.Invalid))
	line("only", optionalID(q.HasOnly, articleID(q.Only)))
	line("fullname", optional(q.HasFullName, q.FullName))
	line("pagetype", optional(f.PageType != "", f.PageType))
	line("name", optional(f.HasName, f.Name))
	line("nameprefix", optional(f.HasNamePrefix, f.NamePrefix))
	line("notags", strconv.FormatBool(f.NoTags))
	line("required", ids(f.RequiredTags))
	line("present", ids(f.PresentTags))
	line("absent", ids(f.AbsentTags))
	line("exact", ids(f.ExactTags))
	line("categories", strings.Join(f.Categories, ","))
	line("notcategories", strings.Join(f.NotCategories, ","))
	line("parent", pointerID(f.HasParent, f.ParentID))
	line("notparent", pointerID(f.HasNotParent, f.NotParentID))
	line("author", pointerID(f.AuthorID != nil, f.AuthorID))
	line("created_at", dumpTime(f.CreatedAt))
	line("rating", dumpNum(f.Rating))
	line("votes", dumpNum(f.Votes))
	line("popularity", dumpNum(f.Popularity))
	line("sort", optional(q.HasSort, f.Sort.Column+" "+direction(f.Sort.Ascending)))
	line("offset", strconv.Itoa(q.Offset))
	line("limit", optionalInt(q.Limit))
	line("page", strconv.Itoa(q.Page))
	line("perpage", strconv.Itoa(q.PerPage))
	return b.String()
}

func articleID(a *db.Article) int64 {
	if a == nil {
		return 0
	}
	return a.ID
}

func optional(present bool, value string) string {
	if !present {
		return "-"
	}
	return value
}

func optionalID(present bool, id int64) string {
	return optional(present, strconv.FormatInt(id, 10))
}

func optionalInt(v *int) string {
	if v == nil {
		return "-"
	}
	return strconv.Itoa(*v)
}

func pointerID(present bool, id *int64) string {
	if !present {
		return "-"
	}
	if id == nil {
		return "null"
	}
	return strconv.FormatInt(*id, 10)
}

func ids(v []int64) string {
	if len(v) == 0 {
		return ""
	}
	sorted := append([]int64(nil), v...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	parts := make([]string, len(sorted))
	for i, id := range sorted {
		parts[i] = strconv.FormatInt(id, 10)
	}
	return strings.Join(parts, ",")
}

func dumpTime(f *db.TimeFilter) string {
	if f == nil {
		return "-"
	}
	stamp := func(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05") }
	return f.Op + " " + stamp(f.Start) + " " + stamp(f.End)
}

func dumpNum(f *db.NumFilter) string {
	if f == nil {
		return "-"
	}
	return f.Op + " " + strconv.FormatFloat(f.Value, 'f', 6, 64)
}

func direction(ascending bool) string {
	if ascending {
		return "asc"
	}
	return "desc"
}
