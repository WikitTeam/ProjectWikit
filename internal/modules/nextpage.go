package modules

import (
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/listpages"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
)

func init() {
	module.Register("nextpage", func(env module.Env, params map[string]string, body string) (string, error) {
		return renderNeighbour(env, params, body, "nextpage", true)
	})
	module.Register("previouspage", func(env module.Env, params map[string]string, body string) (string, error) {
		return renderNeighbour(env, params, body, "previouspage", false)
	})
}

func renderNeighbour(env module.Env, params map[string]string, body, name string, forward bool) (string, error) {
	if env.Render == nil || env.Data == nil {
		return "", &module.Error{Message: env.Text("module-failed", "name", name)}
	}
	pc := env.Page
	if pc == nil || pc.Article == nil {
		return "", nil
	}
	if pc.Depth >= maxNesting {
		return "", &module.Error{Message: env.Text("module-failed", "name", name)}
	}

	found, err := neighbour(env, pc, params, forward)
	if err != nil || found == nil {
		return "", err
	}

	body = strings.TrimSpace(body)
	if body == "" {
		body = "%%linked_title%%"
	}
	vars := page.NewVars(found, env.User, env.Vars, env.Loc)
	nested := pc.CloneWith(found, found, pc.PathParams, pc.User)
	html, err := env.Render(page.PageVars(body, vars, 1, 1)+"\n", nested)
	if err != nil {
		return "", err
	}
	pc.Merge(nested)
	return html, nil
}

func neighbour(env module.Env, pc *page.Context, params map[string]string, forward bool) (*db.Article, error) {
	byTitle := strings.EqualFold(strings.TrimSpace(params["by"]), "title")
	selector := map[string]string{"order": db.SortCreatedAt}
	if byTitle {
		selector["order"] = db.SortTitle
	}
	for _, key := range []string{"category", "tags"} {
		if value, ok := params[key]; ok {
			selector[key] = value
		}
	}
	query, err := listpages.Parse(env.Data, pc.Article, env.User, siteZone(env), selector, pc.PathParams)
	if err != nil {
		return nil, err
	}

	current := pc.Article
	if byTitle {
		return neighbourByTitle(env, query, current, forward)
	}

	one := 1
	query.Limit = &one
	query.Filter.Sort = db.Sort{Column: db.SortCreatedAt, Ascending: forward}
	if forward {
		query.Filter.CreatedAt = &db.TimeFilter{Op: db.TimeGT, End: current.CreatedAt.Add(time.Microsecond)}
	} else {
		query.Filter.CreatedAt = &db.TimeFilter{Op: db.TimeLT, Start: current.CreatedAt}
	}
	result, err := listpages.Run(env.Data, query, env.User, false)
	if err != nil || len(result.Pages) == 0 {
		return nil, err
	}
	return &result.Pages[0], nil
}

func neighbourByTitle(env module.Env, query listpages.Query, current *db.Article, forward bool) (*db.Article, error) {
	query.Filter.Sort = db.Sort{Column: db.SortTitle, Ascending: true}
	result, err := listpages.Run(env.Data, query, env.User, false)
	if err != nil {
		return nil, err
	}
	for i := range result.Pages {
		if result.Pages[i].ID != current.ID {
			continue
		}
		next := i - 1
		if forward {
			next = i + 1
		}
		if next < 0 || next >= len(result.Pages) {
			return nil, nil
		}
		return &result.Pages[next], nil
	}
	return nil, nil
}
