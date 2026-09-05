// Package module is what a wikidot module is written against. The modules
// themselves live in internal/modules, one file each.
package module

import (
	"slices"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/form"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/roles"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

// Error is the failure that reaches the reader as a block on the page rather
// than as a broken response.
type Error struct{ Message string }

func (e *Error) Error() string { return e.Message }

type Data interface {
	TagArticles(categorySlug, name string, hiddenCategories []string) ([]db.Article, error)
	TagCategory(slug string) (db.TagCategory, error)
	HiddenCategories(user *db.User) ([]string, error)
	TagIDsByName(categorySlug, name string) ([]int64, error)
	TagCloud(limit *int) ([]db.CloudTag, error)
	WantedLinks(f db.WantedFilter, offset, limit int) ([]db.WantedLink, error)
	WantedLinkCount(f db.WantedFilter) (int, error)
	ArticleTagIDs(articleID int64) ([]int64, error)
	ArticleByRef(ref string) (*db.Article, error)
	UserByUsername(name string) (*db.User, error)
	UserByWikidotName(name string) (*db.User, error)
	SiteRatingMode() (string, error)
	CategoryRatingMode(category string) (string, error)
	VoteStats(articleID int64) (db.VoteStats, error)
	HasVoted(articleID int64, userID *int64) (bool, error)
	ListArticles(f db.ListFilter, offset int, limit *int) ([]db.Article, error)
	CountArticles(f db.ListFilter, offset int, limit *int) (int, error)
	LatestSource(articleID int64) (string, error)
	CategoryForm(category string) (*form.Definition, error)

	ArticleByID(id int64) (*db.Article, error)
	ArticleAuthors(articleID int64) ([]db.User, error)
	ArticleFiles(articleID int64) ([]db.ArticleFileEntry, error)
	CommentInfo(articleID int64) (db.CommentInfo, error)
	RenderUserByID(id *int64) (string, error)
	RenderUser(u db.User) (string, error)

	ArticleVotes(articleID int64) ([]db.ArticleVote, error)
	ReplaceVote(articleID, userID int64, rate *float64, roleID *int64) (*db.Vote, error)
	VoteGroupRole(userID *int64) (*int64, error)
	AddActionLog(user *db.User, kind, meta string) error
	ArticleObject(article *db.Article, viewer *db.User) (*perms.Object, error)
	UserJSON(u *db.User) (wikijson.Object, error)

	SearchArticles(f db.SearchFilter, offset, limit int) ([]db.SearchHit, error)
	SearchCount(f db.SearchFilter) (int, error)
	TagIDsByFullName(category, name string) ([]int64, error)
	AuthorsOfArticles(ids []int64) (map[int64][]db.User, error)
	VoteStatsOfArticles(ids []int64) (map[int64]db.VoteStats, error)
	CommentCountsOfArticles(ids []int64) (map[int64]int, error)
	TagsOfArticles(ids []int64) (map[int64][]db.ArticleTag, error)
	UserByDisplayName(name string) (*db.User, error)

	SiteChanges(f db.SiteChangeFilter, offset, limit int) ([]db.SiteChange, error)
	SiteChangeCount(f db.SiteChangeFilter) (int, error)
	ArticleCategories(hidden []string) ([]string, error)
	UserIDsByName(name string, partial bool) ([]int64, error)
	UsersByIDs(ids []int64) ([]db.User, error)

	ForumSections() ([]db.ForumSection, error)
	ForumSection(id int64) (*db.ForumSection, error)
	ForumCategories() ([]db.ForumCategory, error)
	ForumCategory(id int64) (*db.ForumCategory, error)
	ForumThreads(categoryID int64, sort db.ForumThreadSort, offset, limit int) ([]db.ForumThread, error)
	ForumCommentThreads(sort db.ForumThreadSort, offset, limit int) ([]db.ForumThread, error)
	ForumThreadPostCount(threadID int64) (int, error)
	ForumThreadLastPost(threadID int64, count int) (*db.ForumPost, error)
	ForumCategoryCounts(categoryID int64) (db.ForumCounts, error)
	ForumCommentCounts() (db.ForumCounts, error)
	ForumCategoryLastPost(categoryID int64) (*db.ForumLastPost, error)
	ForumCommentLastPost() (*db.ForumLastPost, error)

	ForumThreadsInCategories(categoryIDs []int64, offset, limit int) ([]db.ForumThread, error)
	ForumFirstPosts(threadIDs []int64) (map[int64]db.ForumThreadPost, error)
	ForumThreadPostCounts(threadIDs []int64) (map[int64]int, error)

	ForumThread(id int64) (*db.ForumThread, error)
	ForumRootPostCount(threadID int64) (int, error)
	ForumRootPosts(threadID int64, offset, limit int) ([]db.ForumThreadPost, error)
	ForumRootPostIDs(threadID int64) ([]int64, error)
	ForumPostReplies(postID int64) ([]db.ForumThreadPost, error)
	ForumPost(id int64) (*db.ForumThreadPost, error)
	ForumPostContents(postIDs []int64) (map[int64]db.ForumPostContent, error)
	RecentPostCount(categoryIDs []int64, comments bool) (int, error)
	RecentPosts(categoryIDs []int64, comments bool, offset, limit int) ([]db.RecentPost, error)
	UsernamesLower() (map[string]bool, error)
	VoteByUser(articleID int64, userID *int64) (float64, bool, error)
	UserByID(id int64) (*db.User, error)
	Members(roleID *int64, offset, limit int) ([]db.Member, error)
	MemberCount(roleID *int64) (int, error)
	RoleByRef(ref string) (*roles.Role, error)
	ArticleHasAuthor(articleID, userID int64) (bool, error)
	RolesByUser(id int64) ([]roles.Role, error)

	PostLikeCounts(postIDs []int64) (map[int64]int, error)
	PostsLikedBy(userID int64, postIDs []int64) (map[int64]bool, error)
	PostLikeCount(postID int64) (int, error)
	PostLikers(postID int64, offset, limit int) ([]db.User, error)
	LikePost(postID, userID int64) (bool, error)
	UnlikePost(postID, userID int64) error

	ArticleFavouriteCount(articleID int64) (int, error)
	HasFavourited(articleID, userID int64) (bool, error)
	AddFavourite(articleID, userID int64) error
	RemoveFavourite(articleID, userID int64) error

	TagsCategories() ([]db.TagsCategory, error)
	AllTags() ([]db.NamedTag, error)

	ForumPostVersions(postID int64) ([]db.ForumPostVersion, error)
	ForumPostSource(postID int64, at *time.Time) (string, error)
	CreateForumPost(w db.ForumPostWrite) (int64, error)
	CreateForumThread(w db.ForumThreadWrite, source string) (int64, int64, error)
	UpdateForumPost(postID int64, name, source, previous string, authorID *int64) error
	UpdateForumThread(threadID int64, e db.ForumThreadEdit) error
	DeleteForumPost(postID int64) error
	SendNotification(kind, meta string, recipients []int64) error
	ThreadSubscribers(threadID int64) ([]int64, error)
	SubscribeToThread(userID, threadID int64) error
	ActiveUsersByNames(names []string) ([]db.User, error)

	Subject(user *db.User) (perms.Subject, error)
	ForumSectionObject(s *db.ForumSection) *perms.Object
	ForumThreadObject(t *db.ForumThread, u *db.User) (*perms.Object, error)
	ForumPostObject(p *db.ForumThreadPost, thread *perms.Object, u *db.User) *perms.Object
}

type Env struct {
	// Name is what the wikitext spelled, which is what an error block about
	// this module has to quote back.
	Name string

	Page *page.Context
	Loc  *i18n.Localizer
	Site *db.Site
	User *db.User
	Data Data

	// Render runs the wikitext a module produced through ftml. A module that
	// lists pages has to, since what it writes is source and not markup.
	Render func(source string, pc *page.Context) (string, error)

	RenderMessage func(source string) (string, error)

	// A module that summarises a forum post asks for the same pass without the
	// markup, and only when its format wants a summary.
	RenderMessageText func(source string) (string, error)

	Vars page.VarSource
}

func (e Env) Text(id string, args ...any) string {
	if e.Loc == nil {
		return id
	}
	return e.Loc.T(id, args...)
}

func (e Env) MediaDomain() string {
	if e.Site == nil {
		return ""
	}
	return e.Site.MediaDomain
}

type Renderer func(env Env, params map[string]string, body string) (string, error)

var renderers = map[string]Renderer{}

// Register is called from each module's own file. A name the registry does not
// know is a typo, so it panics rather than registering something unreachable.
func Register(name string, r Renderer) {
	if _, ok := registry[name]; !ok {
		panic("module: " + name + " is not in the registry")
	}
	renderers[name] = r
}

// The path parameters are not folded in here, since the modules that want
// them disagree on which side wins.
func Render(env Env, name string, params map[string]string, body string) (string, error) {
	if env.Page != nil && env.Page.PathParams.Get("nomodule") == "true" {
		return "", &Error{Message: env.Text("module-disabled")}
	}
	info, ok := Lookup(name)
	if ok && !info.Removed && allowed(env.Page, info.Name) {
		if render, ok := renderers[info.Name]; ok {
			env.Name = name
			return render(env, params, body)
		}
	}
	return "", &Error{Message: env.Text("module-unknown", "name", name)}
}

// A render that narrowed the list answers for the rest the way it answers for
// a name nobody knows, so a page cannot tell a blocked module from a typo.
func allowed(pc *page.Context, name string) bool {
	if pc == nil || len(pc.OnlyModules) == 0 {
		return true
	}
	return slices.Contains(pc.OnlyModules, name)
}

// BoolParam is get_boolean_param, where anything the list does not name reads
// as the default rather than as false.
func BoolParam(params map[string]string, key string, def bool) bool {
	value, ok := params[key]
	if !ok {
		return def
	}
	if parsed, ok := ParseBool(value); ok {
		return parsed
	}
	return def
}

// Callers have to tell a switch from a string, since one parameter can carry
// either.
func ParseBool(value string) (parsed, ok bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "t", "1", "yes":
		return true, true
	case "false", "f", "0", "no":
		return false, true
	}
	return false, false
}
