package modules

import (
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/listpages"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/wikinum"
)

func init() { module.Register("authorrank", renderAuthorRank) }

const authorRankDefault = 50

func renderAuthorRank(env module.Env, params map[string]string, _ string) (string, error) {
	perPage := authorRankDefault
	for _, key := range []string{"limit", "perpage"} {
		if n, err := strconv.Atoi(strings.TrimSpace(params[key])); err == nil && n > 0 {
			perPage = min(n, listpages.MaxPerPage)
		}
	}
	current := 1
	if env.Page != nil {
		if n, err := wikinum.Int(env.Page.PathParams.Get("p")); err == nil && n > 1 {
			current = n
		}
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
	if len(rows) == 0 {
		return `<div class="w-author-rank-empty">` + escape.HTML(env.Text("module-authorrank-empty")) + "</div>", nil
	}

	var b strings.Builder
	b.WriteString("<div class=\"w-author-rank\">\n<table class=\"wiki-content-table\">\n<tr>")
	for _, key := range []string{"rank", "author", "score", "pages"} {
		b.WriteString("<th>" + escape.HTML(env.Text("module-authorrank-"+key)) + "</th>")
	}
	b.WriteString("</tr>\n")
	for _, row := range rows {
		id := row.UserID
		user, err := env.Data.RenderUserByID(&id)
		if err != nil {
			return "", err
		}
		b.WriteString("<tr><td>" + strconv.Itoa(row.Place) + "</td><td>" + user + "</td><td>" +
			escape.HTML(signedScore(row.Score)) + "</td><td>" + strconv.Itoa(row.Pages) + "</td></tr>\n")
	}
	b.WriteString("</table>")
	basePath := "#"
	if env.Page != nil && env.Page.Article != nil {
		basePath = listpages.BasePath(env.Page.Article.FullName(), env.Page.PathParams)
	}
	if pager := listpages.Pagination(env.Loc, basePath, current, (rows[0].Total+perPage-1)/perPage); pager != "" {
		b.WriteString("\n" + pager)
	}
	b.WriteString("\n</div>")
	return b.String(), nil
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
