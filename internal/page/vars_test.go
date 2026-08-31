package page

import (
	"errors"
	"testing"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

type fakeSource struct {
	source    string
	sourceErr error
	authors   []db.User
	editor    *db.User
	editorErr error
	revisions int
	tags      []string
	votes     db.VoteStats
	siteMode  string
	catMode   string
	voted     bool
	parent    *db.Article
	err       error
	calls     map[string]int
}

func newFakeSource() *fakeSource {
	return &fakeSource{
		siteMode:  "",
		catMode:   "",
		editorErr: db.ErrNotFound,
		sourceErr: db.ErrNotFound,
		calls:     map[string]int{},
	}
}

func (f *fakeSource) count(name string) { f.calls[name]++ }

func (f *fakeSource) LatestSource(articleID int64) (string, error) {
	f.count("LatestSource")
	if f.source != "" {
		return f.source, nil
	}
	return "", f.sourceErr
}

func (f *fakeSource) Authors(articleID int64) ([]db.User, error) {
	f.count("Authors")
	return f.authors, f.err
}

func (f *fakeSource) LatestEditor(articleID int64) (*db.User, error) {
	f.count("LatestEditor")
	if f.editor != nil {
		return f.editor, nil
	}
	return nil, f.editorErr
}

func (f *fakeSource) RevisionCount(articleID int64) (int, error) {
	f.count("RevisionCount")
	return f.revisions, f.err
}

func (f *fakeSource) Tags(articleID int64) ([]string, error) {
	f.count("Tags")
	return f.tags, f.err
}

func (f *fakeSource) VoteStats(articleID int64) (db.VoteStats, error) {
	f.count("VoteStats")
	return f.votes, f.err
}

func (f *fakeSource) SiteRatingMode() (string, error) {
	f.count("SiteRatingMode")
	if f.siteMode == "" {
		return "", db.ErrNotFound
	}
	return f.siteMode, nil
}

func (f *fakeSource) CategoryRatingMode(category string) (string, error) {
	f.count("CategoryRatingMode")
	if f.catMode == "" {
		return "", db.ErrNotFound
	}
	return f.catMode, nil
}

func (f *fakeSource) HasVoted(articleID int64, userID *int64) (bool, error) {
	f.count("HasVoted")
	return f.voted, f.err
}

func (f *fakeSource) ArticleByID(id int64) (*db.Article, error) {
	f.count("ArticleByID")
	if f.parent == nil {
		return nil, db.ErrNotFound
	}
	return f.parent, nil
}

func testArticle() *db.Article {
	return &db.Article{
		ID:        7,
		Category:  db.DefaultCategory,
		Name:      "main",
		Title:     "Host Title",
		CreatedAt: time.Unix(1600000000, 0),
		UpdatedAt: time.Unix(1700000000, 0),
	}
}

func TestLookupArticleFields(t *testing.T) {
	v := NewVars(testArticle(), nil, newFakeSource(), nil)
	tests := []struct{ name, want string }{
		{"name", "main"},
		{"category", "_default"},
		{"fullname", "main"},
		{"title", "Host Title"},
		{"title_linked", "[[[main|]]]"},
		{"linked_title", "[[[main|]]]"},
		{"link", "/Host Title"},
		{"created_at", "[[date 1600000000]]"},
		{"updated_at", "[[date 1700000000]]"},
	}
	for _, tt := range tests {
		got, ok := v.Lookup(tt.name)
		if !ok {
			t.Errorf("Lookup(%q) = _, false, want true", tt.name)
			continue
		}
		if got != tt.want {
			t.Errorf("Lookup(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestLookupCategoryPrefixInFullName(t *testing.T) {
	a := testArticle()
	a.Category = "component"
	a.Name = "box"
	v := NewVars(a, nil, newFakeSource(), nil)

	got, _ := v.Lookup("fullname")
	if want := "component:box"; got != want {
		t.Errorf("Lookup(\"fullname\") = %q, want %q", got, want)
	}
}

func TestLookupIsCaseSensitive(t *testing.T) {
	v := NewVars(testArticle(), nil, newFakeSource(), nil)
	if _, ok := v.Lookup("Title"); ok {
		t.Error("Lookup(\"Title\") = _, true, want false")
	}
}

func TestLookupUnknownName(t *testing.T) {
	v := NewVars(testArticle(), nil, newFakeSource(), nil)
	if _, ok := v.Lookup("nosuchvar"); ok {
		t.Error("Lookup(\"nosuchvar\") = _, true, want false")
	}
}

func TestLookupDateFormat(t *testing.T) {
	v := NewVars(testArticle(), nil, newFakeSource(), nil)
	tests := []struct{ name, want string }{
		{"created_at|%Y", `[[date 1600000000 format="%Y"]]`},
		{"updated_at| %d.%m.%Y ", `[[date 1700000000 format="%d.%m.%Y"]]`},
		{"created_at|", `[[date 1600000000 format=""]]`},
	}
	for _, tt := range tests {
		got, ok := v.Lookup(tt.name)
		if !ok {
			t.Errorf("Lookup(%q) = _, false, want true", tt.name)
			continue
		}
		if got != tt.want {
			t.Errorf("Lookup(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestThisPrefix(t *testing.T) {
	v := NewVars(testArticle(), nil, newFakeSource(), nil)
	tests := []struct {
		param string
		want  string
		ok    bool
	}{
		{"this|title", "Host Title", true},
		{"this|TITLE", "Host Title", true},
		{"this|Title", "Host Title", true},
		{"title", "", false},
		{"notthis", "", false},
		{"THIS|title", "", false},
		{"this|created_at|%Y", "", false},
		{"this|nosuchvar", "", false},
	}
	for _, tt := range tests {
		got, ok := v.This(tt.param)
		if ok != tt.ok {
			t.Errorf("This(%q) = _, %v, want %v", tt.param, ok, tt.ok)
			continue
		}
		if got != tt.want {
			t.Errorf("This(%q) = %q, want %q", tt.param, got, tt.want)
		}
	}
}

func TestLookupQueriesNothingForArticleFields(t *testing.T) {
	src := newFakeSource()
	v := NewVars(testArticle(), nil, src, nil)

	v.Lookup("title")
	v.Lookup("fullname")

	if got := len(src.calls); got != 0 {
		t.Errorf("len(calls) = %d, want 0", got)
	}
}

func TestLookupQueriesOncePerVariable(t *testing.T) {
	src := newFakeSource()
	src.source = "body"
	v := NewVars(testArticle(), nil, src, nil)

	v.Lookup("content")
	v.Lookup("content")
	v.This("this|content")

	if got := src.calls["LatestSource"]; got != 1 {
		t.Errorf("calls[LatestSource] = %d, want 1", got)
	}
}

func TestLookupSharesAuthorsAcrossVariables(t *testing.T) {
	src := newFakeSource()
	src.authors = []db.User{{Username: "alice"}}
	v := NewVars(testArticle(), nil, src, nil)

	v.Lookup("created_by")
	v.Lookup("created_by_linked")
	v.Lookup("authors_count")

	if got := src.calls["Authors"]; got != 1 {
		t.Errorf("calls[Authors] = %d, want 1", got)
	}
}

func TestLookupContentMissingLeavesVariableUnresolved(t *testing.T) {
	src := newFakeSource()
	v := NewVars(testArticle(), nil, src, nil)

	if _, ok := v.Lookup("content"); ok {
		t.Error("Lookup(\"content\") = _, true, want false")
	}
	if err := v.Err(); err != nil {
		t.Errorf("Err() = %v, want nil", err)
	}
}

func TestLookupRecordsQueryError(t *testing.T) {
	src := newFakeSource()
	src.err = errors.New("connection refused")
	v := NewVars(testArticle(), nil, src, nil)

	if _, ok := v.Lookup("revisions"); ok {
		t.Error("Lookup(\"revisions\") = _, true, want false")
	}
	if err := v.Err(); !errors.Is(err, src.err) {
		t.Errorf("Err() = %v, want %v", err, src.err)
	}
}

func TestLookupUsers(t *testing.T) {
	src := newFakeSource()
	src.authors = []db.User{
		{Username: "alice", DisplayName: "Alice"},
		{Username: "bob"},
		{Type: db.UserTypeWikidot, Username: "carol", WikidotUsername: "Carol WD"},
	}
	v := NewVars(testArticle(), nil, src, nil)

	tests := []struct{ name, want string }{
		{"created_by", "[[span]]Alice[[/span]] [[span]]bob[[/span]] [[span]]wd:Carol WD[[/span]]"},
		{"created_by_linked", "[[*user alice]] [[*user bob]] [[*user carol]]"},
		{"created_by_linked_plain", "[[user alice]] [[user bob]] [[user carol]]"},
		{"authors_count", "3"},
	}
	for _, tt := range tests {
		got, _ := v.Lookup(tt.name)
		if got != tt.want {
			t.Errorf("Lookup(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestLookupEditorMissingIsSystem(t *testing.T) {
	v := NewVars(testArticle(), nil, newFakeSource(), nil)

	tests := []struct{ name, want string }{
		{"updated_by", "user-system"},
		{"updated_by_linked", "user-system"},
		{"updated_by_linked_plain", "user-system"},
	}
	for _, tt := range tests {
		got, ok := v.Lookup(tt.name)
		if !ok {
			t.Errorf("Lookup(%q) = _, false, want true", tt.name)
			continue
		}
		if got != tt.want {
			t.Errorf("Lookup(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestLookupEditor(t *testing.T) {
	src := newFakeSource()
	src.editor = &db.User{Username: "dave", DisplayName: "Dave"}
	v := NewVars(testArticle(), nil, src, nil)

	tests := []struct{ name, want string }{
		{"updated_by", "Dave"},
		{"updated_by_linked", "[[*user dave]]"},
		{"updated_by_linked_plain", "[[user dave]]"},
	}
	for _, tt := range tests {
		got, _ := v.Lookup(tt.name)
		if got != tt.want {
			t.Errorf("Lookup(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestLookupTags(t *testing.T) {
	src := newFakeSource()
	src.tags = []string{"Zeta", "alpha", "lang:EN"}
	v := NewVars(testArticle(), nil, src, nil)

	got, _ := v.Lookup("tags")
	if want := "alpha, lang:en, zeta"; got != want {
		t.Errorf("Lookup(\"tags\") = %q, want %q", got, want)
	}
}

func TestLookupTagsLinkedEscapesSeparators(t *testing.T) {
	src := newFakeSource()
	src.tags = []string{"lang:en", "two words"}
	v := NewVars(testArticle(), nil, src, nil)

	got, _ := v.Lookup("tags_linked")
	want := "[/system:page-tags/tag/lang%3Aen lang:en], [/system:page-tags/tag/two%20words two words]"
	if got != want {
		t.Errorf("Lookup(\"tags_linked\") = %q, want %q", got, want)
	}
}

func TestLookupRatingUpDown(t *testing.T) {
	tests := []struct {
		sum  float64
		want string
	}{
		{3, "+3"},
		{0, "+0"},
		{-2, "-2"},
	}
	for _, tt := range tests {
		src := newFakeSource()
		src.votes = db.VoteStats{Sum: tt.sum, Count: 4}
		v := NewVars(testArticle(), nil, src, nil)

		got, _ := v.Lookup("rating")
		if got != tt.want {
			t.Errorf("Lookup(\"rating\") with sum %v = %q, want %q", tt.sum, got, tt.want)
		}
	}
}

func TestLookupRatingStars(t *testing.T) {
	src := newFakeSource()
	src.siteMode = RatingModeStars
	src.votes = db.VoteStats{Count: 3, Average: 4.25}
	v := NewVars(testArticle(), nil, src, nil)

	got, _ := v.Lookup("rating")
	if want := "4.2"; got != want {
		t.Errorf("Lookup(\"rating\") = %q, want %q", got, want)
	}
}

func TestLookupRatingStarsWithoutVotes(t *testing.T) {
	src := newFakeSource()
	src.siteMode = RatingModeStars
	v := NewVars(testArticle(), nil, src, nil)

	got, _ := v.Lookup("rating")
	if want := "—"; got != want {
		t.Errorf("Lookup(\"rating\") = %q, want %q", got, want)
	}
}

func TestLookupRatingDisabled(t *testing.T) {
	src := newFakeSource()
	src.siteMode = RatingModeDisabled
	src.votes = db.VoteStats{Sum: 9, Count: 9}
	v := NewVars(testArticle(), nil, src, nil)

	got, _ := v.Lookup("rating")
	if want := "0"; got != want {
		t.Errorf("Lookup(\"rating\") = %q, want %q", got, want)
	}
}

func TestRatingModeCategoryOverridesSite(t *testing.T) {
	src := newFakeSource()
	src.siteMode = RatingModeStars
	src.catMode = RatingModeUpDown
	src.votes = db.VoteStats{Sum: 5, Count: 5}
	v := NewVars(testArticle(), nil, src, nil)

	got, _ := v.Lookup("rating")
	if want := "+5"; got != want {
		t.Errorf("Lookup(\"rating\") = %q, want %q", got, want)
	}
}

func TestRatingModeDefaultDoesNotOverride(t *testing.T) {
	src := newFakeSource()
	src.siteMode = RatingModeStars
	src.catMode = RatingModeDefault
	src.votes = db.VoteStats{Count: 2, Average: 3}
	v := NewVars(testArticle(), nil, src, nil)

	got, _ := v.Lookup("rating")
	if want := "3.0"; got != want {
		t.Errorf("Lookup(\"rating\") = %q, want %q", got, want)
	}
}

func TestLookupPopularity(t *testing.T) {
	tests := []struct {
		good, count int
		want        string
	}{
		{1, 2, "50"},
		{0, 0, "0"},
		{1, 8, "12"},
		{3, 8, "38"},
		{2, 3, "67"},
	}
	for _, tt := range tests {
		src := newFakeSource()
		src.votes = db.VoteStats{Count: tt.count, GoodUpDown: tt.good}
		v := NewVars(testArticle(), nil, src, nil)

		got, _ := v.Lookup("popularity")
		if got != tt.want {
			t.Errorf("Lookup(\"popularity\") with %d/%d = %q, want %q", tt.good, tt.count, got, tt.want)
		}
	}
}

func TestLookupRatingVotes(t *testing.T) {
	src := newFakeSource()
	src.votes = db.VoteStats{Count: 12}
	v := NewVars(testArticle(), nil, src, nil)

	got, _ := v.Lookup("rating_votes")
	if want := "12"; got != want {
		t.Errorf("Lookup(\"rating_votes\") = %q, want %q", got, want)
	}
}

func TestLookupCurrentUserVoted(t *testing.T) {
	tests := []struct {
		voted bool
		want  string
	}{
		{true, "True"},
		{false, "False"},
	}
	for _, tt := range tests {
		src := newFakeSource()
		src.voted = tt.voted
		v := NewVars(testArticle(), &db.User{ID: 3}, src, nil)

		got, _ := v.Lookup("current_user_voted")
		if got != tt.want {
			t.Errorf("Lookup(\"current_user_voted\") = %q, want %q", got, tt.want)
		}
	}
}

func TestLookupParentAbsentWithoutParent(t *testing.T) {
	src := newFakeSource()
	v := NewVars(testArticle(), nil, src, nil)

	if _, ok := v.Lookup("parent_title"); ok {
		t.Error("Lookup(\"parent_title\") = _, true, want false")
	}
	if got := src.calls["ArticleByID"]; got != 0 {
		t.Errorf("calls[ArticleByID] = %d, want 0", got)
	}
}

func TestLookupParent(t *testing.T) {
	src := newFakeSource()
	src.parent = &db.Article{ID: 2, Category: "component", Name: "box", Title: "Box"}
	a := testArticle()
	parentID := int64(2)
	a.ParentID = &parentID
	v := NewVars(a, nil, src, nil)

	tests := []struct{ name, want string }{
		{"parent_name", "box"},
		{"parent_category", "component"},
		{"parent_fullname", "component:box"},
		{"parent_title", "Box"},
		{"parent_title_linked", "[[[component:box|Box]]]"},
		{"parent_linked_title", "[[[component:box|Box]]]"},
	}
	for _, tt := range tests {
		got, ok := v.Lookup(tt.name)
		if !ok {
			t.Errorf("Lookup(%q) = _, false, want true", tt.name)
			continue
		}
		if got != tt.want {
			t.Errorf("Lookup(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
	if got := src.calls["ArticleByID"]; got != 1 {
		t.Errorf("calls[ArticleByID] = %d, want 1", got)
	}
}

func TestThisVarsPass(t *testing.T) {
	v := NewVars(testArticle(), nil, newFakeSource(), nil)
	tests := []struct{ in, want string }{
		{"%%this|title%%", "Host Title"},
		{"%%notthis%%", "%%notthis%%"},
		{"%%title%%", "%%title%%"},
		{"%%a%% %%this|name%%", "%%a%% main"},
		{"%%this|nosuchvar%%", "%%this|nosuchvar%%"},
	}
	for _, tt := range tests {
		if got := ThisVars(tt.in, v); got != tt.want {
			t.Errorf("ThisVars(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestPageVarsPass(t *testing.T) {
	src := newFakeSource()
	src.source = "body text"
	v := NewVars(testArticle(), nil, src, nil)

	tests := []struct{ in, want string }{
		{"%%content%%", "body text"},
		{"%%title%% %%index%%/%%total%%", "Host Title 1/1"},
		{"%%this|title%%", "%%this|title%%"},
		{"%%Title%%", "%%Title%%"},
	}
	for _, tt := range tests {
		if got := PageVars(tt.in, v, 1, 1); got != tt.want {
			t.Errorf("PageVars(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestPyRound(t *testing.T) {
	tests := []struct {
		in   float64
		want int
	}{
		{0.5, 0},
		{1.5, 2},
		{2.5, 2},
		{-0.5, 0},
		{-1.5, -2},
		{12.4, 12},
		{12.6, 13},
	}
	for _, tt := range tests {
		if got := roundHalfEven(tt.in); got != tt.want {
			t.Errorf("roundHalfEven(%v) = %d, want %d", tt.in, got, tt.want)
		}
	}
}
