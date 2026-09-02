package modules

import (
	"slices"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/listpages"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

func init() { module.Register("listpages", renderListPages) }

// The two interfaces are kept apart, and this is what stops them drifting.
var _ listpages.Source = (module.Data)(nil)

// Without a depth cap a page that includes itself would take the whole process
// down rather than the one request.
const maxNesting = 10

func renderListPages(env module.Env, params map[string]string, body string) (string, error) {
	if env.Render == nil || env.Data == nil {
		return "", &module.Error{Message: env.Text("module-failed", "name", "listpages")}
	}
	pc := env.Page
	if pc == nil {
		pc = page.NewContext(nil, nil, nil, env.User)
	}
	if pc.Depth >= maxNesting {
		return "", &module.Error{Message: env.Text("module-failed", "name", "listpages")}
	}

	body = strings.TrimSpace(body)
	params, nullParams := listpages.URLParams(params, pc.PathParams)

	prepend := params["prependline"]
	appendix := params["appendline"]
	separate := module.BoolParam(params, "separate", true)
	wrapper := module.BoolParam(params, "wrapper", true)

	if body != "" {
		sections := listpages.Split(body)
		if sections.Head != "" {
			prepend = sections.Head
		}
		if sections.Body != "" {
			body = sections.Body
		}
		if sections.Foot != "" {
			appendix = sections.Foot
		}
	}

	retireLegacyParams(params)

	query, err := listpages.Parse(env.Data, pc.Article, env.User, params, pc.PathParams)
	if err != nil {
		return "", err
	}
	result, err := listpages.Run(env.Data, query, env.User, true)
	if err != nil {
		return "", err
	}
	listed := result.Pages
	if module.BoolParam(params, "reverse", false) {
		listed = slices.Clone(listed)
		slices.Reverse(listed)
	}

	common := pc.CloneWith(pc.Article, pc.Article, pc.PathParams, pc.User)
	out, err := renderListed(env, common, listed, body, prepend, appendix, result, separate,
		tagLinkPrefix(params["tagtarget"]))
	if err != nil {
		return "", err
	}
	pc.Merge(common)

	if !wrapper {
		return out, nil
	}
	return wrap(env, pc, out, result, params, body, nullParams)
}

var legacyOrder = map[string]string{
	"datecreatedasc":  "created_at",
	"datecreateddesc": "created_at desc",
	"dateeditedasc":   "updated_at",
	"dateediteddesc":  "updated_at desc",
	"titleasc":        "title",
	"titledesc":       "title desc",
	"ratingasc":       "rating",
	"ratingdesc":      "rating desc",
	"pagelengthasc":   "size",
	"pagelengthdesc":  "size desc",
}

// The spellings Wikidot retired are rewritten in place rather than read
// alongside the current ones, so the frontend gets back the query that ran.
func retireLegacyParams(params map[string]string) {
	rename(params, "date", "created_at")
	rename(params, "categories", "category")
	rename(params, "tag", "tags")

	if _, taken := params["range"]; !taken {
		if skip := module.BoolParam(params, "skipcurrent", false); skip {
			params["range"] = "others"
		}
	}
	if order, ok := params["order"]; ok {
		if current, ok := legacyOrder[strings.ToLower(strings.TrimSpace(order))]; ok {
			params["order"] = current
		}
	}
}

func rename(params map[string]string, from, to string) {
	if _, taken := params[to]; taken {
		return
	}
	if value, ok := params[from]; ok {
		params[to] = value
	}
}

func tagLinkPrefix(target string) string {
	target = strings.Trim(strings.TrimSpace(target), "/")
	if target == "" {
		return ""
	}
	return "/" + target + "/tag/"
}

func renderListed(env module.Env, common *page.Context, listed []db.Article,
	body, prepend, appendix string, result listpages.Result, separate bool,
	tagPrefix string) (string, error) {

	index := result.PageIndex
	next := func(a *db.Article) string {
		index++
		vars := page.NewVars(a, env.User, env.Vars, env.Loc)
		vars.SetTagPrefix(tagPrefix)
		return page.PageVars(body, vars, index, result.Total)
	}

	if !separate {
		var source strings.Builder
		if prepend != "" {
			source.WriteString(prepend + "\n")
		}
		for i := range listed {
			source.WriteString(next(&listed[i]) + "\n")
		}
		source.WriteString(appendix)
		return env.Render(source.String(), common)
	}

	var out strings.Builder
	if prepend != "" {
		html, err := env.Render(prepend+"\n", common)
		if err != nil {
			return "", err
		}
		out.WriteString(html)
	}
	for i := range listed {
		source := next(&listed[i])
		nested := common.CloneWith(&listed[i], &listed[i], common.PathParams, common.User)
		html, err := env.Render(source+"\n", nested)
		if err != nil {
			return "", err
		}
		out.WriteString(html)
		common.Merge(nested)
	}
	if appendix != "" {
		html, err := env.Render(appendix, common)
		if err != nil {
			return "", err
		}
		out.WriteString(html)
	}
	return out.String(), nil
}

func wrap(env module.Env, pc *page.Context, out string, result listpages.Result,
	params map[string]string, body string, nullParams map[string]bool) (string, error) {

	pageID := ""
	basePath := "#"
	if pc.Article != nil {
		pageID = pc.Article.FullName()
		basePath = listpages.BasePath(pageID, pc.PathParams)
	}

	pathJSON, err := wikijson.Marshal(pathParamsObject(pc.PathParams))
	if err != nil {
		return "", err
	}
	paramsJSON, err := wikijson.Marshal(paramsObject(params, nullParams))
	if err != nil {
		return "", err
	}

	pagination := listpages.Pagination(env.Loc, basePath, result.Page, result.TotalPages)
	return listpages.Wrap(out, pagination, pathJSON, paramsJSON, wikijson.String(body), pageID), nil
}

func pathParamsObject(params page.PathParams) wikijson.Object {
	out := make(wikijson.Object, 0, len(params))
	for _, param := range params {
		var value any
		if !param.Bare {
			value = param.Value
		}
		out = append(out, wikijson.Field{Key: param.Key, Value: value})
	}
	return out
}

// The keys are sorted because the order ftml hands them over in is not stable,
// and an attribute that changes between two renders cannot be compared.
func paramsObject(params map[string]string, null map[string]bool) wikijson.Object {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	out := make(wikijson.Object, 0, len(keys))
	for _, key := range keys {
		var value any
		if !null[key] {
			value = params[key]
		}
		out = append(out, wikijson.Field{Key: key, Value: value})
	}
	return out
}
