package perms

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden file and export the corpus")

type roleSpec struct {
	Slug         string   `json:"slug"`
	Permissions  []string `json:"permissions"`
	Restrictions []string `json:"restrictions"`
}

type overrideSpec struct {
	Role         string   `json:"role"`
	Permissions  []string `json:"permissions"`
	Restrictions []string `json:"restrictions"`
}

type objectSpec struct {
	Kind           string         `json:"kind"`
	Overrides      []overrideSpec `json:"overrides"`
	Locked         bool           `json:"locked"`
	Author         bool           `json:"author"`
	HiddenForUsers bool           `json:"hidden_for_users"`
	ThreadAuthor   bool           `json:"thread_author"`
}

type userSpec struct {
	Kind        string   `json:"kind"`
	Roles       []string `json:"roles"`
	ForumBarred bool     `json:"forum_barred"`
}

type scenario struct {
	Name   string      `json:"name"`
	Roles  []roleSpec  `json:"roles"`
	User   userSpec    `json:"user"`
	Object *objectSpec `json:"object"`
}

var queried = []string{
	ViewArticles, RateArticles, CreateArticles, EditArticles, TagArticles,
	MoveArticles, LockArticles, ManageArticleFiles, DeleteArticles,
	ResetArticleVotes, CommentArticles, ViewArticleComments, ManageArticleAuthors,
	ViewForumPosts, CreateForumPosts, EditForumPosts, DeleteForumPosts,
	ViewForumThreads, CreateForumThreads, EditForumThreads, PinForumThreads,
	LockForumThreads, MoveForumThreads, ViewForumSections,
	ViewHiddenForumSections, ViewForumCategories,
}

var baseRoles = []roleSpec{
	{Slug: "everyone", Permissions: []string{ViewArticles, ViewArticleComments, ViewForumSections, ViewForumCategories, ViewForumThreads, ViewForumPosts}},
	{Slug: "registered", Permissions: []string{CommentArticles, RateArticles, CreateForumThreads, CreateForumPosts}},
	{Slug: "editor", Permissions: []string{EditArticles, TagArticles, MoveArticles}},
}

func withRoles(extra ...roleSpec) []roleSpec {
	out := append([]roleSpec{}, baseRoles...)
	return append(out, extra...)
}

var scenarios = []scenario{
	{
		Name:  "anonymous without object",
		Roles: baseRoles,
		User:  userSpec{Kind: "anonymous"},
	},
	{
		Name:   "anonymous on an article",
		Roles:  baseRoles,
		User:   userSpec{Kind: "anonymous"},
		Object: &objectSpec{Kind: "article"},
	},
	{
		Name:   "registered user takes the union of three roles",
		Roles:  baseRoles,
		User:   userSpec{Kind: "normal", Roles: []string{"editor"}},
		Object: &objectSpec{Kind: "article"},
	},
	{
		Name:   "a restriction beats a permission inside one role",
		Roles:  withRoles(roleSpec{Slug: "half", Permissions: []string{DeleteArticles, LockArticles}, Restrictions: []string{DeleteArticles}}),
		User:   userSpec{Kind: "normal", Roles: []string{"half"}},
		Object: &objectSpec{Kind: "article"},
	},
	{
		Name:   "a restriction in one role does not take away what another grants",
		Roles:  withRoles(roleSpec{Slug: "half", Restrictions: []string{ViewArticles}}),
		User:   userSpec{Kind: "normal", Roles: []string{"half"}},
		Object: &objectSpec{Kind: "article"},
	},
	{
		Name:   "a category override grants to one role",
		Roles:  baseRoles,
		User:   userSpec{Kind: "normal", Roles: []string{"editor"}},
		Object: &objectSpec{Kind: "article", Overrides: []overrideSpec{{Role: "editor", Permissions: []string{DeleteArticles}}}},
	},
	{
		Name:   "a category override restricts the role that granted it",
		Roles:  baseRoles,
		User:   userSpec{Kind: "normal", Roles: []string{"editor"}},
		Object: &objectSpec{Kind: "article", Overrides: []overrideSpec{{Role: "everyone", Restrictions: []string{ViewArticles}}}},
	},
	{
		Name:   "a category override restricting one role leaves the others",
		Roles:  withRoles(roleSpec{Slug: "viewer", Permissions: []string{ViewArticles}}),
		User:   userSpec{Kind: "normal", Roles: []string{"viewer"}},
		Object: &objectSpec{Kind: "article", Overrides: []overrideSpec{{Role: "everyone", Restrictions: []string{ViewArticles}}}},
	},
	{
		Name:   "the override reaches a category asked about directly",
		Roles:  baseRoles,
		User:   userSpec{Kind: "normal", Roles: []string{"editor"}},
		Object: &objectSpec{Kind: "category", Overrides: []overrideSpec{{Role: "editor", Permissions: []string{DeleteArticles}}}},
	},
	{
		Name:   "a locked article takes six permissions from whoever cannot unlock",
		Roles:  baseRoles,
		User:   userSpec{Kind: "normal", Roles: []string{"editor"}},
		Object: &objectSpec{Kind: "article", Locked: true},
	},
	{
		Name:   "a locked article keeps everything for whoever can unlock",
		Roles:  withRoles(roleSpec{Slug: "locker", Permissions: []string{LockArticles}}),
		User:   userSpec{Kind: "normal", Roles: []string{"editor", "locker"}},
		Object: &objectSpec{Kind: "article", Locked: true},
	},
	{
		Name:   "an author gains authorship management",
		Roles:  baseRoles,
		User:   userSpec{Kind: "normal", Roles: []string{"editor"}},
		Object: &objectSpec{Kind: "article", Author: true},
	},
	{
		Name:   "a lock beats authorship",
		Roles:  baseRoles,
		User:   userSpec{Kind: "normal", Roles: []string{"editor"}},
		Object: &objectSpec{Kind: "article", Locked: true, Author: true},
	},
	{
		Name:   "an inactive user gets nothing",
		Roles:  baseRoles,
		User:   userSpec{Kind: "inactive", Roles: []string{"editor"}},
		Object: &objectSpec{Kind: "article"},
	},
	{
		Name:   "a superuser gets everything",
		Roles:  baseRoles,
		User:   userSpec{Kind: "superuser"},
		Object: &objectSpec{Kind: "article", Locked: true},
	},
	{
		Name:   "an inactive superuser gets nothing",
		Roles:  baseRoles,
		User:   userSpec{Kind: "inactive_superuser"},
		Object: &objectSpec{Kind: "article"},
	},
	{
		Name:   "a plain forum section",
		Roles:  baseRoles,
		User:   userSpec{Kind: "normal"},
		Object: &objectSpec{Kind: "forum_section"},
	},
	{
		Name:   "a section hidden for users takes browsing away",
		Roles:  baseRoles,
		User:   userSpec{Kind: "normal"},
		Object: &objectSpec{Kind: "forum_section", HiddenForUsers: true},
	},
	{
		Name:   "a hidden section keeps browsing for whoever may see hidden ones",
		Roles:  withRoles(roleSpec{Slug: "moderator", Permissions: []string{ViewHiddenForumSections}}),
		User:   userSpec{Kind: "normal", Roles: []string{"moderator"}},
		Object: &objectSpec{Kind: "forum_section", HiddenForUsers: true},
	},
	{
		Name:   "a hidden section takes nothing from someone who could not browse anyway",
		Roles:  []roleSpec{{Slug: "everyone"}, {Slug: "registered"}},
		User:   userSpec{Kind: "normal"},
		Object: &objectSpec{Kind: "forum_section", HiddenForUsers: true},
	},
	{
		Name:   "a forum category answers as if there were no object",
		Roles:  withRoles(roleSpec{Slug: "moderator", Permissions: []string{ViewHiddenForumSections}}),
		User:   userSpec{Kind: "normal", Roles: []string{"moderator"}},
		Object: &objectSpec{Kind: "forum_category"},
	},
	{
		Name:   "a plain thread",
		Roles:  baseRoles,
		User:   userSpec{Kind: "normal"},
		Object: &objectSpec{Kind: "forum_thread"},
	},
	{
		Name:   "the author of a thread may edit it",
		Roles:  baseRoles,
		User:   userSpec{Kind: "normal"},
		Object: &objectSpec{Kind: "forum_thread", Author: true},
	},
	{
		Name:   "a locked thread silences whoever cannot unlock",
		Roles:  baseRoles,
		User:   userSpec{Kind: "normal"},
		Object: &objectSpec{Kind: "forum_thread", Locked: true},
	},
	{
		Name:   "a locked thread silences its own author too",
		Roles:  baseRoles,
		User:   userSpec{Kind: "normal"},
		Object: &objectSpec{Kind: "forum_thread", Locked: true, Author: true},
	},
	{
		Name:   "a locked thread leaves everything to whoever may unlock",
		Roles:  withRoles(roleSpec{Slug: "moderator", Permissions: []string{LockForumThreads}}),
		User:   userSpec{Kind: "normal", Roles: []string{"moderator"}},
		Object: &objectSpec{Kind: "forum_thread", Locked: true},
	},
	{
		Name:   "a user barred from the forum is silenced on an open thread",
		Roles:  baseRoles,
		User:   userSpec{Kind: "normal", ForumBarred: true},
		Object: &objectSpec{Kind: "forum_thread"},
	},
	{
		Name:   "an anonymous visitor is never barred from the forum",
		Roles:  baseRoles,
		User:   userSpec{Kind: "anonymous"},
		Object: &objectSpec{Kind: "forum_thread"},
	},
	{
		Name:   "an anonymous visitor keeps what everyone grants on a thread",
		Roles:  []roleSpec{{Slug: "everyone", Permissions: []string{ViewForumThreads, CommentArticles, CreateForumPosts}}, {Slug: "registered"}},
		User:   userSpec{Kind: "anonymous"},
		Object: &objectSpec{Kind: "forum_thread"},
	},
	{
		Name:   "a plain post",
		Roles:  baseRoles,
		User:   userSpec{Kind: "normal"},
		Object: &objectSpec{Kind: "forum_post"},
	},
	{
		Name:   "the author of a post may edit it",
		Roles:  baseRoles,
		User:   userSpec{Kind: "normal"},
		Object: &objectSpec{Kind: "forum_post", Author: true},
	},
	{
		Name:   "the author of a post who may not write posts gains nothing",
		Roles:  []roleSpec{{Slug: "everyone", Permissions: []string{ViewForumPosts}}, {Slug: "registered"}},
		User:   userSpec{Kind: "normal"},
		Object: &objectSpec{Kind: "forum_post", Author: true},
	},
	{
		Name:   "a locked thread silences the author of a post under it",
		Roles:  baseRoles,
		User:   userSpec{Kind: "normal"},
		Object: &objectSpec{Kind: "forum_post", Author: true, Locked: true},
	},
	{
		Name:   "the author of a thread keeps editing it while reading a post under it",
		Roles:  baseRoles,
		User:   userSpec{Kind: "normal"},
		Object: &objectSpec{Kind: "forum_post", ThreadAuthor: true},
	},
	{
		Name:   "a superuser is not silenced by a locked thread",
		Roles:  baseRoles,
		User:   userSpec{Kind: "superuser"},
		Object: &objectSpec{Kind: "forum_thread", Locked: true},
	},
}

func (s scenario) roleID(slug string) int64 {
	for i, role := range s.Roles {
		if role.Slug == slug {
			return int64(i + 1)
		}
	}
	return 0
}

func (s scenario) role(slug string) Role {
	for i, spec := range s.Roles {
		if spec.Slug == slug {
			return Role{ID: int64(i + 1), Permissions: spec.Permissions, Restrictions: spec.Restrictions}
		}
	}
	return Role{}
}

func (s scenario) subject() Subject {
	sub := Subject{
		Anonymous:   s.User.Kind == "anonymous",
		Active:      s.User.Kind != "inactive" && s.User.Kind != "inactive_superuser",
		ForumActive: s.User.Kind != "anonymous" && !s.User.ForumBarred,
		Superuser:   s.User.Kind == "superuser" || s.User.Kind == "inactive_superuser",
	}
	sub.Roles = append(sub.Roles, s.role("everyone"))
	if !sub.Anonymous {
		sub.Roles = append(sub.Roles, s.role("registered"))
		for _, slug := range s.User.Roles {
			sub.Roles = append(sub.Roles, s.role(slug))
		}
	}
	return sub
}

func (s scenario) object() *Object {
	if s.Object == nil || s.Object.Kind == "forum_category" {
		return nil
	}
	o := &Object{
		Kind:           objectKinds[s.Object.Kind],
		Locked:         s.Object.Kind != "category" && s.Object.Locked,
		Author:         s.Object.Kind != "category" && s.Object.Author,
		HiddenForUsers: s.Object.HiddenForUsers,
	}
	if o.Kind == KindForumPost {
		o.Thread = &Object{Kind: KindForumThread, Locked: s.Object.Locked, Author: s.Object.ThreadAuthor}
	}
	for _, spec := range s.Object.Overrides {
		o.Overrides = append(o.Overrides, Override{RoleID: s.roleID(spec.Role), Permissions: spec.Permissions, Restrictions: spec.Restrictions})
	}
	return o
}

var objectKinds = map[string]Kind{
	"article":       KindArticle,
	"category":      KindArticle,
	"forum_section": KindForumSection,
	"forum_thread":  KindForumThread,
	"forum_post":    KindForumPost,
}

func TestResolveMatchesGolden(t *testing.T) {
	var b strings.Builder
	for _, s := range scenarios {
		set := Resolve(s.subject(), s.object())
		fmt.Fprintf(&b, "=== %s\n", s.Name)
		for _, name := range queried {
			fmt.Fprintf(&b, "%s = %t\n", name, set.Has(name))
		}
	}

	path := filepath.Join("testdata", "perms.golden")
	if *update {
		if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) = %v, want nil", path, err)
		}
		raw, err := json.MarshalIndent(map[string]any{"queried": queried, "scenarios": scenarios}, "", "  ")
		if err != nil {
			t.Fatalf("MarshalIndent() = %v, want nil", err)
		}
		if err := os.WriteFile(filepath.Join("testdata", "perms_corpus.json"), append(raw, '\n'), 0o644); err != nil {
			t.Fatalf("WriteFile() = %v, want nil", err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) = %v, want nil", path, err)
	}
	if b.String() == string(want) {
		return
	}
	got, wantLines := strings.Split(b.String(), "\n"), strings.Split(string(want), "\n")
	for i := 0; i < len(got) && i < len(wantLines); i++ {
		if got[i] != wantLines[i] {
			t.Fatalf("perms.golden line %d: got %q, want %q", i+1, got[i], wantLines[i])
		}
	}
	t.Fatalf("perms.golden line count: got %d, want %d", len(got), len(wantLines))
}
