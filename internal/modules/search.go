package modules

import (
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

func init() { module.Register("search", renderSearch) }

func renderSearch(env module.Env, params map[string]string, _ string) (string, error) {
	var path page.PathParams
	if env.Page != nil {
		path = env.Page.PathParams
	}
	fromPath := func(key string) string { return strings.TrimSpace(path.Get(key)) }

	config := wikijson.Object{
		{Key: "placeholder", Value: firstNonBlank(params["placeholder"], env.Text("module-search-placeholder"))},
		{Key: "tags", Value: params["tags"]},
		{Key: "category", Value: params["category"]},
		{Key: "q", Value: strings.TrimSpace(firstNonBlank(path.Get("q"), params["q"]))},
		{Key: "author", Value: fromPath("author")},
		{Key: "datefrom", Value: fromPath("datefrom")},
		{Key: "dateto", Value: fromPath("dateto")},
	}
	encoded, err := wikijson.Marshal(config)
	if err != nil {
		return "", err
	}
	return `<div class="w-search-module" data-config="` + escape.HTML(encoded) + `"></div>`, nil
}

func firstNonBlank(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
