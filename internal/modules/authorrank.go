package modules

import (
	"errors"
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/listpages"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/wikinum"
)

func init() { module.Register("authorrank", renderAuthorRank) }

const authorRankDefault = 50

func renderAuthorRank(env module.Env, params map[string]string, body string) (string, error) {
	if env.Render == nil || env.Data == nil {
		return "", &module.Error{Message: env.Text("module-failed", "name", "authorrank")}
	}
	pc := env.Page
	if pc == nil {
		pc = page.NewContext(nil, nil, nil, env.User)
	}
	perPage := authorRankDefault
	for _, key := range []string{"limit", "perpage"} {
		if n, err := strconv.Atoi(strings.TrimSpace(params[key])); err == nil && n > 0 {
			perPage = min(n, listpages.MaxPerPage)
		}
	}
	current := 1
	if n, err := wikinum.Int(pc.PathParams.Get("p")); err == nil && n > 1 {
		current = n
	}
	hidden, err := env.Data.HiddenCategories(env.User)
	if err != nil {
		return "", err
	}
	filter := db.AuthorRankFilter{Negate: splitList(params["negate"], true), Exclude: splitList(params["exclude"], true), Hidden: hidden}
	// Written the way ListPages takes them, a leading - leaves the name out.
	filter.Categories, filter.SkipCategories = splitSigned(params["category"], false)
	var skipTags []string
	filter.Tags, skipTags = splitSigned(params["tags"], true)
	filter.Exclude = append(filter.Exclude, skipTags...)

	query := func(page int) ([]db.AuthorRankRow, error) {
		return env.Data.AuthorRank(filter, (page-1)*perPage, perPage)
	}
	rows, err := query(current)
	if err != nil {
		return "", err
	}
	// A page past the end shows the last one, the way the other paged modules do.
	if len(rows) == 0 && current > 1 {
		if first, err := query(1); err != nil {
			return "", err
		} else if len(first) > 0 {
			current = (first[0].Total + perPage - 1) / perPage
			if rows, err = query(current); err != nil {
				return "", err
			}
		}
	}
	total := 0
	if len(rows) > 0 {
		total = rows[0].Total
	}

	chips := map[string]string{}
	chip := func(token string, id int64) (string, error) {
		if _, done := chips[token]; !done {
			html, err := env.Data.RenderUserByID(&id)
			if err != nil {
				return "", err
			}
			chips[token] = html
		}
		return token, nil
	}

	var picked *db.User
	var pickedRow *db.AuthorRankRow
	if name := strings.TrimSpace(params["user"]); name != "" {
		if picked, err = userByEitherName(env, name); err != nil {
			return "", err
		}
		if picked == nil {
			return `<div class="w-author-rank-empty">` + escape.HTML(env.Text("user-not-found", "name", name)) + "</div>", nil
		}
		if pickedRow, err = env.Data.AuthorRankOf(filter, picked.ID); err != nil {
			return "", err
		}
	}
	shared := func(key string) (string, bool, error) {
		switch key {
		case "total":
			return strconv.Itoa(total), true, nil
		case "user_ranked":
			return strconv.FormatBool(pickedRow != nil), picked != nil, nil
		case "user_linked":
			if picked == nil {
				return "", false, nil
			}
			token, err := chip(chipToken(0), picked.ID)
			return token, true, err
		case "user_rank", "user_score", "user_pages", "user_index":
			if picked == nil {
				return "", false, nil
			}
			if pickedRow == nil {
				return "", true, nil
			}
			return rankValue(strings.TrimPrefix(key, "user_"), *pickedRow), true, nil
		}
		return "", false, nil
	}
	var fillErr error
	fill := func(source string, row *db.AuthorRankRow) string {
		return page.ApplyTemplate(source, func(name string) (string, bool) {
			key := strings.ToLower(strings.TrimSpace(name))
			if row != nil {
				switch key {
				case "rank", "index", "score", "pages":
					return rankValue(key, *row), true
				case "is_user":
					return strconv.FormatBool(picked != nil && row.UserID == picked.ID), true
				case "author", "author_name":
					user, err := env.Data.UserByID(row.UserID)
					if err != nil {
						fillErr = err
						return "", true
					}
					if key == "author" {
						return user.DisplayLabel(), true
					}
					return user.Username, true
				case "author_linked":
					token, err := chip(chipToken(row.Pos), row.UserID)
					if err != nil {
						fillErr = err
					}
					return token, true
				}
			}
			value, ok, err := shared(key)
			if err != nil {
				fillErr = err
			}
			return value, ok
		})
	}

	listRows := module.BoolParam(params, "rows", true)
	if listRows && total == 0 {
		return `<div class="w-author-rank-empty">` + escape.HTML(env.Text("module-authorrank-empty")) + "</div>", nil
	}
	prepend, appendix := params["prependline"], params["appendline"]
	body = strings.TrimSpace(body)
	if body == "" {
		body = "||%%rank%%||%%author_linked%%||%%score%%||%%pages%%||"
		if _, given := params["prependline"]; !given {
			prepend = "||~ " + env.Text("module-authorrank-rank") + "||~ " + env.Text("module-authorrank-author") +
				"||~ " + env.Text("module-authorrank-score") + "||~ " + env.Text("module-authorrank-pages") + "||"
		}
	}

	var lines []string
	if prepend != "" {
		lines = append(lines, fill(prepend, nil))
	}
	if listRows {
		for i := range rows {
			lines = append(lines, fill(body, &rows[i]))
		}
	}
	if appendix != "" {
		lines = append(lines, fill(appendix, nil))
	}
	if fillErr != nil {
		return "", fillErr
	}
	html, err := env.Render(strings.Join(lines, "\n"), pc)
	if err != nil {
		return "", err
	}
	html = putChipsBack(html, chips)

	var b strings.Builder
	b.WriteString("<div class=\"w-author-rank\">\n" + html)
	if listRows {
		basePath := "#"
		if pc.Article != nil {
			basePath = listpages.BasePath(pc.Article.FullName(), pc.PathParams)
		}
		if pager := listpages.Pagination(env.Loc, basePath, current, (total+perPage-1)/perPage); pager != "" {
			b.WriteString("\n" + pager)
		}
	}
	b.WriteString("\n</div>")
	return b.String(), nil
}

func rankValue(key string, row db.AuthorRankRow) string {
	switch key {
	case "rank":
		return strconv.Itoa(row.Place)
	case "index":
		return strconv.Itoa(row.Pos)
	case "score":
		return signedScore(row.Score)
	}
	return strconv.Itoa(row.Pages)
}

// A page that names a person may spell either the account or the shown name.
func userByEitherName(env module.Env, name string) (*db.User, error) {
	for _, lookup := range []func(string) (*db.User, error){env.Data.UserByUsername, env.Data.UserByDisplayName} {
		user, err := lookup(name)
		if err == nil {
			return user, nil
		}
		if !errors.Is(err, db.ErrNotFound) {
			return nil, err
		}
	}
	return nil, nil
}

func splitList(raw string, tags bool) []string {
	var out []string
	for _, one := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ' ' }) {
		if tags {
			one = db.TagFullName(one)
		}
		out = append(out, one)
	}
	return out
}

func splitSigned(raw string, tags bool) (keep, skip []string) {
	for _, one := range splitList(raw, false) {
		out := &keep
		if rest, ok := strings.CutPrefix(one, "-"); ok {
			one, out = rest, &skip
		} else {
			one = strings.TrimPrefix(one, "+")
		}
		if one == "" {
			continue
		}
		if tags {
			one = db.TagFullName(one)
		}
		*out = append(*out, one)
	}
	return keep, skip
}

func signedScore(v float64) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if v > 0 {
		return "+" + s
	}
	return s
}
