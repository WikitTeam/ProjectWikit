package modules

import (
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/module"
)

func init() { module.Register("pagesbytag", renderPagesByTag) }

const defaultTagCategory = "_default"

func renderPagesByTag(env module.Env, params map[string]string, _ string) (string, error) {
	name, ok := params["tag"]
	if !ok {
		return "", nil
	}
	categorySlug, tagName := splitTag(name)
	category, err := env.Data.TagCategory(categorySlug)
	if err != nil {
		return "", nil
	}
	hidden, err := env.Data.HiddenCategories(env.User)
	if err != nil {
		return "", err
	}
	articles, err := env.Data.TagArticles(categorySlug, tagName, hidden)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString(`<a name="pages"></a>` + "\n        <h2>")
	from := ""
	if categorySlug != defaultTagCategory {
		from = env.Text("module-pagesbytag-category", "category", "<em>"+escape.HTML(category.Name)+"</em>")
	}
	b.WriteString(env.Text("module-pagesbytag-heading",
		"tag", "<em>"+escape.HTML(tagName)+"</em>", "category", from))
	b.WriteString("</h2>\n        " + `<div id="tagged-pages-list" class="pages-list">` + "\n            ")
	for i := range articles {
		title := articles[i].Title
		if title == "" {
			title = articles[i].FullName()
		}
		b.WriteString("\n                " + `<div class="pages-list-item">` +
			"\n                    " + `<div class="title">` +
			"\n                        " + `<a href="/` + escape.HTML(articles[i].FullName()) + `">` +
			escape.HTML(title) + "</a>\n                    </div>\n                </div>\n            ")
	}
	b.WriteString("\n        </div>")
	return b.String(), nil
}

func splitTag(fullName string) (categorySlug, name string) {
	lowered := strings.ToLower(fullName)
	if category, rest, found := strings.Cut(lowered, ":"); found {
		return category, rest
	}
	return defaultTagCategory, lowered
}
