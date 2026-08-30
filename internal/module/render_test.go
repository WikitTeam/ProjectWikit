package module_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	_ "github.com/WikitTeam/ProjectWikit/internal/modules"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

type fakeData struct {
	module.Data
	category db.TagCategory
	articles []db.Article
	hidden   []string
	err      error
}

func (f fakeData) TagCategory(string) (db.TagCategory, error) { return f.category, f.err }

func (f fakeData) TagArticles(string, string, []string) ([]db.Article, error) {
	return f.articles, f.err
}

func (f fakeData) HiddenCategories(*db.User) ([]string, error) { return f.hidden, f.err }

func (f fakeData) TagIDsByName(string, string) ([]int64, error) { return nil, f.err }

func (f fakeData) ArticleTagIDs(int64) ([]int64, error) { return nil, f.err }

func (f fakeData) ArticleByRef(string) (*db.Article, error) { return nil, db.ErrNotFound }

func (f fakeData) UserByUsername(string) (*db.User, error) { return nil, db.ErrNotFound }

func (f fakeData) UserByWikidotName(string) (*db.User, error) { return nil, db.ErrNotFound }

func (f fakeData) SiteRatingMode() (string, error) { return "", db.ErrNotFound }

func (f fakeData) CategoryRatingMode(string) (string, error) { return "", db.ErrNotFound }

func (f fakeData) VoteStats(int64) (db.VoteStats, error) { return db.VoteStats{}, f.err }

func (f fakeData) HasVoted(int64, *int64) (bool, error) { return false, f.err }

func (f fakeData) ListArticles(db.ListFilter, int, *int) ([]db.Article, error) {
	return f.articles, f.err
}

func (f fakeData) CountArticles(db.ListFilter, int, *int) (int, error) {
	return len(f.articles), f.err
}

func TestEveryPortedModuleAnswers(t *testing.T) {
	ported := []string{"forumcategory", "forumstart", "forumthread", "listpages", "pagedescription", "pageimage", "pagesbytag", "rate", "redirect", "search"}
	for _, name := range ported {
		if !module.Ported(name) {
			t.Errorf("Ported(%q) = false, want true", name)
		}
	}
	for _, info := range module.All() {
		want := false
		for _, name := range ported {
			if info.Name == name {
				want = true
			}
		}
		if module.Ported(info.Name) != want {
			t.Errorf("Ported(%q) = %v, want %v", info.Name, module.Ported(info.Name), want)
		}
	}
}

func TestRenderRefusesWhenTheUrlSaysSo(t *testing.T) {
	ctx := page.NewContext(nil, nil, page.PathParams{{Key: "nomodule", Value: "true"}}, nil)
	_, err := module.Render(module.Env{Page: ctx}, "redirect", nil, "")

	var moduleErr *module.Error
	if !errors.As(err, &moduleErr) {
		t.Fatalf("Render() err = %v, want a module error", err)
	}
	if moduleErr.Message != "module-disabled" {
		t.Errorf("Render() message = %q, want %q", moduleErr.Message, "module-disabled")
	}
}

func TestRenderOfAnUnknownModule(t *testing.T) {
	var moduleErr *module.Error
	if _, err := module.Render(module.Env{}, "no-such-module", nil, ""); !errors.As(err, &moduleErr) {
		t.Fatalf("Render(no-such-module) err = %v, want a module error", err)
	}
}

func TestRenderOfARemovedModule(t *testing.T) {
	var moduleErr *module.Error
	if _, err := module.Render(module.Env{}, "interwiki", nil, ""); !errors.As(err, &moduleErr) {
		t.Fatalf("Render(interwiki) err = %v, want a module error", err)
	}
}

func TestRenderOfAModuleNobodyWroteYet(t *testing.T) {
	var moduleErr *module.Error
	if _, err := module.Render(module.Env{}, "tagcloud", nil, ""); !errors.As(err, &moduleErr) {
		t.Fatalf("Render(tagcloud) err = %v, want a module error", err)
	}
	if moduleErr.Message != "module-unknown" {
		t.Errorf("Render(tagcloud) message = %q, want %q", moduleErr.Message, "module-unknown")
	}
}

func TestPathParametersGoUnderTheModulesOwn(t *testing.T) {
	ctx := page.NewContext(nil, nil, page.PathParams{{Key: "destination", Value: "/from-path"}}, nil)
	if _, err := module.Render(module.Env{Page: ctx}, "redirect", map[string]string{"destination": "/from-module"}, ""); err != nil {
		t.Fatalf("Render(redirect) err = %v, want nil", err)
	}
	if ctx.RedirectTo != "/from-module" {
		t.Errorf("RedirectTo = %q, want %q", ctx.RedirectTo, "/from-module")
	}
}

func TestPathParametersReachAModuleThatAskedForNothing(t *testing.T) {
	ctx := page.NewContext(nil, nil, page.PathParams{{Key: "destination", Value: "/from-path"}}, nil)
	if _, err := module.Render(module.Env{Page: ctx}, "redirect", nil, ""); err != nil {
		t.Fatalf("Render(redirect) err = %v, want nil", err)
	}
	if ctx.RedirectTo != "/from-path" {
		t.Errorf("RedirectTo = %q, want %q", ctx.RedirectTo, "/from-path")
	}
}

func TestRedirectSetsTheTarget(t *testing.T) {
	ctx := page.NewContext(nil, nil, nil, nil)
	if _, err := module.Render(module.Env{Page: ctx}, "redirect", map[string]string{"destination": "/main"}, ""); err != nil {
		t.Fatalf("Render(redirect) err = %v, want nil", err)
	}
	if ctx.RedirectTo != "/main" {
		t.Errorf("RedirectTo = %q, want %q", ctx.RedirectTo, "/main")
	}
}

func TestRedirectHonoursNoredirect(t *testing.T) {
	ctx := page.NewContext(nil, nil, nil, nil)
	params := map[string]string{"destination": "/main", "noredirect": "true"}
	if _, err := module.Render(module.Env{Page: ctx}, "redirect", params, ""); err != nil {
		t.Fatalf("Render(redirect) err = %v, want nil", err)
	}
	if ctx.RedirectTo != "" {
		t.Errorf("RedirectTo = %q, want %q", ctx.RedirectTo, "")
	}
}

func TestRedirectRefusesARunnableScheme(t *testing.T) {
	for _, target := range []string{"javascript:alert(1)", "JavaScript:alert(1)", "data:text/html,x"} {
		ctx := page.NewContext(nil, nil, nil, nil)
		_, err := module.Render(module.Env{Page: ctx}, "redirect", map[string]string{"destination": target}, "")

		var moduleErr *module.Error
		if !errors.As(err, &moduleErr) {
			t.Errorf("Render(redirect, %q) err = %v, want a module error", target, err)
		}
		if ctx.RedirectTo != "" {
			t.Errorf("RedirectTo after %q = %q, want %q", target, ctx.RedirectTo, "")
		}
	}
}

func TestPageDescriptionOfEmptyBody(t *testing.T) {
	ctx := page.NewContext(nil, nil, nil, nil)
	if _, err := module.Render(module.Env{Page: ctx}, "pagedescription", nil, ""); err != nil {
		t.Fatalf("Render(pagedescription) err = %v, want nil", err)
	}
	if ctx.OGDescription != "" {
		t.Errorf("OGDescription = %q, want %q", ctx.OGDescription, "")
	}
}

func TestPageDescriptionEscapesTheAngleBracket(t *testing.T) {
	ctx := page.NewContext(nil, nil, nil, nil)
	if _, err := module.Render(module.Env{Page: ctx}, "pagedescription", nil, "a <b> c"); err != nil {
		t.Fatalf("Render(pagedescription) err = %v, want nil", err)
	}
	if !strings.HasPrefix(ctx.OGDescription, "a ") || !strings.HasSuffix(ctx.OGDescription, "b> c") {
		t.Errorf("OGDescription = %q, want the angle bracket replaced", ctx.OGDescription)
	}
	if strings.Contains(ctx.OGDescription, "<b>") {
		t.Errorf("OGDescription = %q, want no raw angle bracket", ctx.OGDescription)
	}
}

func TestPageImageResolvesTheSource(t *testing.T) {
	cases := []struct{ src, want string }{
		{"probe:full/cover.png", "https://media.example/local--files/probe:full/cover.png"},
		{"/probe:full/cover.png", "https://media.example/local--files/probe:full/cover.png"},
		{"https://example.org/x.png", "https://example.org/x.png"},
		{"cover.png", "cover.png"},
	}
	for _, c := range cases {
		ctx := page.NewContext(nil, nil, nil, nil)
		env := module.Env{Page: ctx, Site: &db.Site{MediaDomain: "media.example"}}
		if _, err := module.Render(env, "pageimage", map[string]string{"src": c.src}, ""); err != nil {
			t.Fatalf("Render(pageimage, %q) err = %v, want nil", c.src, err)
		}
		if ctx.OGImage != c.want {
			t.Errorf("OGImage after %q = %q, want %q", c.src, ctx.OGImage, c.want)
		}
	}
}

func TestPageImageWithoutASource(t *testing.T) {
	ctx := page.NewContext(nil, nil, nil, nil)
	env := module.Env{Page: ctx, Site: &db.Site{MediaDomain: "media.example"}}
	if _, err := module.Render(env, "pageimage", nil, ""); err != nil {
		t.Fatalf("Render(pageimage) err = %v, want nil", err)
	}
	if ctx.OGImage != "" {
		t.Errorf("OGImage = %q, want %q", ctx.OGImage, "")
	}
}

func TestPagesByTagWithoutATag(t *testing.T) {
	got, err := module.Render(module.Env{Data: fakeData{}}, "pagesbytag", nil, "")
	if err != nil {
		t.Fatalf("Render(pagesbytag) err = %v, want nil", err)
	}
	if got != "" {
		t.Errorf("Render(pagesbytag) = %q, want %q", got, "")
	}
}

func TestPagesByTagListsTheArticles(t *testing.T) {
	data := fakeData{
		category: db.TagCategory{Name: "lang"},
		articles: []db.Article{
			{Category: "_default", Name: "one", Title: "One"},
			{Category: "probe", Name: "two"},
		},
	}
	got, err := module.Render(module.Env{Data: data}, "pagesbytag", map[string]string{"tag": "lang:en"}, "")
	if err != nil {
		t.Fatalf("Render(pagesbytag) err = %v, want nil", err)
	}
	for _, want := range []string{`<a href="/one">One</a>`, `<a href="/probe:two">probe:two</a>`, `id="tagged-pages-list"`} {
		if !strings.Contains(got, want) {
			t.Errorf("Render(pagesbytag) = %q, want it to carry %q", got, want)
		}
	}
}

func TestPagesByTagOfTheDefaultCategory(t *testing.T) {
	data := fakeData{category: db.TagCategory{Name: "默认"}}
	got, err := module.Render(module.Env{Data: data}, "pagesbytag", map[string]string{"tag": "scp"}, "")
	if err != nil {
		t.Fatalf("Render(pagesbytag) err = %v, want nil", err)
	}
	if strings.Contains(got, "module-pagesbytag-category") {
		t.Errorf("Render(pagesbytag) = %q, want no category clause", got)
	}
}
