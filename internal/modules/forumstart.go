package modules

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/pynum"
)

func init() { module.Register("forumstart", renderForumStart) }

type forumStartCategory struct {
	name        string
	description string
	url         string
	threads     int
	posts       int
	lastPostURL string
	lastPostBy  string
	lastPostAt  string
}

type forumStartSection struct {
	name        string
	description string
	url         string
	categories  []forumStartCategory
}

func renderForumStart(env module.Env, _ map[string]string, _ string) (string, error) {
	if env.Data == nil {
		return "", forumFailed(env)
	}
	setTitle(env, env.Text("module-forumstart-title"))

	var path page.PathParams
	if env.Page != nil {
		path = env.Page.PathParams
	}
	hidden := "hide"
	if param, ok := path.Lookup("hidden"); ok {
		hidden = param.Value
	}

	rows, only, err := forumStartSections(env, path)
	if err != nil {
		return "", err
	}

	hideURL, showURL := "/forum/start", "/forum/start/hidden/show"
	if only != nil {
		hideURL = forumSectionURL(only.ID, only.Name)
		showURL = "/forum/s-" + strconv.FormatInt(only.ID, 10) + "/hidden/show"
	}

	// Python's loop variable overwrites the section the URL named, so the
	// breadcrumbs turn up on any listing that has a section in it at all.
	var crumb *db.ForumSection
	if len(rows) != 0 {
		crumb = &rows[0]
	}

	sections, err := forumStartItems(env, rows)
	if err != nil {
		return "", err
	}
	return forumStartHTML(env, sections, crumb, hidden, hideURL, showURL), nil
}

// The hidden filter belongs to the listing alone, so a link to a hidden section
// still opens it.
func forumStartSections(env module.Env, path page.PathParams) ([]db.ForumSection, *db.ForumSection, error) {
	param, ok := path.Lookup("s")
	if !ok {
		all, err := env.Data.ForumSections()
		if err != nil {
			return nil, nil, err
		}
		hidden := path.Get("hidden") == "show"
		var rows []db.ForumSection
		for _, section := range all {
			if hidden || !section.IsHidden {
				rows = append(rows, section)
			}
		}
		return rows, nil, nil
	}

	id, err := pynum.Int(param.Value)
	if err != nil {
		return nil, nil, forumFailed(env)
	}
	section, err := env.Data.ForumSection(int64(id))
	if errors.Is(err, db.ErrNotFound) {
		setStatus(env, http.StatusNotFound)
		return nil, nil, &module.Error{Message: env.Text("module-forum-not-found", "name", param.Value)}
	}
	if err != nil {
		return nil, nil, err
	}
	setTitle(env, env.Text("module-forum-title-named", "name", section.Name))
	return []db.ForumSection{*section}, section, nil
}

func forumStartItems(env module.Env, rows []db.ForumSection) ([]forumStartSection, error) {
	categories, err := env.Data.ForumCategories()
	if err != nil {
		return nil, err
	}
	subject, err := env.Data.Subject(env.User)
	if err != nil {
		return nil, err
	}
	// Django asks this once per category, but a forum category carries no
	// overrides of its own, so the answer cannot differ between them.
	viewCategories := perms.Resolve(subject, nil).Has(perms.ViewForumCategories)

	var out []forumStartSection
	for i := range rows {
		object := env.Data.ForumSectionObject(&rows[i])
		if !perms.Resolve(subject, object).Has(perms.ViewForumSections) {
			continue
		}
		item := forumStartSection{
			name:        rows[i].Name,
			description: rows[i].Description,
			url:         forumSectionURL(rows[i].ID, rows[i].Name),
		}
		if viewCategories {
			for _, category := range categories {
				if category.SectionID != rows[i].ID {
					continue
				}
				row, err := forumStartRow(env, category)
				if err != nil {
					return nil, err
				}
				item.categories = append(item.categories, row)
			}
		}
		if len(item.categories) != 0 {
			out = append(out, item)
		}
	}
	return out, nil
}

func forumStartRow(env module.Env, category db.ForumCategory) (forumStartCategory, error) {
	row := forumStartCategory{
		name:        category.Name,
		description: category.Description,
		url:         forumCategoryURL(category.ID, category.Name),
	}

	counts, err := env.Data.ForumCategoryCounts(category.ID)
	last, lastErr := env.Data.ForumCategoryLastPost(category.ID)
	if category.IsForComments {
		counts, err = env.Data.ForumCommentCounts()
		last, lastErr = env.Data.ForumCommentLastPost()
	}
	if err != nil {
		return row, err
	}
	row.threads, row.posts = counts.Threads, counts.Posts

	if errors.Is(lastErr, db.ErrNotFound) {
		return row, nil
	}
	if lastErr != nil {
		return row, lastErr
	}

	name := last.ThreadName
	if last.ThreadCategoryID == nil && last.ThreadArticleID != nil {
		article, err := env.Data.ArticleByID(*last.ThreadArticleID)
		if err != nil {
			return row, err
		}
		name = article.DisplayName()
	}
	row.lastPostURL = forumPostURL(last.ThreadID, name, last.ID)
	row.lastPostAt = renderDate(env, last.CreatedAt)
	row.lastPostBy, err = env.Data.RenderUserByID(last.AuthorID)
	if err != nil {
		return row, err
	}
	return row, nil
}

func forumStartHTML(env module.Env, sections []forumStartSection, crumb *db.ForumSection, hidden, hideURL, showURL string) string {
	var b strings.Builder
	b.WriteString(`<div class="forum-start-box">` + "\n" + ind12)
	if crumb != nil {
		b.WriteString("\n" + ind16 + `<div class="forum-breadcrumbs">` +
			"\n" + ind20 + `<a href="/forum/start">` + env.Text("module-forum-title") + `</a>` +
			"\n" + ind20 + `&raquo;` +
			"\n" + ind20 + escape.HTML(crumb.Name) +
			"\n" + ind16 + `</div>` +
			"\n" + ind12)
	}
	b.WriteString("\n" + ind12)

	for _, section := range sections {
		b.WriteString("\n" + ind16 + `<div class="forum-group" style="width: 98%">` +
			"\n" + ind20 + `<div class="head">` +
			"\n" + ind24 + `<div class="title"><a href="` + escape.HTML(section.url) + `">` +
			escape.HTML(section.name) + `</a></div>` +
			"\n" + ind24 + `<div class="description">` + escape.HTML(section.description) + `</div>` +
			"\n" + ind20 + `</div>` +
			"\n" + ind20 + `<div>` +
			"\n" + ind24 + `<table>` +
			"\n" + ind24 + `<tbody>` +
			"\n" + ind24 + `<tr class="head">` +
			"\n" + ind28 + `<td>` + env.Text("module-forumstart-section") + `</td>` +
			"\n" + ind28 + `<td>` + env.Text("module-forumstart-threads") + `</td>` +
			"\n" + ind28 + `<td>` + env.Text("module-forumstart-posts") + `</td>` +
			"\n" + ind28 + `<td>` + env.Text("module-forum-last-post") + `</td>` +
			"\n" + ind24 + `</tr>` +
			"\n" + ind24)
		for _, category := range section.categories {
			b.WriteString("\n" + ind28 + `<tr>` +
				"\n" + ind32 + `<td class="name">` +
				"\n" + ind36 + `<div class="title"><a href="` + escape.HTML(category.url) + `">` +
				escape.HTML(category.name) + `</a></div>` +
				"\n" + ind36 + `<div class="description">` + escape.HTML(category.description) + `</div>` +
				"\n" + ind32 + `</td>` +
				"\n" + ind32 + `<td class="threads">` + strconv.Itoa(category.threads) + `</td>` +
				"\n" + ind32 + `<td class="posts">` + strconv.Itoa(category.posts) + `</td>` +
				"\n" + ind32 + `<td class="last">` +
				"\n" + ind36)
			if category.lastPostURL != "" {
				b.WriteString("\n" + ind40 + env.Text("module-forum-author") + category.lastPostBy +
					"\n" + ind40 + `<br>` +
					"\n" + ind40 + category.lastPostAt +
					"\n" + ind40 + `<br>` +
					"\n" + ind40 + `<a href="` + escape.HTML(category.lastPostURL) + `">` +
					env.Text("module-forum-view-post") + `</a>` +
					"\n" + ind36)
			}
			b.WriteString("\n" + ind32 + `</td>` +
				"\n" + ind28 + `</tr>` +
				"\n" + ind24)
		}
		b.WriteString("\n" + ind24 + `</tbody>` +
			"\n" + ind24 + `</table>` +
			"\n" + ind20 + `</div>` +
			"\n" + ind16 + `</div>` +
			"\n" + ind12)
	}

	b.WriteString("\n" + ind8 + `</div>` +
		"\n" + ind8 + `<p style="text-align: right">` +
		"\n" + ind12)
	url, label := showURL, "module-forumstart-show-hidden"
	if hidden == "show" {
		url, label = hideURL, "module-forumstart-hide-hidden"
	}
	b.WriteString("\n" + ind16 + `<a href="` + escape.HTML(url) + `">` + env.Text(label) + `</a>` +
		"\n" + ind12)
	b.WriteString("\n" + ind8 + `</p>`)
	return b.String()
}
