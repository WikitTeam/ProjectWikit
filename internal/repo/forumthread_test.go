package repo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/WikitTeam/ProjectWikit/internal/callbacks"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/printuser"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
	"github.com/WikitTeam/ProjectWikit/internal/renderer/sidecar"
)

const (
	forumThreadGolden = "testdata/forumthread.golden"
	forumThreadCorpus = "testdata/forumthread_corpus.json"
)

type forumThreadCase struct {
	Name    string            `json:"name"`
	Viewer  string            `json:"viewer"`
	Thread  string            `json:"thread,omitempty"`
	Article string            `json:"article,omitempty"`
	Post    string            `json:"post,omitempty"`
	PostIn  string            `json:"postIn,omitempty"`
	Params  map[string]string `json:"params,omitempty"`
	Path    map[string]string `json:"path,omitempty"`
}

func forumThreadCases() []forumThreadCase {
	return []forumThreadCase{
		{Name: "deep", Viewer: "", Thread: "Probe Deep Thread"},
		{Name: "deep-signed-in", Viewer: "probe-author", Thread: "Probe Deep Thread"},
		{Name: "deep-content-only", Viewer: "", Thread: "Probe Deep Thread", Params: map[string]string{"contentonly": "yes"}},
		{Name: "locked", Viewer: "probevoter", Thread: "Probe Locked Thread"},
		{Name: "long-first-page", Viewer: "", Thread: "Probe Long Thread"},
		{Name: "long-second-page", Viewer: "", Thread: "Probe Long Thread", Path: map[string]string{"p": "2"}},
		{Name: "long-past-the-end", Viewer: "", Thread: "Probe Long Thread", Path: map[string]string{"p": "99"}},
		{Name: "long-below-one", Viewer: "", Thread: "Probe Long Thread", Path: map[string]string{"p": "-3"}},
		{Name: "long-junk-page", Viewer: "", Thread: "Probe Long Thread", Path: map[string]string{"p": "later"}},
		{Name: "by-post-on-second-page", Viewer: "", Thread: "Probe Long Thread", Post: "Probe Long Thread post 10"},
		{Name: "by-nested-reply", Viewer: "", Thread: "Probe Deep Thread", Post: "Probe Deep reply 0 0"},
		{Name: "by-post-in-another-thread", Viewer: "", Thread: "Probe Long Thread", Post: "Probe Thread post 0", PostIn: "Probe Thread"},
		{Name: "by-missing-post", Viewer: "", Thread: "Probe Long Thread", Path: map[string]string{"post": "999999"}},
		{Name: "by-junk-post", Viewer: "", Thread: "Probe Long Thread", Path: map[string]string{"post": "later"}},
		{Name: "comments", Viewer: "", Article: "probe:full"},
		{Name: "comments-signed-in", Viewer: "probevoter", Article: "probe:full"},
		{Name: "comments-stars", Viewer: "", Article: "probestars:rated"},
		{Name: "missing", Viewer: "", Path: map[string]string{"t": "999999"}},
		{Name: "unparsable", Viewer: "", Path: map[string]string{"t": "not-a-number"}},
		{Name: "padded-id", Viewer: "", Path: map[string]string{"t": "0999999"}},
		{Name: "no-parameter", Viewer: ""},
	}
}

func TestForumThreadMatchesGolden(t *testing.T) {
	bin := os.Getenv(sidecar.EnvBinary)
	if bin == "" {
		t.Skipf("%s not set, skipping the thread test", sidecar.EnvBinary)
	}
	ctx := context.Background()
	d := testDB(t)
	users, loc := testUsers(t)
	cases := forumThreadCases()

	rend, err := sidecar.New(bin)
	if err != nil {
		t.Fatalf("sidecar.New(%q) err = %v, want nil", bin, err)
	}
	t.Cleanup(func() { rend.Close() })

	site, err := d.SiteByHosts(ctx, []string{"localhost"})
	if err != nil {
		t.Fatalf("SiteByHosts(localhost) err = %v, want nil", err)
	}
	article, err := d.ArticleByName(ctx, "forum:thread")
	if err != nil {
		t.Fatalf("ArticleByName(forum:thread) err = %v, want nil", err)
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

		path := forumThreadPath(t, ctx, d, c)
		pc := page.NewContext(article, article, path, viewer)
		r := newThreadRepo(ctx, d, users, loc, site, viewer, rend)

		body, err := r.RenderModule(pc, "ForumThread", c.Params, "")
		var moduleErr *callbacks.ModuleError
		switch {
		case errors.As(err, &moduleErr):
			body = "!error: " + moduleErr.Message
		case err != nil:
			t.Fatalf("RenderModule(%s) err = %v, want nil", c.Name, err)
		}
		fmt.Fprintf(&b, "=== %s\ntitle: %s\nstatus: %d\n%s\n", c.Name, pc.Title, pc.Status, body)
	}
	compareGolden(t, b.String(), cases, forumThreadGolden, forumThreadCorpus)
}

func newThreadRepo(ctx context.Context, d *db.DB, users *printuser.Renderer, loc *i18n.Localizer,
	site *db.Site, viewer *db.User, rend renderer.Renderer) *Repository {

	var r *Repository
	render := func(source string) (string, error) {
		cb := callbacks.New(loc, r)
		cb.SetContext(page.NewContext(nil, nil, nil, viewer))
		info := renderer.PageInfo{Site: site.Slug, Domain: site.Domain, MediaDomain: site.MediaDomain}
		html, err := rend.RenderHTML(ctx, source, info, cb, renderer.ModeMessage)
		if err != nil {
			return "", err
		}
		return html.Body, nil
	}
	r = New(ctx, d, users, Options{Loc: loc, Site: site, User: viewer, RenderMessage: render})
	return r
}

func forumThreadPath(t *testing.T, ctx context.Context, d *db.DB, c forumThreadCase) page.PathParams {
	t.Helper()
	path := sortedPath(c.Path)

	switch {
	case c.Thread != "":
		path = path.Put(page.PathParam{Key: "t", Value: fmt.Sprint(threadIDByName(t, ctx, d, c.Thread))})
	case c.Article != "":
		found, err := d.ArticleByName(ctx, c.Article)
		if err != nil {
			t.Fatalf("ArticleByName(%q) err = %v, want nil", c.Article, err)
		}
		info, err := d.CommentInfo(ctx, found.ID)
		if err != nil {
			t.Fatalf("CommentInfo(%q) err = %v, want nil", c.Article, err)
		}
		path = path.Put(page.PathParam{Key: "t", Value: fmt.Sprint(info.ThreadID)})
	}
	if c.Post != "" {
		in := c.PostIn
		if in == "" {
			in = c.Thread
		}
		id := postIDByName(t, ctx, d, threadIDByName(t, ctx, d, in), c.Post)
		path = path.Put(page.PathParam{Key: "post", Value: fmt.Sprint(id)})
	}
	return path
}

func threadIDByName(t *testing.T, ctx context.Context, d *db.DB, name string) int64 {
	t.Helper()
	categories, err := d.ForumCategories(ctx)
	if err != nil {
		t.Fatalf("ForumCategories() err = %v, want nil", err)
	}
	for _, category := range categories {
		threads, err := d.ForumThreads(ctx, category.ID, db.ForumThreadsByStart, 0, 1000)
		if err != nil {
			t.Fatalf("ForumThreads(%d) err = %v, want nil", category.ID, err)
		}
		for _, thread := range threads {
			if thread.Name == name {
				return thread.ID
			}
		}
	}
	t.Fatalf("ForumThreads() has no thread named %q, want one seeded", name)
	return 0
}

func postIDByName(t *testing.T, ctx context.Context, d *db.DB, threadID int64, name string) int64 {
	t.Helper()
	roots, err := d.ForumRootPosts(ctx, threadID, 0, 1000)
	if err != nil {
		t.Fatalf("ForumRootPosts(%d) err = %v, want nil", threadID, err)
	}
	if id, ok := findPost(t, ctx, d, roots, name); ok {
		return id
	}
	t.Fatalf("ForumRootPosts(%d) has no post named %q, want one seeded", threadID, name)
	return 0
}

func findPost(t *testing.T, ctx context.Context, d *db.DB, posts []db.ForumThreadPost, name string) (int64, bool) {
	t.Helper()
	for _, post := range posts {
		if post.Name == name {
			return post.ID, true
		}
		replies, err := d.ForumPostReplies(ctx, post.ID)
		if err != nil {
			t.Fatalf("ForumPostReplies(%d) err = %v, want nil", post.ID, err)
		}
		if id, ok := findPost(t, ctx, d, replies, name); ok {
			return id, true
		}
	}
	return 0, false
}
