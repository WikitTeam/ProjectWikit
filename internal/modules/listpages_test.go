package modules

import (
	"strings"
	"testing"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

type listData struct {
	listed []db.Article
	total  int
	filter db.ListFilter
	offset int
	limit  *int
}

func (d *listData) TagArticles(string, string, []string) ([]db.Article, error) { return nil, nil }

func (d *listData) TagCategory(string) (db.TagCategory, error) {
	return db.TagCategory{}, db.ErrNotFound
}

func (d *listData) HiddenCategories(*db.User) ([]string, error) { return nil, nil }

func (d *listData) TagIDsByName(string, string) ([]int64, error) { return nil, nil }

func (d *listData) ArticleTagIDs(int64) ([]int64, error) { return nil, nil }

func (d *listData) ArticleByRef(string) (*db.Article, error) { return nil, db.ErrNotFound }

func (d *listData) UserByUsername(string) (*db.User, error) { return nil, db.ErrNotFound }

func (d *listData) UserByWikidotName(string) (*db.User, error) { return nil, db.ErrNotFound }

func (d *listData) SiteRatingMode() (string, error) { return "", db.ErrNotFound }

func (d *listData) CategoryRatingMode(string) (string, error) { return "", db.ErrNotFound }

func (d *listData) VoteStats(int64) (db.VoteStats, error) { return db.VoteStats{}, nil }

func (d *listData) ListArticles(f db.ListFilter, offset int, limit *int) ([]db.Article, error) {
	d.filter, d.offset, d.limit = f, offset, limit
	return d.listed, nil
}

func (d *listData) CountArticles(db.ListFilter, int, *int) (int, error) { return d.total, nil }

type nopVars struct{}

func (nopVars) LatestSource(int64) (string, error)        { return "", db.ErrNotFound }
func (nopVars) Authors(int64) ([]db.User, error)          { return nil, nil }
func (nopVars) LatestEditor(int64) (*db.User, error)      { return nil, db.ErrNotFound }
func (nopVars) RevisionCount(int64) (int, error)          { return 0, nil }
func (nopVars) Tags(int64) ([]string, error)              { return nil, nil }
func (nopVars) VoteStats(int64) (db.VoteStats, error)     { return db.VoteStats{}, nil }
func (nopVars) SiteRatingMode() (string, error)           { return "", db.ErrNotFound }
func (nopVars) CategoryRatingMode(string) (string, error) { return "", db.ErrNotFound }
func (nopVars) HasVoted(int64, *int64) (bool, error)      { return false, nil }
func (nopVars) ArticleByID(int64) (*db.Article, error)    { return nil, db.ErrNotFound }

func listedArticles(names ...string) []db.Article {
	out := make([]db.Article, len(names))
	for i, name := range names {
		out[i] = db.Article{
			ID: int64(i + 100), Category: db.DefaultCategory, Name: name, Title: strings.ToUpper(name),
			CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		}
	}
	return out
}

func listEnv(t *testing.T, data *listData, ctx *page.Context) (module.Env, *[]string) {
	t.Helper()
	var seen []string
	return module.Env{
		Page: ctx,
		Data: data,
		Vars: nopVars{},
		Render: func(source string, pc *page.Context) (string, error) {
			seen = append(seen, source)
			return "<" + source + ">", nil
		},
	}, &seen
}

func hostPage() *db.Article {
	return &db.Article{ID: 1, Category: db.DefaultCategory, Name: "main", Title: "Main"}
}

func TestListPagesRendersOneSourcePerRow(t *testing.T) {
	data := &listData{listed: listedArticles("a", "b"), total: 2}
	ctx := page.NewContext(hostPage(), hostPage(), nil, nil)
	env, seen := listEnv(t, data, ctx)

	got, err := module.Render(env, "listpages", map[string]string{"wrapper": "no"}, "%%name%% %%index%%/%%total%%")
	if err != nil {
		t.Fatalf("Render(listpages) err = %v, want nil", err)
	}
	want := []string{"a 1/2\n", "b 2/2\n"}
	if len(*seen) != len(want) {
		t.Fatalf("rendered sources = %q, want %q", *seen, want)
	}
	for i := range want {
		if (*seen)[i] != want[i] {
			t.Errorf("rendered source %d = %q, want %q", i, (*seen)[i], want[i])
		}
	}
	if got != "<a 1/2\n><b 2/2\n>" {
		t.Errorf("Render(listpages) = %q, want the two rows concatenated", got)
	}
}

func TestListPagesJoinsTheRowsWhenSeparateIsOff(t *testing.T) {
	data := &listData{listed: listedArticles("a", "b"), total: 2}
	ctx := page.NewContext(hostPage(), hostPage(), nil, nil)
	env, seen := listEnv(t, data, ctx)

	params := map[string]string{"wrapper": "no", "separate": "no"}
	if _, err := module.Render(env, "listpages", params, "%%name%%"); err != nil {
		t.Fatalf("Render(listpages) err = %v, want nil", err)
	}
	if len(*seen) != 1 {
		t.Fatalf("rendered sources = %q, want one pass", *seen)
	}
	if (*seen)[0] != "a\nb\n" {
		t.Errorf("rendered source = %q, want %q", (*seen)[0], "a\nb\n")
	}
}

func TestListPagesReversesTheRows(t *testing.T) {
	data := &listData{listed: listedArticles("a", "b"), total: 2}
	ctx := page.NewContext(hostPage(), hostPage(), nil, nil)
	env, seen := listEnv(t, data, ctx)

	params := map[string]string{"wrapper": "no", "reverse": "yes"}
	if _, err := module.Render(env, "listpages", params, "%%name%%"); err != nil {
		t.Fatalf("Render(listpages) err = %v, want nil", err)
	}
	if (*seen)[0] != "b\n" || (*seen)[1] != "a\n" {
		t.Errorf("rendered sources = %q, want [b a]", *seen)
	}
}

func TestListPagesIndexCountsFromThePaginationOffset(t *testing.T) {
	data := &listData{listed: listedArticles("a"), total: 45}
	ctx := page.NewContext(hostPage(), hostPage(), page.PathParams{{Key: "p", Value: "3"}}, nil)
	env, seen := listEnv(t, data, ctx)

	if _, err := module.Render(env, "listpages", map[string]string{"wrapper": "no"}, "%%index%%"); err != nil {
		t.Fatalf("Render(listpages) err = %v, want nil", err)
	}
	if (*seen)[0] != "41\n" {
		t.Errorf("rendered source = %q, want %q", (*seen)[0], "41\n")
	}
}

func TestListPagesPrependAndAppendRenderApart(t *testing.T) {
	data := &listData{listed: listedArticles("a"), total: 1}
	ctx := page.NewContext(hostPage(), hostPage(), nil, nil)
	env, seen := listEnv(t, data, ctx)

	params := map[string]string{"wrapper": "no", "prependline": "top", "appendline": "bottom"}
	if _, err := module.Render(env, "listpages", params, "%%name%%"); err != nil {
		t.Fatalf("Render(listpages) err = %v, want nil", err)
	}
	want := []string{"top\n", "a\n", "bottom"}
	if len(*seen) != 3 {
		t.Fatalf("rendered sources = %q, want %q", *seen, want)
	}
	for i := range want {
		if (*seen)[i] != want[i] {
			t.Errorf("rendered source %d = %q, want %q", i, (*seen)[i], want[i])
		}
	}
}

func TestListPagesBodySectionsOverrideTheParameters(t *testing.T) {
	data := &listData{listed: listedArticles("a"), total: 1}
	ctx := page.NewContext(hostPage(), hostPage(), nil, nil)
	env, seen := listEnv(t, data, ctx)

	params := map[string]string{"wrapper": "no", "prependline": "ignored"}
	body := "[[head]]\nXtop\n[[/head]]\n[[body]]\nX%%name%%\n[[/body]]"
	if _, err := module.Render(env, "listpages", params, body); err != nil {
		t.Fatalf("Render(listpages) err = %v, want nil", err)
	}
	if (*seen)[0] != "top\n" {
		t.Errorf("rendered source 0 = %q, want %q", (*seen)[0], "top\n")
	}
	if (*seen)[1] != "a\n" {
		t.Errorf("rendered source 1 = %q, want %q", (*seen)[1], "a\n")
	}
}

func TestListPagesWrapperCarriesTheParameters(t *testing.T) {
	data := &listData{listed: listedArticles("a"), total: 1}
	ctx := page.NewContext(hostPage(), hostPage(), page.PathParams{{Key: "tag", Value: "x"}}, nil)
	env, _ := listEnv(t, data, ctx)

	got, err := module.Render(env, "listpages", map[string]string{"category": "*"}, "%%name%%")
	if err != nil {
		t.Fatalf("Render(listpages) err = %v, want nil", err)
	}
	if !strings.Contains(got, `data-list-pages-path-params="{&quot;tag&quot;: &quot;x&quot;}"`) {
		t.Errorf("Render(listpages) = %q, want the path params in an attribute", got)
	}
	if !strings.Contains(got, `data-list-pages-params="{&quot;category&quot;: &quot;*&quot;}"`) {
		t.Errorf("Render(listpages) = %q, want the module params in an attribute", got)
	}
	if !strings.Contains(got, `data-list-pages-page-id="main"`) {
		t.Errorf("Render(listpages) = %q, want the page id in an attribute", got)
	}
}

func TestListPagesWrapperSortsTheParameterKeys(t *testing.T) {
	data := &listData{listed: nil, total: 0}
	ctx := page.NewContext(hostPage(), hostPage(), nil, nil)
	env, _ := listEnv(t, data, ctx)

	params := map[string]string{"zebra": "1", "alpha": "2", "middle": "3", "category": "*"}
	got, err := module.Render(env, "listpages", params, "")
	if err != nil {
		t.Fatalf("Render(listpages) err = %v, want nil", err)
	}
	want := `data-list-pages-params="{&quot;alpha&quot;: &quot;2&quot;, &quot;category&quot;: &quot;*&quot;,` +
		` &quot;middle&quot;: &quot;3&quot;, &quot;zebra&quot;: &quot;1&quot;}"`
	if !strings.Contains(got, want) {
		t.Errorf("Render(listpages) = %q, want %q", got, want)
	}
}

func TestListPagesUrlParameterReachesTheQuery(t *testing.T) {
	data := &listData{listed: nil, total: 0}
	ctx := page.NewContext(hostPage(), hostPage(), page.PathParams{{Key: "name", Value: "from-url"}}, nil)
	env, _ := listEnv(t, data, ctx)

	params := map[string]string{"name": "@url|fallback", "category": "*"}
	if _, err := module.Render(env, "listpages", params, ""); err != nil {
		t.Fatalf("Render(listpages) err = %v, want nil", err)
	}
	if !data.filter.HasName || data.filter.Name != "from-url" {
		t.Errorf("filter name = %q, want %q", data.filter.Name, "from-url")
	}
}

func TestListPagesLeavesThePathParametersAlone(t *testing.T) {
	data := &listData{listed: nil, total: 0}
	ctx := page.NewContext(hostPage(), hostPage(), page.PathParams{{Key: "name", Value: "from-url"}}, nil)
	env, _ := listEnv(t, data, ctx)

	if _, err := module.Render(env, "listpages", map[string]string{"category": "*"}, ""); err != nil {
		t.Fatalf("Render(listpages) err = %v, want nil", err)
	}
	if data.filter.HasName {
		t.Errorf("filter name = %q, want the path parameter ignored", data.filter.Name)
	}
}

func TestListPagesPagerAppearsOnlyWhenThereIsMoreThanOnePage(t *testing.T) {
	data := &listData{listed: listedArticles("a"), total: 45}
	ctx := page.NewContext(hostPage(), hostPage(), nil, nil)
	env, _ := listEnv(t, data, ctx)

	got, err := module.Render(env, "listpages", map[string]string{"category": "*"}, "")
	if err != nil {
		t.Fatalf("Render(listpages) err = %v, want nil", err)
	}
	if !strings.Contains(got, `<div class="pager">`) {
		t.Errorf("Render(listpages) = %q, want a pager", got)
	}
	if !strings.Contains(got, `href="/main/p/2"`) {
		t.Errorf("Render(listpages) = %q, want a link to the next page", got)
	}
}

func TestListPagesStopsRecursing(t *testing.T) {
	data := &listData{listed: listedArticles("a"), total: 1}
	ctx := page.NewContext(hostPage(), hostPage(), nil, nil)
	ctx.Depth = maxNesting
	env, _ := listEnv(t, data, ctx)

	_, err := module.Render(env, "listpages", nil, "")
	var moduleErr *module.Error
	if !asModuleError(err, &moduleErr) {
		t.Fatalf("Render(listpages) err = %v, want a module error", err)
	}
}

func asModuleError(err error, target **module.Error) bool {
	if e, ok := err.(*module.Error); ok {
		*target = e
		return true
	}
	return false
}
