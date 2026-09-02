package listpages

import (
	"testing"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/form"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

func (f *fakeSource) LatestSource(articleID int64) (string, error) {
	return f.sources[articleID], nil
}

func (f *fakeSource) CategoryForm(category string) (*form.Definition, error) {
	return f.forms[category], nil
}

type fakeSource struct {
	sources map[int64]string
	forms   map[string]*form.Definition

	tags         map[string][]int64
	articleTags  map[int64][]int64
	articles     map[string]*db.Article
	users        map[string]*db.User
	wikidotUsers map[string]*db.User
	siteMode     string
	categoryMode map[string]string
	votes        db.VoteStats
	hidden       []string

	listed []db.Article
	total  int

	gotFilter db.ListFilter
	gotOffset int
	gotLimit  *int
	listCalls int
}

func newFakeSource() *fakeSource {
	return &fakeSource{
		tags: map[string][]int64{
			":euclid":       {1},
			"_default:.":    nil,
			"_default:hub":  {2},
			"meta:hub":      {2},
			":hub":          {2, 3},
			"scp:hub":       {3},
			":safe":         {4},
			"_default:safe": {4},
		},
		articleTags: map[int64][]int64{7: {1, 4}},
		articles: map[string]*db.Article{
			"scp:scp-173": {ID: 7, Category: "scp", Name: "scp-173", Title: "173"},
			"main":        {ID: 3, Category: db.DefaultCategory, Name: "main", Title: "Main"},
		},
		users:        map[string]*db.User{"alice": {ID: 11, Username: "alice"}},
		wikidotUsers: map[string]*db.User{"Bob": {ID: 12, Username: "bob-1", WikidotUsername: "Bob"}},
		categoryMode: map[string]string{},
	}
}

func (f *fakeSource) TagIDsByName(categorySlug, name string) ([]int64, error) {
	return f.tags[categorySlug+":"+name], nil
}

func (f *fakeSource) ArticleTagIDs(articleID int64) ([]int64, error) {
	return f.articleTags[articleID], nil
}

func (f *fakeSource) ArticleByRef(ref string) (*db.Article, error) {
	if a, ok := f.articles[ref]; ok {
		return a, nil
	}
	return nil, db.ErrNotFound
}

func (f *fakeSource) UserByUsername(name string) (*db.User, error) {
	if u, ok := f.users[name]; ok {
		return u, nil
	}
	return nil, db.ErrNotFound
}

func (f *fakeSource) UserByWikidotName(name string) (*db.User, error) {
	if u, ok := f.wikidotUsers[name]; ok {
		return u, nil
	}
	return nil, db.ErrNotFound
}

func (f *fakeSource) SiteRatingMode() (string, error) {
	if f.siteMode == "" {
		return "", db.ErrNotFound
	}
	return f.siteMode, nil
}

func (f *fakeSource) CategoryRatingMode(category string) (string, error) {
	if mode, ok := f.categoryMode[category]; ok {
		return mode, nil
	}
	return "", db.ErrNotFound
}

func (f *fakeSource) VoteStats(int64) (db.VoteStats, error) { return f.votes, nil }

func (f *fakeSource) HiddenCategories(*db.User) ([]string, error) { return f.hidden, nil }

func (f *fakeSource) ListArticles(filter db.ListFilter, offset int, limit *int) ([]db.Article, error) {
	f.gotFilter, f.gotOffset, f.gotLimit = filter, offset, limit
	f.listCalls++
	return f.listed, nil
}

func (f *fakeSource) CountArticles(db.ListFilter, int, *int) (int, error) { return f.total, nil }

func article173() *db.Article {
	parent := int64(3)
	return &db.Article{
		ID: 7, Category: "scp", Name: "scp-173", Title: "173", ParentID: &parent,
		CreatedAt: time.Date(2021, 6, 5, 14, 30, 0, 0, time.UTC),
	}
}

func parse(t *testing.T, src Source, a *db.Article, viewer *db.User, params map[string]string) Query {
	t.Helper()
	q, err := Parse(src, a, viewer, params, nil)
	if err != nil {
		t.Fatalf("Parse(%v) err = %v, want nil", params, err)
	}
	return q
}

func TestParseDefaultsToTheOwnCategoryAndNormalPages(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, nil)

	if q.Filter.PageType != db.PageTypeNormal {
		t.Errorf("PageType = %q, want %q", q.Filter.PageType, db.PageTypeNormal)
	}
	if len(q.Filter.Categories) != 1 || q.Filter.Categories[0] != "scp" {
		t.Errorf("Categories = %v, want [scp]", q.Filter.Categories)
	}
	if q.Filter.Sort != (db.Sort{Column: db.SortCreatedAt}) {
		t.Errorf("Sort = %+v, want created_at desc", q.Filter.Sort)
	}
	if q.PerPage != 20 || q.Page != 1 {
		t.Errorf("Page, PerPage = %d, %d, want 1, 20", q.Page, q.PerPage)
	}
}

func TestParseWithoutAnArticleHasNoCategory(t *testing.T) {
	q := parse(t, newFakeSource(), nil, nil, nil)
	if !q.Invalid {
		t.Error("Parse(nil article).Invalid = false, want true")
	}
}

func TestParseNameDot(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"name": "."})
	if !q.HasOnly || q.Only.ID != 7 {
		t.Errorf("Parse(name=.).Only = %+v, want article 7", q.Only)
	}
}

func TestParseRangeDotAndFullNameDotMeanTheSame(t *testing.T) {
	for _, key := range []string{"range", "fullname"} {
		q := parse(t, newFakeSource(), article173(), nil, map[string]string{key: "."})
		if !q.HasOnly {
			t.Errorf("Parse(%s=.).HasOnly = false, want true", key)
		}
	}
}

func TestParseFullName(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"fullname": "main"})
	if !q.HasFullName || q.FullName != "main" {
		t.Errorf("Parse(fullname=main).FullName = %q, want %q", q.FullName, "main")
	}
}

func TestParseNamePercentBecomesAPrefix(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"name": "SCP-%"})
	if !q.Filter.HasNamePrefix || q.Filter.NamePrefix != "scp-" {
		t.Errorf("NamePrefix = %q, want %q", q.Filter.NamePrefix, "scp-")
	}
}

func TestParseNameEqualsTakesTheOwnName(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"name": "="})
	if !q.Filter.HasName || q.Filter.Name != "scp-173" {
		t.Errorf("Name = %q, want %q", q.Filter.Name, "scp-173")
	}
}

func TestParseNameStarListsEverything(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"name": "*"})
	if q.Filter.HasName || q.Filter.HasNamePrefix {
		t.Errorf("Name, NamePrefix set = %v, %v, want false, false", q.Filter.HasName, q.Filter.HasNamePrefix)
	}
}

func TestParseTagsSplitsBySign(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"tags": "+euclid,safe,-scp:hub"})
	if len(q.Filter.RequiredTags) != 1 || q.Filter.RequiredTags[0] != 1 {
		t.Errorf("RequiredTags = %v, want [1]", q.Filter.RequiredTags)
	}
	if len(q.Filter.PresentTags) != 1 || q.Filter.PresentTags[0] != 4 {
		t.Errorf("PresentTags = %v, want [4]", q.Filter.PresentTags)
	}
	if len(q.Filter.AbsentTags) != 1 || q.Filter.AbsentTags[0] != 3 {
		t.Errorf("AbsentTags = %v, want [3]", q.Filter.AbsentTags)
	}
}

func TestParseTagsWithAnUnknownRequiredTag(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"tags": "+nosuchtag"})
	if !q.Invalid {
		t.Error("Parse(tags=+nosuchtag).Invalid = false, want true")
	}
}

func TestParseTagsWithOnlyUnknownPresentTags(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"tags": "nosuchtag"})
	if !q.Invalid {
		t.Error("Parse(tags=nosuchtag).Invalid = false, want true")
	}
}

func TestParseTagsKeepsGoingWhenOneOfSeveralIsUnknown(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"tags": "nosuchtag euclid"})
	if q.Invalid {
		t.Error("Parse(tags=nosuchtag euclid).Invalid = true, want false")
	}
	if len(q.Filter.PresentTags) != 1 || q.Filter.PresentTags[0] != 1 {
		t.Errorf("PresentTags = %v, want [1]", q.Filter.PresentTags)
	}
}

func TestParseTagsWithAnUnknownAbsentTag(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"tags": "-nosuchtag"})
	if q.Invalid {
		t.Error("Parse(tags=-nosuchtag).Invalid = true, want false")
	}
	if len(q.Filter.AbsentTags) != 0 {
		t.Errorf("AbsentTags = %v, want []", q.Filter.AbsentTags)
	}
}

func TestParseTagsDash(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"tags": "-"})
	if !q.Filter.NoTags {
		t.Error("Parse(tags=-).NoTags = false, want true")
	}
}

func TestParseTagsEqualsTakesTheOwnTags(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"tags": "="})
	if len(q.Filter.RequiredTags) != 2 {
		t.Errorf("RequiredTags = %v, want two entries", q.Filter.RequiredTags)
	}
	if len(q.Filter.ExactTags) != 0 {
		t.Errorf("ExactTags = %v, want []", q.Filter.ExactTags)
	}
}

func TestParseTagsDoubleEqualsTakesTheOwnTags(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"tags": "=="})
	if len(q.Filter.ExactTags) != 2 {
		t.Errorf("ExactTags = %v, want two entries", q.Filter.ExactTags)
	}
}

func TestParseTagsWithoutACategoryMatchesEveryCategory(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"tags": "hub"})
	if len(q.Filter.PresentTags) != 2 {
		t.Errorf("PresentTags = %v, want two entries", q.Filter.PresentTags)
	}
}

func TestParseCategoryStarListsTheWholeSite(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"category": "*"})
	if len(q.Filter.Categories) != 0 || len(q.Filter.NotCategories) != 0 {
		t.Errorf("Categories, NotCategories = %v, %v, want empty", q.Filter.Categories, q.Filter.NotCategories)
	}
}

func TestParseCategoryDropsWhatFollowsAColon(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"category": "scp:page -meta:page"})
	if len(q.Filter.Categories) != 1 || q.Filter.Categories[0] != "scp" {
		t.Errorf("Categories = %v, want [scp]", q.Filter.Categories)
	}
	if len(q.Filter.NotCategories) != 1 || q.Filter.NotCategories[0] != "meta" {
		t.Errorf("NotCategories = %v, want [meta]", q.Filter.NotCategories)
	}
}

func TestParseCategoryDotInAList(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"category": ". meta"})
	want := []string{"scp", "meta"}
	if len(q.Filter.Categories) != 2 || q.Filter.Categories[0] != want[0] || q.Filter.Categories[1] != want[1] {
		t.Errorf("Categories = %v, want %v", q.Filter.Categories, want)
	}
}

func TestParseParentDash(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"parent": "-"})
	if !q.Filter.HasParent || q.Filter.ParentID != nil {
		t.Errorf("HasParent, ParentID = %v, %v, want true, nil", q.Filter.HasParent, q.Filter.ParentID)
	}
}

func TestParseParentEqualsTakesTheOwnParent(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"parent": "="})
	if q.Filter.ParentID == nil || *q.Filter.ParentID != 3 {
		t.Errorf("ParentID = %v, want 3", q.Filter.ParentID)
	}
}

func TestParseParentDashEquals(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"parent": "-="})
	if !q.Filter.HasNotParent || q.Filter.NotParentID == nil || *q.Filter.NotParentID != 3 {
		t.Errorf("NotParentID = %v, want 3", q.Filter.NotParentID)
	}
}

func TestParseParentDotIsThePageItself(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"parent": "."})
	if q.Filter.ParentID == nil || *q.Filter.ParentID != 7 {
		t.Errorf("ParentID = %v, want 7", q.Filter.ParentID)
	}
}

func TestParseParentByName(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"parent": "MAIN"})
	if q.Filter.ParentID == nil || *q.Filter.ParentID != 3 {
		t.Errorf("ParentID = %v, want 3", q.Filter.ParentID)
	}
}

func TestParseParentThatDoesNotExist(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"parent": "no-such-page"})
	if !q.Invalid {
		t.Error("Parse(parent=no-such-page).Invalid = false, want true")
	}
}

func TestParseCreatedByDotWithoutAViewer(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"created_by": "."})
	if !q.Invalid {
		t.Error("Parse(created_by=.).Invalid = false, want true")
	}
}

func TestParseCreatedByDotTakesTheViewer(t *testing.T) {
	viewer := &db.User{ID: 99, Username: "carol"}
	q := parse(t, newFakeSource(), article173(), viewer, map[string]string{"created_by": "."})
	if q.Filter.AuthorID == nil || *q.Filter.AuthorID != 99 {
		t.Errorf("AuthorID = %v, want 99", q.Filter.AuthorID)
	}
}

func TestParseCreatedByWikidotPrefix(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"created_by": "wd:Bob"})
	if q.Filter.AuthorID == nil || *q.Filter.AuthorID != 12 {
		t.Errorf("AuthorID = %v, want 12", q.Filter.AuthorID)
	}
}

func TestParseCreatedByUnknownUser(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"created_by": "nobody"})
	if !q.Invalid {
		t.Error("Parse(created_by=nobody).Invalid = false, want true")
	}
}

func TestParseCreatedAtBounds(t *testing.T) {
	cases := []struct {
		in         string
		op         string
		start, end string
	}{
		{"2021", db.TimeRange, "2021-01-01T00:00:00Z", "2021-12-31T00:00:00Z"},
		{"2021-02", db.TimeRange, "2021-02-01T00:00:00Z", "2021-02-28T00:00:00Z"},
		{"2020-02", db.TimeRange, "2020-02-01T00:00:00Z", "2020-02-29T00:00:00Z"},
		{"2021-02-09", db.TimeRange, "2021-02-09T00:00:00Z", "2021-02-09T00:00:00Z"},
		{"2021-13-99", db.TimeRange, "2021-12-31T00:00:00Z", "2021-12-31T00:00:00Z"},
		{"2021-00-00", db.TimeRange, "2021-01-01T00:00:00Z", "2021-01-01T00:00:00Z"},
		{">2021", db.TimeGT, "2021-01-01T00:00:00Z", "2021-12-31T00:00:00Z"},
		{">=2021", db.TimeGTE, "2021-01-01T00:00:00Z", "2021-12-31T00:00:00Z"},
		{"<2021", db.TimeLT, "2021-01-01T00:00:00Z", "2021-12-31T00:00:00Z"},
		{"<=2021", db.TimeLTE, "2021-01-01T00:00:00Z", "2021-12-31T00:00:00Z"},
		{"<>2021", db.TimeExcludeRange, "2021-01-01T00:00:00Z", "2021-12-31T00:00:00Z"},
	}
	for _, c := range cases {
		q := parse(t, newFakeSource(), article173(), nil, map[string]string{"created_at": c.in})
		got := q.Filter.CreatedAt
		if got == nil {
			t.Errorf("Parse(created_at=%q).CreatedAt = nil, want a filter", c.in)
			continue
		}
		if got.Op != c.op {
			t.Errorf("Parse(created_at=%q).Op = %q, want %q", c.in, got.Op, c.op)
		}
		if start := got.Start.Format(time.RFC3339); start != c.start {
			t.Errorf("Parse(created_at=%q).Start = %q, want %q", c.in, start, c.start)
		}
		if end := got.End.Format(time.RFC3339); end != c.end {
			t.Errorf("Parse(created_at=%q).End = %q, want %q", c.in, end, c.end)
		}
	}
}

func TestParseCreatedAtEqualsIsTheOwnDay(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"created_at": "="})
	got := q.Filter.CreatedAt
	if got == nil {
		t.Fatal("Parse(created_at==).CreatedAt = nil, want a filter")
	}
	if start := got.Start.Format(time.RFC3339); start != "2021-06-05T00:00:00Z" {
		t.Errorf("Start = %q, want %q", start, "2021-06-05T00:00:00Z")
	}
	if end := got.End.Format(time.RFC3339); end != "2021-06-05T23:59:59Z" {
		t.Errorf("End = %q, want %q", end, "2021-06-05T23:59:59Z")
	}
}

func TestParseLinkTo(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"link_to": "theme:black"})
	if !q.Filter.HasLinkTo || q.Filter.LinkTo != "theme:black" {
		t.Errorf("LinkTo = %q, want %q", q.Filter.LinkTo, "theme:black")
	}
}

func TestParseLinkToDotIsTheOwnPage(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"link_to": "."})
	if q.Filter.LinkTo != "scp:scp-173" {
		t.Errorf("LinkTo = %q, want %q", q.Filter.LinkTo, "scp:scp-173")
	}
}

func TestParseLinkToDotWithoutAnArticle(t *testing.T) {
	q := parse(t, newFakeSource(), nil, nil, map[string]string{"link_to": "."})
	if q.Filter.HasLinkTo {
		t.Errorf("Parse(link_to=., nil article).LinkTo = %q, want it unset", q.Filter.LinkTo)
	}
}

func TestParseUpdatedAtBounds(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"updated_at": ">=2021-02"})
	got := q.Filter.UpdatedAt
	if got == nil {
		t.Fatal("Parse(updated_at=>=2021-02).UpdatedAt = nil, want a filter")
	}
	if got.Op != db.TimeGTE {
		t.Errorf("Op = %q, want %q", got.Op, db.TimeGTE)
	}
	if end := got.End.Format(time.RFC3339); end != "2021-02-28T00:00:00Z" {
		t.Errorf("End = %q, want %q", end, "2021-02-28T00:00:00Z")
	}
}

func TestParseUpdatedAtDoesNotTouchCreatedAt(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"updated_at": "2021"})
	if q.Filter.CreatedAt != nil {
		t.Errorf("CreatedAt = %+v, want nil", q.Filter.CreatedAt)
	}
}

func TestParseUpdatedAtEqualsIsTheOwnDay(t *testing.T) {
	a := article173()
	a.UpdatedAt = time.Date(2022, 3, 9, 8, 0, 0, 0, time.UTC)
	q := parse(t, newFakeSource(), a, nil, map[string]string{"updated_at": "="})
	got := q.Filter.UpdatedAt
	if got == nil {
		t.Fatal("Parse(updated_at==).UpdatedAt = nil, want a filter")
	}
	if start := got.Start.Format(time.RFC3339); start != "2022-03-09T00:00:00Z" {
		t.Errorf("Start = %q, want %q", start, "2022-03-09T00:00:00Z")
	}
	if end := got.End.Format(time.RFC3339); end != "2022-03-09T23:59:59Z" {
		t.Errorf("End = %q, want %q", end, "2022-03-09T23:59:59Z")
	}
}

func TestParseOrderBySize(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"order": "size desc"})
	if want := (db.Sort{Column: db.SortSize}); q.Filter.Sort != want {
		t.Errorf("Sort = %+v, want %+v", q.Filter.Sort, want)
	}
}

func TestParseCreatedAtThatDoesNotParse(t *testing.T) {
	for _, in := range []string{"twenty", "2021-", "-5", "0", "10000", " >2021"} {
		q := parse(t, newFakeSource(), article173(), nil, map[string]string{"created_at": in})
		if !q.Invalid {
			t.Errorf("Parse(created_at=%q).Invalid = false, want true", in)
		}
	}
}

func TestParseIgnoresTheLegacyDateSpelling(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"date": "2021"})
	if q.Filter.CreatedAt != nil {
		t.Errorf("Parse(date=2021).CreatedAt = %+v, want nil", q.Filter.CreatedAt)
	}
}

func TestParseRatingOperators(t *testing.T) {
	cases := []struct {
		in    string
		op    string
		value float64
	}{
		{"5", db.NumEQ, 5},
		{"=5", db.NumEQ, 5},
		{">5", db.NumGT, 5},
		{">=5", db.NumGTE, 5},
		{"<5", db.NumLT, 5},
		{"<=5", db.NumLTE, 5},
		{"<>5", db.NumNE, 5},
		{"3.5", db.NumEQ, 3.5},
		{"-2", db.NumEQ, -2},
	}
	for _, c := range cases {
		q := parse(t, newFakeSource(), article173(), nil, map[string]string{"rating": c.in})
		got := q.Filter.Rating
		if got == nil {
			t.Errorf("Parse(rating=%q).Rating = nil, want a filter", c.in)
			continue
		}
		if got.Op != c.op || got.Value != c.value {
			t.Errorf("Parse(rating=%q) = %s %v, want %s %v", c.in, got.Op, got.Value, c.op, c.value)
		}
	}
}

func TestParseVotesRejectsAFraction(t *testing.T) {
	q := parse(t, newFakeSource(), article173(), nil, map[string]string{"votes": "3.5"})
	if !q.Invalid {
		t.Error("Parse(votes=3.5).Invalid = false, want true")
	}
}

func TestParseRatingModeComesFromTheFirstCategory(t *testing.T) {
	src := newFakeSource()
	src.siteMode = page.RatingModeUpDown
	src.categoryMode["meta"] = page.RatingModeStars

	q := parse(t, src, article173(), nil, map[string]string{"rating": ">1", "category": "meta scp"})
	if q.Filter.RatingMode != page.RatingModeStars {
		t.Errorf("RatingMode = %q, want %q", q.Filter.RatingMode, page.RatingModeStars)
	}
}

func TestParseRatingModeIsLeftUnsetWhenNothingReadsIt(t *testing.T) {
	src := newFakeSource()
	src.siteMode = page.RatingModeStars
	q := parse(t, src, article173(), nil, nil)
	if q.Filter.RatingMode != "" {
		t.Errorf("RatingMode = %q, want %q", q.Filter.RatingMode, "")
	}
}

func TestParseSort(t *testing.T) {
	cases := []struct {
		in        string
		column    string
		ascending bool
	}{
		{"name", db.SortName, true},
		{"name desc", db.SortName, false},
		{"name asc", db.SortName, true},
		{"name desc extra", db.SortName, true},
		{"nosuchcolumn", "nosuchcolumn", true},
		{"", "", true},
		{"random", db.SortRandom, true},
	}
	for _, c := range cases {
		q := parse(t, newFakeSource(), article173(), nil, map[string]string{"order": c.in})
		if q.Filter.Sort.Column != c.column || q.Filter.Sort.Ascending != c.ascending {
			t.Errorf("Parse(order=%q).Sort = %+v, want %s %v", c.in, q.Filter.Sort, c.column, c.ascending)
		}
	}
}

func TestParseWindow(t *testing.T) {
	src := newFakeSource()
	q, err := Parse(src, article173(), nil,
		map[string]string{"offset": "5", "limit": "40", "perpage": "300"},
		page.PathParams{{Key: "p", Value: "3"}})
	if err != nil {
		t.Fatalf("Parse() err = %v, want nil", err)
	}
	if q.Offset != 5 {
		t.Errorf("Offset = %d, want 5", q.Offset)
	}
	if q.Limit == nil || *q.Limit != 40 {
		t.Errorf("Limit = %v, want 40", q.Limit)
	}
	if q.PerPage != 300 {
		t.Errorf("PerPage = %d, want 300", q.PerPage)
	}
	if q.Page != 3 {
		t.Errorf("Page = %d, want 3", q.Page)
	}
}

func TestParseWindowFallsBackOnJunk(t *testing.T) {
	src := newFakeSource()
	q, err := Parse(src, article173(), nil,
		map[string]string{"offset": "x", "limit": "y", "perpage": "z"},
		page.PathParams{{Key: "p", Value: "0"}})
	if err != nil {
		t.Fatalf("Parse() err = %v, want nil", err)
	}
	if q.Offset != 0 {
		t.Errorf("Offset = %d, want 0", q.Offset)
	}
	if q.Limit != nil {
		t.Errorf("Limit = %v, want nil", q.Limit)
	}
	if q.PerPage != 20 {
		t.Errorf("PerPage = %d, want 20", q.PerPage)
	}
	if q.Page != 1 {
		t.Errorf("Page = %d, want 1", q.Page)
	}
}
