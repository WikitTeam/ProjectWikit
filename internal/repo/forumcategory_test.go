package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/callbacks"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

const (
	forumCategoryGolden = "testdata/forumcategory.golden"
	forumCategoryCorpus = "testdata/forumcategory_corpus.json"
)

type forumCategoryCase struct {
	Name     string            `json:"name"`
	Viewer   string            `json:"viewer"`
	Category string            `json:"category,omitempty"`
	Path     map[string]string `json:"path,omitempty"`
}

func forumCategoryCases() []forumCategoryCase {
	return []forumCategoryCase{
		{Name: "chat", Viewer: "", Category: "Probe Chat"},
		{Name: "chat-by-start", Viewer: "", Category: "Probe Chat", Path: map[string]string{"sort": "start"}},
		{Name: "chat-by-junk-sort", Viewer: "", Category: "Probe Chat", Path: map[string]string{"sort": "reply"}},
		{Name: "chat-signed-in", Viewer: "probe-author", Category: "Probe Chat"},
		{Name: "empty", Viewer: "", Category: "Probe Quiet"},
		{Name: "comments", Viewer: "", Category: "Probe Comments"},
		{Name: "busy-first-page", Viewer: "", Category: "Probe Busy"},
		{Name: "busy-second-page", Viewer: "", Category: "Probe Busy", Path: map[string]string{"p": "2"}},
		{Name: "busy-past-the-end", Viewer: "", Category: "Probe Busy", Path: map[string]string{"p": "99"}},
		{Name: "busy-below-one", Viewer: "", Category: "Probe Busy", Path: map[string]string{"p": "-3"}},
		{Name: "busy-junk-page", Viewer: "", Category: "Probe Busy", Path: map[string]string{"p": "later"}},
		{Name: "hidden-section-category", Viewer: "", Category: "Probe Hidden Chat"},
		{Name: "missing", Viewer: "", Path: map[string]string{"c": "999999"}},
		{Name: "unparsable", Viewer: "", Path: map[string]string{"c": "not-a-number"}},
		{Name: "padded-id", Viewer: "", Path: map[string]string{"c": "0999999"}},
		{Name: "no-parameter", Viewer: ""},
	}
}

func TestForumCategoryMatchesGolden(t *testing.T) {
	ctx := context.Background()
	d := testDB(t)
	users, loc := testUsers(t)
	cases := forumCategoryCases()

	article, err := d.ArticleByName(ctx, "forum:category")
	if err != nil {
		t.Fatalf("ArticleByName(forum:category) err = %v, want nil", err)
	}
	categories, err := d.ForumCategories(ctx)
	if err != nil {
		t.Fatalf("ForumCategories() err = %v, want nil", err)
	}

	var b strings.Builder
	for _, c := range cases {
		var viewer *db.User
		if c.Viewer != "" {
			viewer, err = d.UserByUsername(ctx, c.Viewer)
			if err != nil {
				t.Fatalf("UserByUsername(%q) err = %v, want nil", c.Viewer, err)
			}
		}

		path := forumCategoryPath(t, c, categories)
		pc := page.NewContext(article, article, path, viewer)
		r := New(ctx, d, users, Options{Loc: loc, User: viewer})

		body, err := r.RenderModule(pc, "ForumCategory", nil, "")
		var moduleErr *callbacks.ModuleError
		switch {
		case errors.As(err, &moduleErr):
			body = "!error: " + moduleErr.Message
		case err != nil:
			t.Fatalf("RenderModule(%s) err = %v, want nil", c.Name, err)
		}
		fmt.Fprintf(&b, "=== %s\ntitle: %s\nstatus: %d\n%s\n", c.Name, pc.Title, pc.Status, body)
	}
	compareGolden(t, b.String(), cases, forumCategoryGolden, forumCategoryCorpus)
}

func forumCategoryPath(t *testing.T, c forumCategoryCase, categories []db.ForumCategory) page.PathParams {
	t.Helper()
	path := sortedPath(c.Path)
	if c.Category == "" {
		return path
	}
	for _, category := range categories {
		if category.Name == c.Category {
			return path.Put(page.PathParam{Key: "c", Value: fmt.Sprint(category.ID)})
		}
	}
	t.Fatalf("ForumCategories() has no category named %q, want one seeded", c.Category)
	return nil
}
