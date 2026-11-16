package modules

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/listpages"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

func init() {
	module.RegisterAPI("countpages", "get", countPagesGet)
	module.RegisterWriteAPI("listpages", "get", listPagesGet)
	module.RegisterAPI("listpages", "feed", listPagesFeed)
	module.RegisterAPI("listusers", "get", listUsersGet)
	module.RegisterWriteAPI("newpage", "check", newPageCheck)
	module.RegisterAPI("tagcloud", "list_tags", tagCloudList)
}

func listedPages(env module.Env, params map[string]string) ([]db.Article, error) {
	if env.Data == nil {
		return nil, &module.Error{Message: env.Text("module-failed", "name", env.Name)}
	}
	pc := env.Page
	if pc == nil {
		pc = page.NewContext(nil, nil, nil, env.User)
	}
	params, _ = listpages.URLParams(params, pc.PathParams)
	query, err := listpages.Parse(env.Data, pc.Article, env.User, siteZone(env), params, pc.PathParams)
	if err != nil {
		return nil, err
	}
	result, err := listpages.Run(env.Data, query, env.User, false)
	if err != nil {
		return nil, err
	}
	return result.Pages, nil
}

func countPagesGet(env module.Env, params map[string]string) (wikijson.Object, error) {
	pages, err := listedPages(env, params)
	if err != nil {
		return nil, err
	}
	total := strconv.Itoa(len(pages))
	return wikijson.Object{
		{Key: "total", Value: total},
		{Key: "count", Value: total},
	}, nil
}

func listPagesGet(env module.Env, params map[string]string) (wikijson.Object, error) {
	pages, err := listedPages(env, params)
	if err != nil {
		return nil, err
	}
	names := make(wikijson.Array, 0, len(pages))
	for i := range pages {
		names = append(names, pages[i].FullName())
	}
	return wikijson.Object{{Key: "pages", Value: names}}, nil
}

// A feed ignores the dates the listing on the page asks for, and stops at
// rssLimit, or at the listing's own limit or page size.
func listPagesFeed(env module.Env, params map[string]string) (wikijson.Object, error) {
	retireLegacyParams(params)
	delete(params, "created_at")
	count := feedSize(params)
	params["limit"] = strconv.Itoa(count)
	pages, err := listedPages(env, params)
	if err != nil {
		return nil, err
	}
	items := make(wikijson.Array, 0, len(pages))
	for i := range pages {
		a := &pages[i]
		summary, _ := page.NewVars(a, env.User, env.Vars, env.Loc).Lookup("preview")
		title := a.Title
		if title == "" {
			title = a.FullName()
		}
		items = append(items, wikijson.Object{
			{Key: "title", Value: title},
			{Key: "name", Value: a.FullName()},
			{Key: "at", Value: a.CreatedAt},
			{Key: "summary", Value: summary},
		})
	}
	return wikijson.Object{
		{Key: "title", Value: params["rss"]},
		{Key: "description", Value: params["rssdescription"]},
		{Key: "home", Value: params["rsshome"]},
		{Key: "items", Value: items},
	}, nil
}

const feedDefault = 20

func feedSize(params map[string]string) int {
	if n, err := strconv.Atoi(strings.TrimSpace(params["rsslimit"])); err == nil && n > 0 {
		return min(n, listpages.MaxPerPage)
	}
	size := feedDefault
	if n, err := strconv.Atoi(strings.TrimSpace(params["perpage"])); err == nil && n > 0 {
		size = n
	}
	if n, err := strconv.Atoi(strings.TrimSpace(params["limit"])); err == nil && n >= 0 {
		size = min(size, n)
	}
	return min(size, listpages.MaxPerPage)
}

func listUsersGet(env module.Env, params map[string]string) (wikijson.Object, error) {
	number, name := "-1", params["anonname"]
	if name == "" {
		name = env.Text("module-listusers-anonymous")
	}
	if env.User != nil {
		number = strconv.FormatInt(env.User.ID, 10)
		name = env.User.Username
	}
	return wikijson.Object{
		{Key: "number", Value: number},
		{Key: "title", Value: name},
		{Key: "name", Value: name},
		{Key: "avatar", Value: viewerAvatar(env.User)},
		{Key: "is_authenticated", Value: strconv.FormatBool(env.User != nil)},
	}, nil
}

func viewerAvatar(u *db.User) string {
	if u != nil && u.Avatar != "" {
		return "/local--files/" + u.Avatar
	}
	return "/-/static/images/default_avatar.png"
}

func newPageCheck(env module.Env, params map[string]string) (wikijson.Object, error) {
	wanted := strings.TrimSpace(params["new_fullname"])
	category := strings.TrimSpace(params["category"])
	if wanted == "" {
		return nil, &module.Error{Message: env.Text("module-newpage-no-name")}
	}
	if category != "" {
		wanted = category + ":" + wanted
	}
	name := wikidot.Normalize(wanted)
	if !wikidot.NameAllowed(name) {
		return nil, &module.Error{Message: env.Text("module-newpage-bad-name")}
	}

	existing, err := env.Data.ArticleByRef(name)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, &module.Error{Message: env.Text("module-newpage-taken")}
	}
	return wikijson.Object{
		{Key: "url", Value: "/" + keepColons(name) + "/edit/true"},
	}, nil
}

func keepColons(name string) string {
	parts := strings.Split(name, ":")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, ":")
}

func tagCloudList(env module.Env, _ map[string]string) (wikijson.Object, error) {
	categories, err := env.Data.TagsCategories()
	if err != nil {
		return nil, err
	}
	shown := make(wikijson.Array, 0, len(categories))
	for i := range categories {
		shown = append(shown, wikijson.Object{
			{Key: "id", Value: categories[i].ID},
			{Key: "name", Value: categories[i].Name},
			{Key: "description", Value: categories[i].Description},
			{Key: "slug", Value: categories[i].Slug},
		})
	}

	tags, err := env.Data.AllTags()
	if err != nil {
		return nil, err
	}
	listed := make(wikijson.Array, 0, len(tags))
	for i := range tags {
		listed = append(listed, wikijson.Object{
			{Key: "categoryId", Value: tags[i].CategoryID},
			{Key: "name", Value: tags[i].Name},
		})
	}
	return wikijson.Object{
		{Key: "categories", Value: shown},
		{Key: "tags", Value: listed},
	}, nil
}
