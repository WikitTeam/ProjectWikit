package modules

import (
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/listpages"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

func init() { module.Register("wantedpages", renderWantedPages) }

const wantedPerPageCap = 250

func renderWantedPages(env module.Env, params map[string]string, _ string) (string, error) {
	if env.Data == nil {
		return "", &module.Error{Message: env.Text("module-failed", "name", env.Name)}
	}
	pc := env.Page
	if pc == nil {
		pc = page.NewContext(nil, nil, nil, env.User)
	}

	// The source side reuses the page filter whole, so category_from is copied
	// over the name that filter knows and its own window is dropped.
	if from, ok := params["category_from"]; ok {
		params["category"] = from
	}
	dropped := map[string]bool{}
	for _, key := range []string{"limit", "offset"} {
		if _, ok := params[key]; ok {
			dropped[key] = true
			delete(params, key)
		}
	}

	categoryTo := "*"
	if to, ok := params["category_to"]; ok {
		categoryTo = to
	}
	var path page.PathParams
	if value, ok := pc.PathParams.Lookup("p"); ok {
		path = path.Put(value)
	}
	window, err := listpages.Parse(env.Data, pc.Article, env.User,
		map[string]string{"category": categoryTo, "perpage": params["perpage"]}, path)
	if err != nil {
		return "", err
	}

	sources, err := wantedSourceNames(env, pc, params)
	if err != nil {
		return "", err
	}

	filter := db.WantedFilter{
		From:          sources,
		Categories:    window.Filter.Categories,
		NotCategories: window.Filter.NotCategories,
	}
	total, err := env.Data.WantedLinkCount(filter)
	if err != nil {
		return "", err
	}

	perPage := min(window.PerPage, wantedPerPageCap)
	links, err := env.Data.WantedLinks(filter, (window.Page-1)*perPage, perPage)
	if err != nil {
		return "", err
	}

	pathJSON, err := wikijson.Marshal(pathParamsObject(pc.PathParams))
	if err != nil {
		return "", err
	}
	// Dropping the two keys outright would tell the frontend they were never
	// given, so they go back with no value instead.
	shownParams := map[string]string{}
	for key, value := range params {
		shownParams[key] = value
	}
	for key := range dropped {
		shownParams[key] = ""
	}
	paramsJSON, err := wikijson.Marshal(paramsObject(shownParams, dropped))
	if err != nil {
		return "", err
	}

	pageID := ""
	if pc.Article != nil {
		pageID = pc.Article.FullName()
	}
	pagination := listpages.Pagination(env.Loc, "", window.Page, wantedPageCount(total, perPage))
	return wantedPagesHTML(env, links, pagination, pathJSON, paramsJSON, pageID), nil
}

func wantedSourceNames(env module.Env, pc *page.Context, params map[string]string) ([]string, error) {
	query, err := listpages.Parse(env.Data, pc.Article, env.User, params, nil)
	if err != nil {
		return nil, err
	}
	result, err := listpages.Run(env.Data, query, env.User, false)
	if err != nil {
		return nil, err
	}
	// The link table stores whatever the wikitext spelled, so the comparison
	// happens on the form that always carries a category.
	out := make([]string, 0, len(result.Pages))
	for i := range result.Pages {
		out = append(out, result.Pages[i].Category+":"+result.Pages[i].Name)
	}
	return out, nil
}

func wantedPageCount(total, perPage int) int {
	if perPage <= 0 {
		return 1
	}
	return max(1, (total+perPage-1)/perPage)
}

func wantedPagesHTML(env module.Env, links []db.WantedLink, pagination, pathJSON, paramsJSON, pageID string) string {
	const ind16 = "\n                "
	const ind20 = "\n                    "
	const ind24 = "\n                        "
	const ind28 = "\n                            "

	var b strings.Builder
	b.WriteString(`<div class="w-wanted-pages"` +
		ind16 + `data-wanted-pages-path-params="` + escape.HTML(pathJSON) + `"` +
		ind16 + `data-wanted-pages-params="` + escape.HTML(paramsJSON) + `"` +
		ind16 + `data-wanted-pages-page-id="` + escape.HTML(pageID) + `">` +
		ind16 + pagination +
		ind16 + `<table class="form grid" style="margin: 1em auto;">` +
		ind20 + "<tbody>" +
		ind24 + "<tr><th>" + env.Text("module-wantedpages-source") +
		"</th><th>" + env.Text("module-wantedpages-missing") + "</th></tr>" +
		ind24)
	for _, link := range links {
		b.WriteString(ind24 + "<tr>" +
			ind28 + `<td><a href="/` + escape.HTML(link.From) + `">` + escape.HTML(link.Title) + "</a></td>" +
			ind28 + `<td><a href="/` + escape.HTML(link.To) + `" class="newpage">` + escape.HTML(link.To) + "</a></td>" +
			ind24 + "</tr>\n" + ind24)
	}
	b.WriteString(ind20 + "</tbody>" +
		ind16 + "</table>" +
		ind16 + pagination +
		ind16 + "</div>")
	return b.String()
}
