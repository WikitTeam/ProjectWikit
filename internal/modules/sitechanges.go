package modules

import (
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/changelog"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/escape"
	"github.com/WikitTeam/ProjectWikit/internal/listpages"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
	"github.com/WikitTeam/ProjectWikit/internal/wikinum"
)

func init() { module.Register("sitechanges", renderSiteChanges) }

// Every path parameter this rejects reaches the reader as the same block, so
// nothing here records which one it was.
var errSiteChange = errors.New("modules: site change is not renderable")

// Sorting this list would reorder the type checkboxes on the page.
var siteChangeTypes = []string{
	"source", "title", "name", "tags", "new", "parent",
	"file_added", "file_deleted", "file_renamed", "votes_deleted",
	"authorship", "wikidot", "revert",
}

type siteChangeRow struct {
	revNumber int
	flags     []changelog.Flag
	comment   string
	author    string
	date      string
	title     string
	category  string
	url       string
}

type siteChangesOption struct {
	name     string
	id       string
	selected bool
}

type siteChangesView struct {
	rows       []siteChangeRow
	categories []string
	types      []siteChangesOption
	typesEmpty bool
	category   string
	userName   string
	perPage    int
	pagination string
	pathParams string
	params     string
}

func renderSiteChanges(env module.Env, params map[string]string, _ string) (string, error) {
	if env.Data == nil {
		return "", siteChangesFailed(env)
	}
	var path page.PathParams
	if env.Page != nil {
		path = env.Page.PathParams
	}

	view, err := siteChangesBuild(env, path, params)
	if errors.Is(err, errSiteChange) {
		return "", siteChangesFailed(env)
	}
	if err != nil {
		return "", err
	}
	return siteChangesHTML(env, view), nil
}

func siteChangesFailed(env module.Env) error {
	return &module.Error{Message: env.Text("module-failed", "name", env.Name)}
}

func siteChangesBuild(env module.Env, path page.PathParams, params map[string]string) (siteChangesView, error) {
	filterTypes, err := siteChangeFilterTypes(path)
	if err != nil {
		return siteChangesView{}, err
	}
	perPage, err := siteChangePerPage(path)
	if err != nil {
		return siteChangesView{}, err
	}
	category, err := siteChangeParam(path, "category", "*")
	if err != nil {
		return siteChangesView{}, err
	}
	category = strings.ToLower(category)
	userName, err := siteChangeParam(path, "username", "")
	if err != nil {
		return siteChangesView{}, err
	}
	userName = strings.TrimSpace(strings.ToLower(userName))

	hidden, err := env.Data.HiddenCategories(env.User)
	if err != nil {
		return siteChangesView{}, err
	}
	filter := db.SiteChangeFilter{Hidden: hidden, Types: filterTypes}
	if category != "*" {
		filter.Category, filter.HasCategory = category, true
	}
	if err := siteChangeUserFilter(env, &filter, userName); err != nil {
		return siteChangesView{}, err
	}

	total, err := env.Data.SiteChangeCount(filter)
	if err != nil {
		return siteChangesView{}, err
	}
	if perPage <= 0 {
		return siteChangesView{}, errSiteChange
	}

	current := 1
	if n, err := wikinum.Int(path.Get("p")); err == nil {
		current = n
	}
	if current < 1 {
		current = 1
	}
	maxPage := max(1, (total+perPage-1)/perPage)
	if current > maxPage {
		current = maxPage
	}

	view := siteChangesView{
		typesEmpty: len(filterTypes) == 0,
		category:   category,
		userName:   userName,
		perPage:    perPage,
		pagination: listpages.Pagination(env.Loc, "", current, maxPage),
	}
	changes, err := env.Data.SiteChanges(filter, (current-1)*perPage, perPage)
	if err != nil {
		return siteChangesView{}, err
	}
	for _, change := range changes {
		row, err := siteChangeRowOf(env, change)
		if err != nil {
			return siteChangesView{}, err
		}
		view.rows = append(view.rows, row)
	}

	if view.categories, err = env.Data.ArticleCategories(hidden); err != nil {
		return siteChangesView{}, err
	}
	slices.Sort(view.categories)

	for _, t := range siteChangeTypes {
		if t == "revert" {
			continue
		}
		_, desc := changelog.TypeName(env.Loc, t)
		view.types = append(view.types, siteChangesOption{
			name: desc, id: t, selected: slices.Contains(filterTypes, t),
		})
	}

	if view.pathParams, err = wikijson.Marshal(pathParamsObject(path)); err != nil {
		return siteChangesView{}, err
	}
	if view.params, err = wikijson.Marshal(paramsObject(params, nil)); err != nil {
		return siteChangesView{}, err
	}
	return view, nil
}

func siteChangeParam(path page.PathParams, key, def string) (string, error) {
	param, ok := path.Lookup(key)
	if !ok {
		return def, nil
	}
	if param.Bare {
		return "", errSiteChange
	}
	return param.Value, nil
}

func siteChangeFilterTypes(path page.PathParams) ([]string, error) {
	var out []string
	for _, param := range path {
		if !slices.Contains(siteChangeTypes, param.Key) {
			continue
		}
		if param.Bare {
			return nil, errSiteChange
		}
		if strings.ToLower(param.Value) == "true" && !slices.Contains(out, param.Key) {
			out = append(out, param.Key)
		}
	}
	return out, nil
}

func siteChangePerPage(path page.PathParams) (int, error) {
	raw, err := siteChangeParam(path, "perpage", "20")
	if err != nil {
		return 0, err
	}
	n, err := wikinum.Int(raw)
	if err != nil {
		return 0, errSiteChange
	}
	return n, nil
}

// A search that is a piece of "system" also takes in the entries no account
// made, which is why the partial form asks for containment the other way up.
func siteChangeUserFilter(env module.Env, filter *db.SiteChangeFilter, userName string) error {
	partial := strings.HasPrefix(userName, "~")
	search := userName
	if partial {
		search = strings.TrimSpace(userName[len("~"):])
	}
	if search == "" {
		return nil
	}

	ids, err := env.Data.UserIDsByName(search, partial)
	if err != nil {
		return err
	}
	filter.HasUser, filter.UserIDs = true, ids
	if partial {
		filter.WithSystem = strings.Contains("system", search)
	} else {
		filter.WithSystem = search == "system"
	}
	return nil
}

func siteChangeRowOf(env module.Env, change db.SiteChange) (siteChangeRow, error) {
	entry, err := changelog.Of(env.Loc, env.Data.UsersByIDs, change)
	if errors.Is(err, changelog.ErrUnreadable) {
		return siteChangeRow{}, errSiteChange
	}
	if err != nil {
		return siteChangeRow{}, err
	}

	row := siteChangeRow{
		flags:     entry.Flags,
		comment:   entry.Comment,
		revNumber: change.RevNumber,
		date:      renderDate(env, change.CreatedAt),
		title:     change.ArticleTitle,
		category:  change.ArticleCategory,
		url:       "/" + change.ArticleName,
	}
	if change.ArticleCategory != "_default" {
		row.url = "/" + change.ArticleCategory + ":" + change.ArticleName
	}

	if row.author, err = env.Data.RenderUserByID(change.UserID); err != nil {
		return siteChangeRow{}, err
	}
	return row, nil
}

func siteChangesHTML(env module.Env, v siteChangesView) string {
	var b strings.Builder
	b.WriteString(`<div class="site-changes-box w-site-changes" data-site-changes-path-params="` +
		escape.HTML(v.pathParams) + `" data-site-changes-params="` + escape.HTML(v.params) + `">` +
		"\n" + ind12 + `<style>` + siteChangesStyle +
		"\n" + ind12 + `</style>` +
		"\n" + ind12 + `<form onsubmit="return false;" action="" method="get">` +
		"\n" + ind16 + `<table class="form">` +
		"\n" + ind16 + `<tbody>` +
		"\n" + ind16 + `<tr>` +
		"\n" + ind20 + `<td>` + env.Text("module-sitechanges-types") + `</td>` +
		"\n" + ind20 + `<td class="w-type-filter">` +
		"\n" + ind24 + `<label>` +
		"\n" + ind28 + `<input type="checkbox" class="checkbox"`)
	if v.typesEmpty {
		b.WriteString(` checked`)
	}
	b.WriteString(` name="*">` +
		"\n" + ind28 + env.Text("module-sitechanges-types-all") +
		"\n" + ind24 + `</label>` +
		"\n" + ind24 + `<br>` +
		"\n" + ind24)
	for _, option := range v.types {
		b.WriteString("\n" + ind28 + `<label>` +
			"\n" + ind32 + `<input type="checkbox" class="checkbox"`)
		if option.selected {
			b.WriteString(` checked`)
		}
		b.WriteString(` name="` + escape.HTML(option.id) + `">` +
			"\n" + ind32 + escape.HTML(option.name) +
			"\n" + ind28 + `</label>` +
			"\n" + ind28 + `<br>` +
			"\n" + ind24)
	}
	b.WriteString("\n" + ind20 + `</td>` +
		"\n" + ind16 + `</tr>` +
		"\n" + ind16 + `<tr>` +
		"\n" + ind20 + `<td>` + env.Text("module-sitechanges-category") + `</td>` +
		"\n" + ind20 + `<td>` +
		"\n" + ind24 + `<select id="rev-category">` +
		"\n" + ind28 + `<option value="*"`)
	if v.category == "*" {
		b.WriteString(` selected`)
	}
	b.WriteString(`>` + env.Text("module-sitechanges-category-all") + `</option>` +
		"\n" + ind28)
	for _, category := range v.categories {
		b.WriteString("\n" + ind32 + `<option value="` + escape.HTML(category) + `"`)
		if category == v.category {
			b.WriteString(` selected`)
		}
		b.WriteString(`>` +
			"\n" + ind36 + escape.HTML(category) +
			"\n" + ind32 + `</option>` +
			"\n" + ind28)
	}
	b.WriteString("\n" + ind24 + `</select>` +
		"\n" + ind20 + `</td>` +
		"\n" + ind16 + `</tr>` +
		"\n" + ind16 + `<tr>` +
		"\n" + ind20 + `<td>` + env.Text("module-sitechanges-username") +
		`<br><span style="font-size: 75%">` + env.Text("module-sitechanges-username-hint") +
		`</span></td>` +
		"\n" + ind20 + `<td>` +
		"\n" + ind24 + `<input value="` + escape.HTML(v.userName) + `" id="rev-username"> ` +
		"\n" + ind20 + `</td>` +
		"\n" + ind16 + `</tr>` +
		"\n" + ind16 + `<tr>` +
		"\n" + ind20 + `<td>` + env.Text("module-sitechanges-perpage") + `</td>` +
		"\n" + ind20 + `<td>` +
		"\n" + ind24 + `<select id="rev-perpage">`)
	for _, size := range []int{10, 20, 50, 100, 200} {
		text := strconv.Itoa(size)
		b.WriteString("\n" + ind28 + `<option value="` + text + `"`)
		if v.perPage == size {
			b.WriteString(` selected`)
		}
		b.WriteString(`>` + text + `</option>`)
	}
	b.WriteString("\n" + ind24 + `</select>` +
		"\n" + ind20 + `</td>` +
		"\n" + ind16 + `</tr>` +
		"\n" + ind16 + `</tbody>` +
		"\n" + ind16 + `</table>` +
		"\n" + ind16 + `<div class="buttons">` +
		"\n" + ind20 + `<input class="btn btn-default btn-sm" type="button" value="` +
		env.Text("module-sitechanges-refresh") + `">` +
		"\n" + ind16 + `</div>` +
		"\n" + ind12 + `</form>` +
		"\n" + ind12 + `<div id="site-changes-list" class="changes-list">` +
		"\n" + ind16 + v.pagination +
		"\n" + ind16)

	for _, row := range v.rows {
		b.WriteString("\n" + ind16 + `<div class="changes-list-item">` +
			"\n" + ind20 + `<table>` +
			"\n" + ind20 + `<tbody>` +
			"\n" + ind20 + `<tr>` +
			"\n" + ind24 + `<td class="title">` +
			"\n" + ind28 + `<a href="` + escape.HTML(row.url) + `">` +
			"\n" + ind32)
		if row.category != "_default" {
			b.WriteString("\n" + ind36 + escape.HTML(row.category) + `:` +
				"\n" + ind32)
		}
		b.WriteString("\n" + ind32 + escape.HTML(row.title) +
			"\n" + ind28 + `</a>` +
			"\n" + ind24 + `</td>` +
			"\n" + ind24 + `<td class="flags">` +
			"\n" + ind28)
		for _, flag := range row.flags {
			b.WriteString(`<span class="spantip" title="` + escape.HTML(flag.Desc) + `">` +
				escape.HTML(flag.ID) + `</span>`)
		}
		b.WriteString("\n" + ind24 + `</td>` +
			"\n" + ind24 + `<td class="mod-date">` +
			"\n" + ind28 + row.date +
			"\n" + ind24 + `</td>` +
			"\n" + ind24 + `<td class="revision-no">` +
			"\n" + ind28 + `(` + env.Text("module-sitechanges-revision") + ` ` +
			strconv.Itoa(row.revNumber) + `) ` +
			"\n" + ind24 + `</td>` +
			"\n" + ind24 + `<td class="mod-by">` +
			"\n" + ind28 + row.author +
			"\n" + ind24 + `</td>` +
			"\n" + ind20 + `</tr>` +
			"\n" + ind20 + `</tbody>` +
			"\n" + ind20 + `</table>` +
			"\n" + ind20)
		if row.comment != "" {
			b.WriteString("\n" + ind24 + `<div class="comments">` +
				"\n" + ind28 + escape.HTML(row.comment) +
				"\n" + ind24 + `</div>` +
				"\n" + ind20)
		}
		b.WriteString("\n" + ind16 + `</div>` +
			"\n" + ind16)
	}

	b.WriteString("\n" + ind16 + v.pagination +
		"\n" + ind12 + `</div>` +
		"\n" + ind8 + `</div>`)
	return b.String()
}

const siteChangesStyle = `
                .changes-list{}
                
                .changes-list .pager{
                    margin: 1em 0;
                    text-align: center;
                }
                
                .changes-list-item{
                    /* overflow: hidden; */
                    padding: 2px;
                }
                
                .changes-list-item:hover{
                    background-color: #F2F2F2;
                }
                
                .changes-list-item table{
                    width: 98%;
                    
                }
                
                .changes-list-item .title{
                
                }
                
                .changes-list-item .mod-date{
                    text-align: right;
                    width: 13em;
                
                }
                .changes-list-item .revision-no{
                    text-align: center;
                    width: 7em;
                }
                .changes-list-item .flags{
                    text-align: left;
                    width: 3em;
                }
                .changes-list-item .mod-by{
                    width: 15em;
                    text-align: left;
                }
                
                .changes-list-item .comments{
                    font-size: 95%;
                    color: #666;
                    margin: 2px 0;
                }`
