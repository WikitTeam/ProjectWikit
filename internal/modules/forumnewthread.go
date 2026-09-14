package modules

import (
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

func init() { module.Register("forumnewthread", renderForumNewThread) }

func renderForumNewThread(env module.Env, _ map[string]string, _ string) (string, error) {
	if env.Data == nil {
		return "", forumFailed(env)
	}
	setTitle(env, env.Text("module-forum-new-thread"))

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
	if !perms.Resolve(subject, nil).Has(perms.CreateForumThreads) {
		return "", &module.Error{Message: env.Text("module-forumnewthread-forbidden")}
	}

	counts, err := env.Data.ForumCategoryCounts(category.ID)
	if err != nil {
		return "", err
	}
	section, err := env.Data.ForumSection(category.SectionID)
	if err != nil {
		return "", err
	}
	viewer, err := forumViewerJSON(env)
	if err != nil {
		return "", err
	}

	canonical := forumCategoryURL(category.ID, category.Name)
	config, err := wikijson.Marshal(wikijson.Object{
		{Key: "categoryId", Value: category.ID},
		{Key: "user", Value: viewer},
		{Key: "cancelUrl", Value: canonical},
	})
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString(`<div class="forum-new-thread-box">` +
		"\n" + ind12 + `<div class="forum-breadcrumbs">` +
		"\n" + ind16 + `<a href="/forum/start">` + env.Text("module-forum-title") + `</a>` +
		"\n" + ind16 + `&raquo;` +
		"\n" + ind16 + `<a href="` + escape.HTML(canonical) + `">` +
		escape.HTML(section.Name+" / "+category.Name) + `</a>` +
		"\n" + ind16 + `&raquo;` +
		"\n" + ind16 + env.Text("module-forum-new-thread") +
		"\n" + ind12 + `</div>` +
		"\n" + ind12 + `<div class="description well">` +
		"\n" + ind16 + `<div class="statistics">` +
		"\n" + ind20 + env.Text("module-forum-threads") + strconv.Itoa(counts.Threads) +
		"\n" + ind20 + `<br>` +
		"\n" + ind20 + env.Text("module-forum-posts") + strconv.Itoa(counts.Posts) +
		"\n" + ind16 + `</div>` +
		"\n" + ind16 + escape.HTML(category.Description) +
		"\n" + ind12 + `</div>` +
		"\n" + ind8 + `</div>` +
		"\n" + ind8 + `<div class="w-forum-new-thread" data-config="` + escape.HTML(config) + `"></div>`)
	return b.String(), nil
}
