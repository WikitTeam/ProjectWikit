package modules

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/listpages"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/wikinum"
)

func init() { module.Register("forumcategory", renderForumCategory) }

const forumThreadsPerPage = 20

type forumCategoryThread struct {
	name        string
	description string
	url         string
	createdBy   string
	createdAt   string
	postCount   int
	isPinned    bool
	lastPostURL string
	lastPostBy  string
	lastPostAt  string
}

type forumCategoryView struct {
	category    db.ForumCategory
	sectionURL  string
	sectionName string
	rows        []forumCategoryThread
	counts      db.ForumCounts
	byStart     bool
	canCreate   bool
	pagination  string
}

func renderForumCategory(env module.Env, _ map[string]string, _ string) (string, error) {
	if env.Data == nil {
		return "", forumFailed(env)
	}
	setTitle(env, env.Text("module-forum-title"))

	var path page.PathParams
	if env.Page != nil {
		path = env.Page.PathParams
	}

	category, err := forumCategoryOf(env, path)
	if err != nil {
		return "", err
	}

	subject, err := env.Data.Subject(env.User)
	if err != nil {
		return "", err
	}
	sitewide := perms.Resolve(subject, nil)
	if !sitewide.Has(perms.ViewForumCategories) {
		return "", &module.Error{Message: env.Text("module-forumcategory-forbidden")}
	}
	setTitle(env, env.Text("module-forum-title-named", "name", category.Name))

	view := forumCategoryView{
		category:  *category,
		byStart:   path.Get("sort") == "start",
		canCreate: sitewide.Has(perms.CreateForumThreads),
	}
	if view.counts, err = env.Data.ForumCategoryCounts(category.ID); err != nil {
		return "", err
	}
	section, err := env.Data.ForumSection(category.SectionID)
	if err != nil {
		return "", err
	}
	view.sectionName = section.Name
	view.sectionURL = forumSectionURL(section.ID, section.Name)

	if err := forumCategoryThreads(env, &view, path); err != nil {
		return "", err
	}
	return forumCategoryHTML(env, view), nil
}

func forumCategoryOf(env module.Env, path page.PathParams) (*db.ForumCategory, error) {
	shown := "None"
	if param, ok := path.Lookup("c"); ok {
		shown = param.Value
	}
	notFound := func() error {
		setStatus(env, http.StatusNotFound)
		return &module.Error{Message: env.Text("module-forum-not-found", "name", shown)}
	}

	id, err := wikinum.Int(shown)
	if err != nil {
		return nil, notFound()
	}
	shown = strconv.Itoa(id)

	category, err := env.Data.ForumCategory(int64(id))
	if errors.Is(err, db.ErrNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, err
	}
	return category, nil
}

func forumCategoryThreads(env module.Env, view *forumCategoryView, path page.PathParams) error {
	sort := db.ForumThreadsByReply
	if view.byStart {
		sort = db.ForumThreadsByStart
	}

	total := view.counts.Threads
	if view.category.IsForComments {
		comments, err := env.Data.ForumCommentCounts()
		if err != nil {
			return err
		}
		total = comments.Threads
	}

	page := 1
	if n, err := wikinum.Int(path.Get("p")); err == nil {
		page = n
	}
	if page < 1 {
		page = 1
	}
	maxPage := (total + forumThreadsPerPage - 1) / forumThreadsPerPage
	if maxPage < 1 {
		maxPage = 1
	}
	if page > maxPage {
		page = maxPage
	}

	offset := (page - 1) * forumThreadsPerPage
	var (
		threads []db.ForumThread
		err     error
	)
	if view.category.IsForComments {
		threads, err = env.Data.ForumCommentThreads(sort, offset, forumThreadsPerPage)
	} else {
		threads, err = env.Data.ForumThreads(view.category.ID, sort, offset, forumThreadsPerPage)
	}
	if err != nil {
		return err
	}

	for _, thread := range threads {
		row, err := forumCategoryRow(env, thread)
		if err != nil {
			return err
		}
		view.rows = append(view.rows, row)
	}
	view.pagination = listpages.Pagination(env.Loc, forumCategoryShortURL(view.category.ID), page, maxPage)
	return nil
}

func forumCategoryRow(env module.Env, thread db.ForumThread) (forumCategoryThread, error) {
	name := thread.Name
	if thread.CategoryID == nil && thread.ArticleID != nil {
		article, err := env.Data.ArticleByID(*thread.ArticleID)
		if err != nil {
			return forumCategoryThread{}, err
		}
		name = article.DisplayName()
	}

	url := forumThreadURL(thread.ID, name)
	row := forumCategoryThread{
		name:        name,
		description: thread.Description,
		url:         url,
		createdAt:   renderDate(env, thread.CreatedAt),
		isPinned:    thread.IsPinned,
	}
	createdBy, err := env.Data.RenderUserByID(thread.AuthorID)
	if err != nil {
		return forumCategoryThread{}, err
	}
	row.createdBy = createdBy

	row.postCount, err = env.Data.ForumThreadPostCount(thread.ID)
	if err != nil {
		return forumCategoryThread{}, err
	}
	if row.postCount == 0 {
		return row, nil
	}

	last, err := env.Data.ForumThreadLastPost(thread.ID, row.postCount)
	if err != nil {
		return forumCategoryThread{}, err
	}
	row.lastPostURL = url + "#post-" + strconv.FormatInt(last.ID, 10)
	row.lastPostAt = renderDate(env, last.CreatedAt)
	row.lastPostBy, err = env.Data.RenderUserByID(last.AuthorID)
	if err != nil {
		return forumCategoryThread{}, err
	}
	return row, nil
}

func forumCategoryHTML(env module.Env, view forumCategoryView) string {
	category := view.category
	canonical := forumCategoryURL(category.ID, category.Name)
	short := forumCategoryShortURL(category.ID)

	var b strings.Builder
	b.WriteString(`<div class="forum-category-box">` +
		"\n" + ind12 + `<div class="forum-breadcrumbs">` +
		"\n" + ind16 + `<a href="/forum/start">` + env.Text("module-forum-title") + `</a>` +
		"\n" + ind16 + `&raquo;` +
		"\n" + ind16 + `<a href="` + escape.HTML(view.sectionURL) + `">` + escape.HTML(view.sectionName) + `</a>` +
		"\n" + ind16 + `&raquo;` +
		"\n" + ind16 + escape.HTML(category.Name) +
		"\n" + ind12 + `</div>` +
		"\n" + ind12 + `<div class="description-block well">` +
		"\n" + ind16 + `<div class="statistics">` +
		"\n" + ind20 + env.Text("module-forum-threads") + strconv.Itoa(view.counts.Threads) +
		"\n" + ind20 + `<br>` +
		"\n" + ind20 + env.Text("module-forum-posts") + strconv.Itoa(view.counts.Posts) +
		"\n" + ind16 + `</div>` +
		"\n" + ind16 + escape.HTML(category.Description) +
		"\n" + ind12 + `</div>` +
		"\n" + ind12 + `<div class="options">` +
		"\n" + ind16 + env.Text("module-forumcategory-sort") +
		"\n" + ind16 + `<div>` +
		"\n" + ind20)
	b.WriteString("\n" + ind24 +
		forumSortButton(env, canonical, "module-forumcategory-sort-reply", view.byStart) +
		"\n" + ind20)
	b.WriteString("\n" + ind20 + `<br>` +
		"\n" + ind20)
	b.WriteString("\n" + ind24 +
		forumSortButton(env, short+"/sort/start", "module-forumcategory-sort-start", !view.byStart) +
		"\n" + ind20)
	b.WriteString("\n" + ind16 + `</div>` +
		"\n" + ind12 + `</div>` +
		"\n" + ind12)

	if !category.IsForComments && view.canCreate {
		b.WriteString("\n" + ind16 + `<div class="new-post">` +
			"\n" + ind20 + `<a href="/forum:new-thread/c/` + strconv.FormatInt(category.ID, 10) + `">` +
			env.Text("module-forum-new-thread") + `</a>` +
			"\n" + ind16 + `</div>` +
			"\n" + ind12)
	}

	b.WriteString("\n" + ind12 + view.pagination +
		"\n" + ind12 + `<table class="table" style="width: 98%">` +
		"\n" + ind12 + `<tbody>` +
		"\n" + ind12 + `<tr class="head">` +
		"\n" + ind16 + `<td>` + env.Text("module-forumcategory-thread") + `</td>` +
		"\n" + ind16 + `<td>` + env.Text("module-forumcategory-started") + `</td>` +
		"\n" + ind16 + `<td>` + env.Text("module-forumcategory-replies") + `</td>` +
		"\n" + ind16 + `<td>` + env.Text("module-forum-last-post") + `</td>` +
		"\n" + ind12 + `</tr>` +
		"\n" + ind12)

	for _, row := range view.rows {
		b.WriteString("\n" + ind16 + `<tr>` +
			"\n" + ind20 + `<td class="name">` +
			"\n" + ind24 + `<div class="title">` +
			"\n" + ind28)
		if row.isPinned {
			b.WriteString("\n" + ind28 + env.Text("module-forumcategory-pinned") + " " +
				"\n" + ind28)
		}
		b.WriteString("\n" + ind28 + `<a href="` + escape.HTML(row.url) + `">` + escape.HTML(row.name) + `</a>` +
			"\n" + ind24 + `</div>` +
			"\n" + ind24 + `<div class="description">` + escape.HTML(row.description) + `</div>` +
			"\n" + ind20 + `</td>` +
			"\n" + ind20 + `<td class="started">` +
			"\n" + ind24 + env.Text("module-forum-author") + row.createdBy +
			"\n" + ind24 + `<br>` +
			"\n" + ind24 + row.createdAt +
			"\n" + ind20 + `</td>` +
			"\n" + ind20 + `<td class="posts">` +
			"\n" + ind24 + strconv.Itoa(row.postCount) +
			"\n" + ind20 + `</td>` +
			"\n" + ind20 + `<td class="last">` +
			"\n" + ind24)
		if row.lastPostURL != "" {
			b.WriteString("\n" + ind24 + env.Text("module-forum-author") + row.lastPostBy +
				"\n" + ind24 + `<br>` +
				"\n" + ind24 + row.lastPostAt +
				"\n" + ind24 + `<br>` +
				"\n" + ind24 + `<a href="` + escape.HTML(row.lastPostURL) + `">` +
				env.Text("module-forum-view-post") + `</a>` +
				"\n" + ind24)
		}
		b.WriteString("\n" + ind20 + `</td>` +
			"\n" + ind16 + `</tr>` +
			"\n" + ind12)
	}

	b.WriteString("\n" + ind12 + `</tbody>` +
		"\n" + ind12 + `</table>` +
		"\n" + ind12 + view.pagination +
		"\n" + ind8 + `</div>`)
	return b.String()
}

func forumSortButton(env module.Env, href, label string, active bool) string {
	if active {
		return `<a href="` + escape.HTML(href) + `" class="btn btn-primary btn-small btn-sm">` +
			env.Text(label) + `</a>`
	}
	return `<span class="btn btn-primary disabled btn-small btn-sm"><strong>` +
		env.Text(label) + `</strong></span>`
}

func forumCategoryShortURL(id int64) string {
	return "/forum/c-" + strconv.FormatInt(id, 10)
}
