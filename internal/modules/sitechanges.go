package modules

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

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

var siteChangeFlags = map[string]string{
	"source": "S", "title": "T", "name": "R", "tags": "A", "new": "N",
	"parent": "M", "file_added": "F", "file_deleted": "F", "file_renamed": "F",
	"votes_deleted": "V", "authorship": "C", "wikidot": "W",
}

type siteChangeFlag struct {
	id   string
	desc string
}

type siteChangeRow struct {
	revNumber int
	flags     []siteChangeFlag
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
		_, desc := siteChangeTypeName(env, t)
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

func siteChangeTypeName(env module.Env, t string) (string, string) {
	flag, ok := siteChangeFlags[t]
	if !ok {
		return "?", "?"
	}
	return flag, env.Text("module-sitechanges-type-" + strings.ReplaceAll(t, "_", "-"))
}

func siteChangeRowOf(env module.Env, change db.SiteChange) (siteChangeRow, error) {
	meta, err := siteChangeMetaOf(change.Meta)
	if err != nil {
		return siteChangeRow{}, err
	}

	row := siteChangeRow{
		revNumber: change.RevNumber,
		date:      renderDate(env, change.CreatedAt),
		title:     change.ArticleTitle,
		category:  change.ArticleCategory,
		url:       "/" + change.ArticleName,
	}
	if change.ArticleCategory != "_default" {
		row.url = "/" + change.ArticleCategory + ":" + change.ArticleName
	}

	if raw, ok := meta["subtypes"]; ok {
		var subtypes []string
		if err := json.Unmarshal(raw, &subtypes); err != nil {
			return siteChangeRow{}, errSiteChange
		}
		for _, subtype := range subtypes {
			id, desc := siteChangeTypeName(env, subtype)
			row.flags = append(row.flags, siteChangeFlag{id: id, desc: desc})
		}
	} else {
		id, desc := siteChangeTypeName(env, change.Type)
		row.flags = append(row.flags, siteChangeFlag{id: id, desc: desc})
	}

	if row.comment, err = siteChangeComment(env, change, meta); err != nil {
		return siteChangeRow{}, err
	}
	if row.author, err = env.Data.RenderUserByID(change.UserID); err != nil {
		return siteChangeRow{}, err
	}
	return row, nil
}

func siteChangeComment(env module.Env, change db.SiteChange, meta siteChangeMeta) (string, error) {
	if strings.TrimSpace(change.Comment) != "" {
		return change.Comment, nil
	}

	switch change.Type {
	case "new":
		return env.Text("module-sitechanges-comment-new"), nil
	case "title":
		prev, title, err := meta.two("prev_title", "title")
		if err != nil {
			return "", err
		}
		return env.Text("module-sitechanges-comment-title", "prev", prev, "title", title), nil
	case "name":
		prev, name, err := meta.two("prev_name", "name")
		if err != nil {
			return "", err
		}
		return env.Text("module-sitechanges-comment-name", "prev", prev, "name", name), nil
	case "tags":
		return siteChangeTagComment(env, meta)
	case "parent":
		return siteChangeParentComment(env, meta)
	case "file_added":
		name, err := meta.str("name")
		if err != nil {
			return "", err
		}
		return env.Text("module-sitechanges-comment-file-added", "name", name), nil
	case "file_deleted":
		name, err := meta.str("name")
		if err != nil {
			return "", err
		}
		return env.Text("module-sitechanges-comment-file-deleted", "name", name), nil
	case "file_renamed":
		prev, name, err := meta.two("prev_name", "name")
		if err != nil {
			return "", err
		}
		return env.Text("module-sitechanges-comment-file-renamed", "prev", prev, "name", name), nil
	case "votes_deleted":
		return siteChangeVotesComment(env, meta)
	case "authorship":
		return siteChangeAuthorComment(env, meta)
	case "revert":
		rev, err := meta.str("rev_number")
		if err != nil {
			return "", err
		}
		return env.Text("module-sitechanges-comment-revert", "rev", rev), nil
	}
	return "", nil
}

func siteChangeTagComment(env module.Env, meta siteChangeMeta) (string, error) {
	added, err := meta.names("added_tags")
	if err != nil {
		return "", err
	}
	removed, err := meta.names("removed_tags")
	if err != nil {
		return "", err
	}

	var parts []string
	if len(added) > 0 {
		parts = append(parts, env.Text("module-sitechanges-comment-tags-added",
			"tags", strings.Join(added, ", ")))
	}
	if len(removed) > 0 {
		parts = append(parts, env.Text("module-sitechanges-comment-tags-removed",
			"tags", strings.Join(removed, ", ")))
	}
	return strings.Join(parts, " "), nil
}

func siteChangeParentComment(env module.Env, meta siteChangeMeta) (string, error) {
	prev, parent, err := meta.two("prev_parent", "parent")
	if err != nil {
		return "", err
	}
	switch {
	case meta.truthy("prev_parent") && meta.truthy("parent"):
		return env.Text("module-sitechanges-comment-parent-changed", "prev", prev, "parent", parent), nil
	case meta.truthy("prev_parent"):
		return env.Text("module-sitechanges-comment-parent-removed", "prev", prev), nil
	case meta.truthy("parent"):
		return env.Text("module-sitechanges-comment-parent-set", "parent", parent), nil
	}
	return "", nil
}

func siteChangeVotesComment(env module.Env, meta siteChangeMeta) (string, error) {
	mode, err := meta.str("rating_mode")
	if err != nil {
		return "", err
	}
	votes, err := meta.str("votes_count")
	if err != nil {
		return "", err
	}
	popularity, err := meta.str("popularity")
	if err != nil {
		return "", err
	}

	rating := env.Text("module-sitechanges-comment-votes-none")
	switch mode {
	case "updown":
		n, err := meta.integer("rating")
		if err != nil {
			return "", err
		}
		rating = fmt.Sprintf("%+d", n)
	case "stars":
		f, err := meta.float("rating")
		if err != nil {
			return "", err
		}
		rating = strconv.FormatFloat(f, 'f', 1, 64)
	}
	return env.Text("module-sitechanges-comment-votes",
		"rating", rating, "votes", votes, "popularity", popularity), nil
}

func siteChangeAuthorComment(env module.Env, meta siteChangeMeta) (string, error) {
	label := func(key string) (string, error) {
		ids, err := meta.ids(key)
		if err != nil {
			return "", err
		}
		users, err := env.Data.UsersByIDs(ids)
		if err != nil {
			return "", err
		}
		names := make([]string, 0, len(users))
		for i := range users {
			names = append(names, users[i].DisplayLabel())
		}
		return strings.Join(names, ", "), nil
	}

	added, err := label("added_authors")
	if err != nil {
		return "", err
	}
	removed, err := label("removed_authors")
	if err != nil {
		return "", err
	}

	var parts []string
	if added != "" {
		parts = append(parts, env.Text("module-sitechanges-comment-authors-added", "names", added))
	}
	if removed != "" {
		parts = append(parts, env.Text("module-sitechanges-comment-authors-removed", "names", removed))
	}
	return strings.Join(parts, " "), nil
}

type siteChangeMeta map[string]json.RawMessage

func siteChangeMetaOf(raw []byte) (siteChangeMeta, error) {
	if len(raw) == 0 {
		return siteChangeMeta{}, nil
	}
	var meta siteChangeMeta
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil, errSiteChange
	}
	return meta, nil
}

func (m siteChangeMeta) str(key string) (string, error) {
	raw, ok := m[key]
	if !ok {
		return "", errSiteChange
	}
	return metaText(raw), nil
}

func (m siteChangeMeta) two(first, second string) (string, string, error) {
	a, err := m.str(first)
	if err != nil {
		return "", "", err
	}
	b, err := m.str(second)
	if err != nil {
		return "", "", err
	}
	return a, b, nil
}

func (m siteChangeMeta) truthy(key string) bool {
	switch text := strings.TrimSpace(string(m[key])); text {
	case "", "null", "false", "0", "0.0", `""`, "[]", "{}":
		return false
	}
	return true
}

func (m siteChangeMeta) names(key string) ([]string, error) {
	raw, ok := m[key]
	if !ok {
		return nil, nil
	}
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, errSiteChange
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		name, ok := item["name"]
		if !ok {
			return nil, errSiteChange
		}
		out = append(out, metaText(name))
	}
	return out, nil
}

func (m siteChangeMeta) ids(key string) ([]int64, error) {
	raw, ok := m[key]
	if !ok {
		return nil, nil
	}
	var ids []int64
	if err := json.Unmarshal(raw, &ids); err != nil {
		return nil, errSiteChange
	}
	return ids, nil
}

func (m siteChangeMeta) integer(key string) (int, error) {
	raw, ok := m[key]
	if !ok {
		return 0, errSiteChange
	}
	if quoted, ok := unquoteJSON(raw); ok {
		n, err := wikinum.Int(quoted)
		if err != nil {
			return 0, errSiteChange
		}
		return n, nil
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(string(raw)), 64)
	if err != nil {
		return 0, errSiteChange
	}
	return int(f), nil
}

func (m siteChangeMeta) float(key string) (float64, error) {
	raw, ok := m[key]
	if !ok {
		return 0, errSiteChange
	}
	if quoted, ok := unquoteJSON(raw); ok {
		f, err := wikinum.Float(quoted)
		if err != nil {
			return 0, errSiteChange
		}
		return f, nil
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(string(raw)), 64)
	if err != nil {
		return 0, errSiteChange
	}
	return f, nil
}

func unquoteJSON(raw json.RawMessage) (string, bool) {
	if !strings.HasPrefix(strings.TrimSpace(string(raw)), `"`) {
		return "", false
	}
	var out string
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", false
	}
	return out, true
}

// The comment lines interpolate whatever the log entry stored, so a value that
// is not a string still has to come out as text.
func metaText(raw json.RawMessage) string {
	text := strings.TrimSpace(string(raw))
	switch text {
	case "null":
		return "None"
	case "true":
		return "True"
	case "false":
		return "False"
	}
	if unquoted, ok := unquoteJSON(raw); ok {
		return unquoted
	}
	return text
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
			b.WriteString(`<span class="spantip" title="` + escape.HTML(flag.desc) + `">` +
				escape.HTML(flag.id) + `</span>`)
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
