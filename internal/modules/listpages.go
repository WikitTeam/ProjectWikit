package modules

import (
	"slices"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/listpages"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/pyjson"
)

func init() { module.Register("listpages", renderListPages) }

// The two interfaces are kept apart, and this is what stops them drifting.
var _ listpages.Source = (module.Data)(nil)

// Python runs out of stack and loses one request, while Go would lose the
// whole process.
const maxNesting = 10

func renderListPages(env module.Env, params map[string]string, body string) (string, error) {
	if env.Render == nil || env.Data == nil {
		return "", module.ErrNotPorted
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

	// The legacy spelling is copied into the parameters rather than read
	// alongside them, so the frontend gets back what the query actually used.
	if _, ok := params["created_at"]; !ok {
		if legacy, ok := params["date"]; ok {
			params["created_at"] = legacy
		}
	}

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
	out, err := renderListed(env, common, listed, body, prepend, appendix, result, separate)
	if err != nil {
		return "", err
	}
	pc.Merge(common)

	if !wrapper {
		return out, nil
	}
	return wrap(env, pc, out, result, params, body, nullParams)
}

func renderListed(env module.Env, common *page.Context, listed []db.Article,
	body, prepend, appendix string, result listpages.Result, separate bool) (string, error) {

	index := result.PageIndex
	next := func(a *db.Article) string {
		index++
		vars := page.NewVars(a, env.User, env.Vars, env.Loc)
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

	pathJSON, err := pyjson.Marshal(pathParamsObject(pc.PathParams))
	if err != nil {
		return "", err
	}
	paramsJSON, err := pyjson.Marshal(paramsObject(params, nullParams))
	if err != nil {
		return "", err
	}

	pagination := listpages.Pagination(env.Loc, basePath, result.Page, result.TotalPages)
	return listpages.Wrap(out, pagination, pathJSON, paramsJSON, pyjson.String(body), pageID), nil
}

func pathParamsObject(params page.PathParams) pyjson.Object {
	out := make(pyjson.Object, 0, len(params))
	for _, param := range params {
		var value any
		if !param.Bare {
			value = param.Value
		}
		out = append(out, pyjson.Field{Key: param.Key, Value: value})
	}
	return out
}

// The keys are sorted because the order ftml hands them over in is not stable,
// and an attribute that changes between two renders cannot be compared.
func paramsObject(params map[string]string, null map[string]bool) pyjson.Object {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	out := make(pyjson.Object, 0, len(keys))
	for _, key := range keys {
		var value any
		if !null[key] {
			value = params[key]
		}
		out = append(out, pyjson.Field{Key: key, Value: value})
	}
	return out
}
