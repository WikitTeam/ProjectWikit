package modules

import (
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/listpages"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
	"github.com/WikitTeam/ProjectWikit/internal/wikinum"
)

func init() { module.Register("recentposts", renderRecentPosts) }

const recentPostsPerPage = 20

type recentPostRow struct {
	id           int64
	name         string
	url          string
	isOP         bool
	author       string
	authorRate   string
	createdAt    string
	content      string
	hasCategory  bool
	sectionName  string
	sectionURL   string
	categoryName string
	categoryURL  string
	threadName   string
	threadURL    string
}

type recentPostsView struct {
	rows       []recentPostRow
	options    []recentPostsOption
	selected   string
	pagination string
	pathParams string
	params     string
}

type recentPostsOption struct {
	name      string
	id        string
	canSelect bool
}

func renderRecentPosts(env module.Env, params map[string]string, _ string) (string, error) {
	if env.Data == nil || env.RenderMessage == nil {
		return "", forumFailed(env)
	}
	setTitle(env, env.Text("module-recentposts-title"))

	var path page.PathParams
	if env.Page != nil {
		path = env.Page.PathParams
	}

	subject, err := env.Data.Subject(env.User)
	if err != nil {
		return "", err
	}
	visible, chosen, err := recentPostsCategories(env, path, subject)
	if err != nil {
		return "", err
	}

	comments := false
	for _, c := range visible {
		if c.IsForComments {
			comments = true
			break
		}
	}
	if chosen != nil {
		comments = chosen.IsForComments
	}

	view := recentPostsView{selected: "*"}
	if chosen != nil {
		view.selected = strconv.FormatInt(chosen.ID, 10)
	}

	ids := make([]int64, 0, len(visible))
	for _, c := range visible {
		ids = append(ids, c.ID)
	}
	if chosen != nil {
		ids = []int64{chosen.ID}
		if chosen.IsForComments {
			ids, comments = nil, true
		} else {
			comments = false
		}
	}

	total, err := env.Data.RecentPostCount(ids, comments)
	if err != nil {
		return "", err
	}

	current := 1
	if n, err := wikinum.Int(path.Get("p")); err == nil {
		current = n
	}
	if current < 1 {
		current = 1
	}
	maxPage := (total + recentPostsPerPage - 1) / recentPostsPerPage
	if maxPage < 1 {
		maxPage = 1
	}
	if current > maxPage {
		current = maxPage
	}

	if err := recentPostsRows(env, &view, visible, chosen, ids, comments, current); err != nil {
		return "", err
	}
	if err := recentPostsOptions(env, &view, visible, subject); err != nil {
		return "", err
	}

	view.pagination = listpages.Pagination(env.Loc, "", current, maxPage)
	if view.pathParams, err = wikijson.Marshal(pathParamsObject(path)); err != nil {
		return "", err
	}
	if view.params, err = wikijson.Marshal(paramsObject(params, nil)); err != nil {
		return "", err
	}
	return recentPostsHTML(env, view), nil
}

// A value that is not a number takes the whole category list down with it, so
// the reader gets an empty dropdown and no posts at all.
func recentPostsCategories(env module.Env, path page.PathParams, subject perms.Subject) ([]db.ForumCategory, *db.ForumCategory, error) {
	all, err := env.Data.ForumCategories()
	if err != nil {
		return nil, nil, err
	}
	visible := all
	if !perms.Resolve(subject, nil).Has(perms.ViewForumCategories) {
		visible = nil
	}

	asked := path.Get("c")
	if !path.Has("c") {
		asked = "*"
	}
	if asked == "*" {
		return visible, nil, nil
	}

	id, err := wikinum.Int(asked)
	if err != nil {
		return nil, nil, nil
	}
	for i := range visible {
		if visible[i].ID == int64(id) {
			return visible, &visible[i], nil
		}
	}
	return visible, nil, nil
}

func recentPostsRows(env module.Env, view *recentPostsView, visible []db.ForumCategory,
	chosen *db.ForumCategory, ids []int64, comments bool, current int) error {

	posts, err := env.Data.RecentPosts(ids, comments, (current-1)*recentPostsPerPage, recentPostsPerPage)
	if err != nil {
		return err
	}
	if len(posts) == 0 {
		return nil
	}

	usernames, err := env.Data.UsernamesLower()
	if err != nil {
		return err
	}
	postIDs := make([]int64, 0, len(posts))
	for _, post := range posts {
		postIDs = append(postIDs, post.ID)
	}
	contents, err := env.Data.ForumPostContents(postIDs)
	if err != nil {
		return err
	}

	sections, err := env.Data.ForumSections()
	if err != nil {
		return err
	}
	sectionByID := make(map[int64]db.ForumSection, len(sections))
	for _, s := range sections {
		sectionByID[s.ID] = s
	}
	categoryByID := make(map[int64]db.ForumCategory, len(visible))
	for _, c := range visible {
		categoryByID[c.ID] = c
	}

	fallback := chosen
	if fallback == nil || !fallback.IsForComments {
		fallback = nil
		for i := range visible {
			if visible[i].IsForComments {
				fallback = &visible[i]
				break
			}
		}
	}

	for _, post := range posts {
		row, err := recentPostsRow(env, post, contents[post.ID], usernames,
			categoryByID, sectionByID, fallback)
		if err != nil {
			return err
		}
		view.rows = append(view.rows, row)
	}
	return nil
}

func recentPostsRow(env module.Env, post db.RecentPost, content db.ForumPostContent,
	usernames map[string]bool, categories map[int64]db.ForumCategory,
	sections map[int64]db.ForumSection, fallback *db.ForumCategory) (recentPostRow, error) {

	row := recentPostRow{
		id:        post.ID,
		name:      env.Text("module-recentposts-goto"),
		isOP:      samePerson(post.ThreadAuthorID, post.AuthorID),
		createdAt: renderDate(env, post.CreatedAt),
	}
	if trimmed := strings.TrimSpace(post.Name); trimmed != "" {
		row.name = trimmed
	}

	row.threadName = post.ThreadName
	var article *db.Article
	if post.ThreadArticleID != nil {
		found, err := env.Data.ArticleByID(*post.ThreadArticleID)
		if err != nil {
			return recentPostRow{}, err
		}
		article = found
		if post.ThreadCategoryID == nil {
			row.threadName = article.DisplayName()
		}
	}
	row.threadURL = forumThreadURL(post.ThreadID, row.threadName)
	row.url = row.threadURL + "#post-" + strconv.FormatInt(post.ID, 10)

	var err error
	if row.author, err = env.Data.RenderUserByID(post.AuthorID); err != nil {
		return recentPostRow{}, err
	}
	if article != nil {
		mode, err := articleRatingMode(env, article.Category)
		if err != nil {
			return recentPostRow{}, err
		}
		rate, voted, err := env.Data.VoteByUser(article.ID, post.AuthorID)
		if err != nil {
			return recentPostRow{}, err
		}
		row.authorRate = forumVoteHTML(env, mode, rate, voted)

		if post.AuthorID != nil {
			author, err := env.Data.ArticleHasAuthor(article.ID, *post.AuthorID)
			if err != nil {
				return recentPostRow{}, err
			}
			if author {
				row.isOP = true
			}
		}
	}

	html, err := env.RenderMessage(content.Source)
	if err != nil {
		return recentPostRow{}, err
	}
	row.content = highlightMentions(html, usernames)

	shown := fallback
	if post.ThreadCategoryID != nil {
		if own, ok := categories[*post.ThreadCategoryID]; ok {
			shown = &own
		}
	}
	if shown == nil {
		return row, nil
	}
	section := sections[shown.SectionID]
	row.hasCategory = true
	row.sectionName = section.Name
	row.sectionURL = forumSectionURL(section.ID, section.Name)
	row.categoryName = shown.Name
	row.categoryURL = forumCategoryURL(shown.ID, shown.Name)
	return row, nil
}

func recentPostsOptions(env module.Env, view *recentPostsView, visible []db.ForumCategory,
	subject perms.Subject) error {

	sections, err := env.Data.ForumSections()
	if err != nil {
		return err
	}
	for _, section := range sections {
		object := env.Data.ForumSectionObject(&section)
		if !perms.Resolve(subject, object).Has(perms.ViewForumSections) {
			continue
		}
		var under []recentPostsOption
		for _, category := range visible {
			if category.SectionID != section.ID {
				continue
			}
			under = append(under, recentPostsOption{
				name:      "  " + category.Name,
				id:        strconv.FormatInt(category.ID, 10),
				canSelect: true,
			})
		}
		if len(under) == 0 {
			continue
		}
		view.options = append(view.options, recentPostsOption{name: section.Name, id: "None"})
		view.options = append(view.options, under...)
	}
	return nil
}

func recentPostsHTML(env module.Env, view recentPostsView) string {
	var b strings.Builder
	b.WriteString(`<div class="forum-recent-posts-box w-forum-recent-posts" data-recent-posts-path-params="` +
		escape.HTML(view.pathParams) + `" data-recent-posts-params="` + escape.HTML(view.params) + `">` +
		"\n" + ind12 + `<form onsubmit="return false;" action="" method="get">` +
		"\n" + ind16 + `<table class="form">` +
		"\n" + ind16 + `<tbody>` +
		"\n" + ind16 + `<tr>` +
		"\n" + ind20 + `<td>` + env.Text("module-recentposts-select") + `</td>` +
		"\n" + ind20 + `<td>` +
		"\n" + ind24 + `<select id="recent-posts-category">` +
		"\n" + ind28 + `<option value="*"`)
	if view.selected == "*" {
		b.WriteString(` selected`)
	}
	b.WriteString(`>` + env.Text("module-recentposts-all") + `</option>` +
		"\n" + ind28)
	for _, option := range view.options {
		b.WriteString("\n" + ind32 + `<option value="` + escape.HTML(option.id) + `"`)
		if !option.canSelect {
			b.WriteString(` disabled`)
		}
		if option.canSelect && option.id == view.selected {
			b.WriteString(` selected`)
		}
		b.WriteString(`>` +
			"\n" + ind36 + escape.HTML(option.name) +
			"\n" + ind32 + `</option>` +
			"\n" + ind28)
	}
	b.WriteString("\n" + ind24 + `</select>` +
		"\n" + ind24 + `<input class="buttons btn btn-primary" type="button" value="` +
		env.Text("module-recentposts-refresh") + `">` +
		"\n" + ind20 + `</td>` +
		"\n" + ind16 + `</tr>` +
		"\n" + ind16 + `</tbody>` +
		"\n" + ind16 + `</table>` +
		"\n" + ind12 + `</form>` +
		"\n" + ind12 + `<div id="forum-recent-posts-list">` +
		"\n" + ind16 + `<div class="thread-container">` +
		"\n" + ind20 + view.pagination +
		"\n" + ind20)

	for _, row := range view.rows {
		head := `<div class="head ">`
		if row.isOP {
			head = `<div class="head op-post">`
		}
		b.WriteString("\n" + ind20 + `<div class="post-container">` +
			"\n" + ind24 + `<div class="post" id="post-` + strconv.FormatInt(row.id, 10) + `">` +
			"\n" + ind28 + `<div class="long">` +
			"\n" + ind32 + head +
			"\n" + ind36 + `<div class="title">` +
			"\n" + ind40 + `<a href="` + escape.HTML(row.url) + `">` + escape.HTML(row.name) + `</a>` +
			"\n" + ind36 + `</div>` +
			"\n" + ind36 + `<div class="info">` +
			"\n" + ind40 + row.author + " " + row.createdAt + " " + row.authorRate +
			"\n" + ind36 + `</div>` +
			"\n" + ind36 + `<span>` +
			"\n" + ind40 + env.Text("module-recentposts-from") +
			"\n" + ind40)
		if row.hasCategory {
			b.WriteString("\n" + ind40 + `<a href="` + escape.HTML(row.sectionURL) + `">` +
				escape.HTML(row.sectionName) + `</a> &raquo;` +
				"\n" + ind40 + `<a href="` + escape.HTML(row.categoryURL) + `">` +
				escape.HTML(row.categoryName) + `</a> &raquo;` +
				"\n" + ind40)
		}
		b.WriteString("\n" + ind40 + `<a href="` + escape.HTML(row.threadURL) + `">` +
			escape.HTML(row.threadName) + `</a>` +
			"\n" + ind36 + `</span>` +
			"\n" + ind32 + `</div>` +
			"\n" + ind32 + `<div class="content">` +
			"\n" + ind36 + row.content +
			"\n" + ind32 + `</div>` +
			"\n" + ind28 + `</div>` +
			"\n" + ind24 + `</div>` +
			"\n" + ind20 + `</div>` +
			"\n" + ind20)
	}

	b.WriteString("\n" + ind20 + view.pagination +
		"\n" + ind16 + `</div>` +
		"\n" + ind12 + `</div>` +
		"\n" + ind8 + `</div>`)
	return b.String()
}
