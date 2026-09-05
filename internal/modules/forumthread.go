package modules

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/listpages"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/pageconfig"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
	"github.com/WikitTeam/ProjectWikit/internal/wikinum"
)

func init() { module.Register("forumthread", renderForumThread) }

const (
	forumPostsPerPage = 10
	ind17             = ind16 + " "
)

type forumThreadPost struct {
	id        int64
	name      string
	isOP      bool
	author    string
	createdAt string
	likes     string
	content   string
	options   string
	replies   []forumThreadPost
}

type forumThreadView struct {
	thread  db.ForumThread
	name    string
	article *db.Article

	category    *db.ForumCategory
	sectionName string
	sectionURL  string
	categoryURL string

	createdBy  string
	createdAt  string
	total      int
	pagination string
	posts      []forumThreadPost

	canReply    bool
	contentOnly bool

	pathParams    string
	params        string
	threadOptions string
	newPost       string
}

func renderForumThread(env module.Env, params map[string]string, _ string) (string, error) {
	if env.Data == nil || env.RenderMessage == nil {
		return "", forumFailed(env)
	}
	setTitle(env, env.Text("module-forum-title"))

	var path page.PathParams
	if env.Page != nil {
		path = env.Page.PathParams
	}

	thread, err := forumThreadOf(env, path)
	if err != nil {
		return "", err
	}

	subject, err := env.Data.Subject(env.User)
	if err != nil {
		return "", err
	}
	object, err := env.Data.ForumThreadObject(thread, env.User)
	if err != nil {
		return "", err
	}
	granted := perms.Resolve(subject, object)

	view := forumThreadView{thread: *thread, contentOnly: params["contentonly"] == "yes"}
	if err := forumThreadCategory(env, &view, subject, granted); err != nil {
		return "", err
	}
	if err := forumThreadName(env, &view); err != nil {
		return "", err
	}
	setTitle(env, env.Text("module-forum-title-named", "name", view.name))

	if view.createdBy, err = env.Data.RenderUserByID(thread.AuthorID); err != nil {
		return "", err
	}
	view.createdAt = renderDate(env, thread.CreatedAt)

	if err := forumThreadPosts(env, &view, path, object); err != nil {
		return "", err
	}
	if err := forumThreadConfigs(env, &view, params, subject, granted); err != nil {
		return "", err
	}
	return forumThreadHTML(env, view), nil
}

func forumThreadOf(env module.Env, path page.PathParams) (*db.ForumThread, error) {
	shown := "None"
	if param, ok := path.Lookup("t"); ok {
		shown = param.Value
	}
	notFound := func() error {
		setStatus(env, http.StatusNotFound)
		return &module.Error{Message: env.Text("module-forumthread-not-found", "name", shown)}
	}

	id, err := wikinum.Int(shown)
	if err != nil {
		return nil, notFound()
	}
	shown = strconv.Itoa(id)

	thread, err := env.Data.ForumThread(int64(id))
	if errors.Is(err, db.ErrNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, err
	}
	return thread, nil
}

func forumThreadCategory(env module.Env, view *forumThreadView, subject perms.Subject, granted perms.Set) error {
	if view.thread.CategoryID != nil {
		if !granted.Has(perms.ViewForumThreads) {
			return &module.Error{Message: env.Text("module-forumthread-forbidden")}
		}
		category, err := env.Data.ForumCategory(*view.thread.CategoryID)
		if err != nil {
			return err
		}
		view.category = category
	} else {
		if !granted.Has(perms.ViewArticleComments) {
			return &module.Error{Message: env.Text("module-forumthread-comments-forbidden")}
		}
		if perms.Resolve(subject, nil).Has(perms.ViewForumCategories) {
			categories, err := env.Data.ForumCategories()
			if err != nil {
				return err
			}
			for i := range categories {
				if categories[i].IsForComments {
					view.category = &categories[i]
					break
				}
			}
		}
	}

	if view.category == nil {
		return nil
	}
	section, err := env.Data.ForumSection(view.category.SectionID)
	if err != nil {
		return err
	}
	view.sectionName = section.Name
	view.sectionURL = forumSectionURL(section.ID, section.Name)
	view.categoryURL = forumCategoryURL(view.category.ID, view.category.Name)
	return nil
}

func forumThreadName(env module.Env, view *forumThreadView) error {
	view.name = view.thread.Name
	if view.thread.ArticleID == nil {
		return nil
	}
	article, err := env.Data.ArticleByID(*view.thread.ArticleID)
	if err != nil {
		return err
	}
	view.article = article
	if view.thread.CategoryID == nil {
		view.name = article.DisplayName()
	}
	return nil
}

func forumThreadPosts(env module.Env, view *forumThreadView, path page.PathParams, object *perms.Object) error {
	total, err := env.Data.ForumRootPostCount(view.thread.ID)
	if err != nil {
		return err
	}
	view.total = total

	current := 1
	if n, err := wikinum.Int(path.Get("p")); err == nil {
		current = n
	} else if n, err := forumThreadPageOfPost(env, view.thread.ID, path); err != nil {
		return err
	} else {
		current = n
	}
	if current < 1 {
		current = 1
	}
	maxPage := (total + forumPostsPerPage - 1) / forumPostsPerPage
	if maxPage < 1 {
		maxPage = 1
	}
	if current > maxPage {
		current = maxPage
	}

	usernames, err := env.Data.UsernamesLower()
	if err != nil {
		return err
	}
	posts, err := env.Data.ForumRootPosts(view.thread.ID, (current-1)*forumPostsPerPage, forumPostsPerPage)
	if err != nil {
		return err
	}
	if view.posts, err = forumThreadPostInfo(env, view, posts, usernames, object); err != nil {
		return err
	}

	if env.Page != nil {
		env.Page.PathParams = env.Page.PathParams.Put(page.PathParam{Key: "p", Value: strconv.Itoa(current)})
	}
	view.pagination = listpages.Pagination(env.Loc, forumThreadShortURL(view.thread.ID), current, maxPage)
	return nil
}

// Naming a post rather than a page means walking up to the root it hangs off,
// since only root posts are paged over.
func forumThreadPageOfPost(env module.Env, threadID int64, path page.PathParams) (int, error) {
	param, ok := path.Lookup("post")
	if !ok || param.Bare {
		return 1, nil
	}
	id, err := wikinum.Int(param.Value)
	if err != nil {
		return 0, forumFailed(env)
	}

	wanted := int64(id)
	post, err := env.Data.ForumPost(wanted)
	if errors.Is(err, db.ErrNotFound) {
		post = nil
	} else if err != nil {
		return 0, err
	}
	for post != nil {
		if post.ThreadID != threadID {
			return 1, nil
		}
		if post.ReplyToID == nil {
			break
		}
		if post, err = env.Data.ForumPost(*post.ReplyToID); err != nil {
			return 0, err
		}
		wanted = post.ID
	}

	ids, err := env.Data.ForumRootPostIDs(threadID)
	if err != nil {
		return 0, err
	}
	for i, id := range ids {
		if id == wanted {
			return i/forumPostsPerPage + 1, nil
		}
	}
	return 1, nil
}

func forumThreadPostInfo(env module.Env, view *forumThreadView, posts []db.ForumThreadPost,
	usernames map[string]bool, object *perms.Object) ([]forumThreadPost, error) {

	if len(posts) == 0 {
		return nil, nil
	}
	ids := make([]int64, 0, len(posts))
	for _, post := range posts {
		ids = append(ids, post.ID)
	}
	contents, err := env.Data.ForumPostContents(ids)
	if err != nil {
		return nil, err
	}
	counts, liked, err := forumLikeState(env, ids)
	if err != nil {
		return nil, err
	}

	out := make([]forumThreadPost, 0, len(posts))
	for _, post := range posts {
		one := forumThreadPost{
			id:        post.ID,
			name:      post.Name,
			isOP:      samePerson(view.thread.AuthorID, post.AuthorID),
			createdAt: renderDate(env, post.CreatedAt),
		}
		if one.author, err = env.Data.RenderUserByID(post.AuthorID); err != nil {
			return nil, err
		}
		one.likes = forumLikeHTML(env, post.ID, counts[post.ID], liked[post.ID])
		if view.article != nil {
			if post.AuthorID != nil {
				author, err := env.Data.ArticleHasAuthor(view.article.ID, *post.AuthorID)
				if err != nil {
					return nil, err
				}
				if author {
					one.isOP = true
				}
			}
		}

		html, err := env.RenderMessage(contents[post.ID].Source)
		if err != nil {
			return nil, err
		}
		one.content = highlightMentions(html, usernames)

		replies, err := env.Data.ForumPostReplies(post.ID)
		if err != nil {
			return nil, err
		}
		if one.replies, err = forumThreadPostInfo(env, view, replies, usernames, object); err != nil {
			return nil, err
		}
		if one.options, err = forumPostOptions(env, view, post, contents[post.ID], object); err != nil {
			return nil, err
		}
		out = append(out, one)
	}
	return out, nil
}

func samePerson(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func forumPostOptions(env module.Env, view *forumThreadView, post db.ForumThreadPost,
	content db.ForumPostContent, object *perms.Object) (string, error) {

	subject, err := env.Data.Subject(env.User)
	if err != nil {
		return "", err
	}
	granted := perms.Resolve(subject, env.Data.ForumPostObject(&post, object, env.User))
	threadGranted := perms.Resolve(subject, object)

	reviser, err := forumUserJSON(env, content.AuthorID)
	if err != nil {
		return "", err
	}
	viewer, err := forumViewerJSON(env)
	if err != nil {
		return "", err
	}

	return wikijson.Marshal(wikijson.Object{
		{Key: "threadId", Value: view.thread.ID},
		{Key: "threadName", Value: view.name},
		{Key: "postId", Value: post.ID},
		{Key: "postName", Value: post.Name},
		{Key: "hasRevisions", Value: !post.CreatedAt.Equal(post.UpdatedAt)},
		{Key: "lastRevisionDate", Value: isoTimestamp(post.UpdatedAt)},
		{Key: "lastRevisionAuthor", Value: reviser},
		{Key: "user", Value: viewer},
		{Key: "canReply", Value: threadGranted.Has(forumReplyPerm(view))},
		{Key: "canEdit", Value: granted.Has(perms.EditForumPosts)},
		{Key: "canDelete", Value: granted.Has(perms.DeleteForumPosts)},
	})
}

func forumReplyPerm(view *forumThreadView) string {
	if view.article != nil {
		return perms.CommentArticles
	}
	return perms.CreateForumPosts
}

func forumThreadConfigs(env module.Env, view *forumThreadView, params map[string]string,
	subject perms.Subject, granted perms.Set) error {

	view.canReply = granted.Has(forumReplyPerm(view))

	viewer, err := forumViewerJSON(env)
	if err != nil {
		return err
	}
	if view.newPost, err = wikijson.Marshal(wikijson.Object{
		{Key: "threadId", Value: view.thread.ID},
		{Key: "threadName", Value: view.name},
		{Key: "user", Value: viewer},
	}); err != nil {
		return err
	}

	moveTo, err := forumThreadMoveTo(env, subject, granted)
	if err != nil {
		return err
	}
	ownThread := view.thread.ArticleID == nil
	var categoryID any
	if view.thread.CategoryID != nil {
		categoryID = *view.thread.CategoryID
	}
	if view.threadOptions, err = wikijson.Marshal(wikijson.Object{
		{Key: "threadId", Value: view.thread.ID},
		{Key: "threadName", Value: view.name},
		{Key: "threadDescription", Value: view.thread.Description},
		{Key: "canEdit", Value: ownThread && granted.Has(perms.EditForumThreads)},
		{Key: "canPin", Value: ownThread && granted.Has(perms.PinForumThreads)},
		{Key: "canLock", Value: granted.Has(perms.LockForumThreads)},
		{Key: "canMove", Value: ownThread && granted.Has(perms.MoveForumThreads)},
		{Key: "isLocked", Value: view.thread.IsLocked},
		{Key: "isPinned", Value: view.thread.IsPinned},
		{Key: "moveTo", Value: moveTo},
		{Key: "categoryId", Value: categoryID},
	}); err != nil {
		return err
	}

	var path page.PathParams
	if env.Page != nil {
		path = env.Page.PathParams
	}
	if view.pathParams, err = wikijson.Marshal(pathParamsObject(path)); err != nil {
		return err
	}
	view.params, err = wikijson.Marshal(paramsObject(params, nil))
	return err
}

// The section test asks about the thread rather than the section, so a section
// hidden from users never drops out here.
func forumThreadMoveTo(env module.Env, subject perms.Subject, granted perms.Set) (wikijson.Array, error) {
	out := wikijson.Array{}
	if !granted.Has(perms.ViewForumSections) {
		return out, nil
	}
	if !perms.Resolve(subject, nil).Has(perms.ViewForumCategories) {
		return out, nil
	}

	sections, err := env.Data.ForumSections()
	if err != nil {
		return nil, err
	}
	categories, err := env.Data.ForumCategories()
	if err != nil {
		return nil, err
	}
	for _, section := range sections {
		var under wikijson.Array
		for _, category := range categories {
			if category.SectionID != section.ID {
				continue
			}
			under = append(under, wikijson.Object{
				{Key: "name", Value: "  " + category.Name},
				{Key: "canMove", Value: !category.IsForComments},
				{Key: "id", Value: category.ID},
			})
		}
		if len(under) == 0 {
			continue
		}
		out = append(out, wikijson.Object{
			{Key: "name", Value: section.Name},
			{Key: "canMove", Value: false},
			{Key: "id", Value: nil},
		})
		out = append(out, under...)
	}
	return out, nil
}

func forumViewerJSON(env module.Env) (wikijson.Object, error) {
	if env.User == nil {
		return pageconfig.AnonymousUserJSON(env.Loc, true).Object(), nil
	}
	return forumUserJSON(env, &env.User.ID)
}

func forumUserJSON(env module.Env, id *int64) (wikijson.Object, error) {
	if id == nil {
		return pageconfig.SystemUserJSON().Object(), nil
	}
	user, err := env.Data.UserByID(*id)
	if err != nil {
		return nil, err
	}
	userRoles, err := env.Data.RolesByUser(user.ID)
	if err != nil {
		return nil, err
	}
	subject, err := env.Data.Subject(user)
	if err != nil {
		return nil, err
	}
	editor := perms.Resolve(subject, nil).Has(perms.EditArticles)
	return pageconfig.SignedInUserJSON(user, userRoles, true, editor).Object(), nil
}

var mentionPattern = regexp.MustCompile(`@[\p{L}\p{N}_.-]+`)

func highlightMentions(html string, usernames map[string]bool) string {
	return mentionPattern.ReplaceAllStringFunc(html, func(match string) string {
		if usernames[strings.ToLower(match[1:])] {
			return `<span class="w-user-mention">` + match + `</span>`
		}
		return match
	})
}

func isoTimestamp(t time.Time) string {
	t = t.UTC()
	if t.Nanosecond() == 0 {
		return t.Format("2006-01-02T15:04:05-07:00")
	}
	return t.Format("2006-01-02T15:04:05.000000-07:00")
}

func forumThreadShortURL(id int64) string {
	return "/forum/t-" + strconv.FormatInt(id, 10)
}

func forumThreadHTML(env module.Env, view forumThreadView) string {
	var b strings.Builder
	if !view.contentOnly {
		b.WriteString("\n" + ind8 + `<div class="forum-thread-box">` +
			"\n" + ind12 + `<div class="forum-breadcrumbs">` +
			"\n" + ind16 + `<a href="/forum/start">` + env.Text("module-forum-title") + `</a>` +
			"\n" + ind16)
		if view.category != nil {
			b.WriteString("\n" + ind16 + `&raquo;` +
				"\n" + ind16 + `<a href="` + escape.HTML(view.sectionURL) + `">` + escape.HTML(view.sectionName) + `</a>` +
				"\n" + ind16 + `&raquo;` +
				"\n" + ind16 + `<a href="` + escape.HTML(view.categoryURL) + `">` + escape.HTML(view.category.Name) + `</a>` +
				"\n" + ind16)
		}
		b.WriteString("\n" + ind16 + `&raquo;` +
			"\n" + ind16 + escape.HTML(view.name) +
			"\n" + ind12 + `</div>` +
			"\n" + ind12 + `<div class="description-block well">` +
			"\n" + ind16 + `<div class="statistics">` +
			"\n" + ind20 + env.Text("module-forumthread-created-by") + view.createdBy +
			"\n" + ind20 + `<br>` +
			"\n" + ind20 + env.Text("module-forumthread-date") + view.createdAt +
			"\n" + ind20 + `<br>` +
			"\n" + ind20 + env.Text("module-forumthread-posts") + strconv.Itoa(view.total) +
			"\n" + ind16 + `</div>` +
			"\n" + ind16)
		switch {
		case view.article != nil:
			link := `<a href="/` + escape.HTML(view.article.FullName()) + `">` +
				escape.HTML(view.article.DisplayName()) + `</a>`
			b.WriteString("\n" + ind20 + env.Text("module-forumthread-article", "link", link) + " " +
				"\n" + ind16)
		case view.thread.Description != "":
			b.WriteString("\n" + ind20 + `<div class="head">` +
				"\n" + ind24 + env.Text("module-forumthread-description") +
				"\n" + ind20 + `</div>` +
				"\n" + ind20 + escape.HTML(view.thread.Description) +
				"\n" + ind16)
		}
		b.WriteString("\n" + ind12 + `</div>` +
			"\n" + ind12 + `<div class="options w-forum-thread-options page-options-bottom" data-config="` +
			escape.HTML(view.threadOptions) + `"></div>` +
			"\n" + ind12)
	}

	b.WriteString("\n" + ind12 + `<div class="thread-container w-forum-thread"` +
		"\n" + ind17 + `id="thread-container"` +
		"\n" + ind17 + `data-forum-thread-path-params="` + escape.HTML(view.pathParams) + `"` +
		"\n" + ind17 + `data-forum-thread-params="` + escape.HTML(view.params) + `">` +
		"\n" + ind16 + `<div id="thread-container-posts">` +
		"\n" + ind20 + view.pagination +
		"\n" + ind20 + forumPostsHTML(view.posts) +
		"\n" + ind20 + view.pagination +
		"\n" + ind16 + `</div>` +
		"\n" + ind12 + `</div>` +
		"\n" + ind12)
	if view.canReply && !view.contentOnly {
		b.WriteString("\n" + ind16 + `<div class="w-forum-new-post" data-config="` +
			escape.HTML(view.newPost) + `"></div>` +
			"\n" + ind12)
	}
	b.WriteString("\n" + ind8 + `</div>`)
	return b.String()
}

func forumPostsHTML(posts []forumThreadPost) string {
	var b strings.Builder
	for _, post := range posts {
		id := strconv.FormatInt(post.id, 10)
		head := `<div class="head ">`
		if post.isOP {
			head = `<div class="head op-post">`
		}
		b.WriteString("\n" + ind8 + `<div class="post-container" id="fpc-` + id + `">` +
			"\n" + ind12 + `<div class="post" id="post-` + id + `">` +
			"\n" + ind16 + `<div class="long">` +
			"\n" + ind20 + head +
			"\n" + ind24 + `<div class="title" id="post-title-` + id + `">` +
			"\n" + ind28 + escape.HTML(post.name) +
			"\n" + ind24 + `</div>` +
			"\n" + ind24 + `<div class="info">` +
			"\n" + ind28 + post.author + " " + post.createdAt + post.likes +
			"\n" + ind24 + `</div>` +
			"\n" + ind20 + `</div>` +
			"\n" + ind20 + `<div class="content" id="post-content-` + id + `">` +
			"\n" + ind24 + post.content +
			"\n" + ind20 + `</div>` +
			"\n" + ind20 + `<div class="options w-forum-post-options" id="post-options-` + id +
			`" data-config="` + escape.HTML(post.options) + `"></div>` +
			"\n" + ind16 + `</div>` +
			"\n" + ind12 + `</div>` +
			"\n" + ind12)
		if len(post.replies) > 0 {
			b.WriteString("\n" + ind16 + forumPostsHTML(post.replies) +
				"\n" + ind12)
		}
		b.WriteString("\n" + ind8 + `</div>` +
			"\n" + ind8)
	}
	return b.String()
}

// Both halves are one query each, so a thread costs the same whether it holds
// three posts or three hundred.
func forumLikeState(env module.Env, postIDs []int64) (map[int64]int, map[int64]bool, error) {
	counts, err := env.Data.PostLikeCounts(postIDs)
	if err != nil {
		return nil, nil, err
	}
	if env.User == nil {
		return counts, map[int64]bool{}, nil
	}
	liked, err := env.Data.PostsLikedBy(env.User.ID, postIDs)
	if err != nil {
		return nil, nil, err
	}
	return counts, liked, nil
}

func forumLikeHTML(env module.Env, postID int64, count int, liked bool) string {
	state := "far"
	if liked {
		state = "fas"
	}
	id := strconv.FormatInt(postID, 10)
	return ` <span class="w-post-likes" data-post-id="` + id + `" data-liked="` +
		strconv.FormatBool(liked) + `" data-count="` + strconv.Itoa(count) + `">` +
		`<a href="javascript:;" class="like-toggle" title="` + escape.HTML(env.Text("forum-like-toggle")) + `">` +
		`<i class="` + state + ` fa-thumbs-up"></i></a>` +
		`<a href="javascript:;" class="like-count" title="` + escape.HTML(env.Text("forum-like-who")) + `">` +
		strconv.Itoa(count) + `</a></span>`
}
