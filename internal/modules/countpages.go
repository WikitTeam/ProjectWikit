package modules

import (
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/listpages"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

func init() { module.Register("countpages", renderCountPages) }

func renderCountPages(env module.Env, params map[string]string, body string) (string, error) {
	if env.Render == nil || env.Data == nil {
		return "", &module.Error{Message: env.Text("module-failed", "name", "countpages")}
	}
	pc := env.Page
	if pc == nil {
		pc = page.NewContext(nil, nil, nil, env.User)
	}

	params, _ = listpages.URLParams(params, pc.PathParams)
	query, err := listpages.Parse(env.Data, pc.Article, env.User, params, pc.PathParams)
	if err != nil {
		return "", err
	}
	// The count is the rows the filter leaves, so the query runs unpaginated
	// and the reader's per-page never reaches it.
	result, err := listpages.Run(env.Data, query, env.User, false)
	if err != nil {
		return "", err
	}

	total := strconv.Itoa(len(result.Pages))
	source := page.ApplyTemplate(strings.TrimSpace(body), func(name string) (string, bool) {
		switch strings.ToLower(name) {
		case "total", "count":
			return total, true
		}
		return "", false
	})
	return env.Render(source, pc)
}
