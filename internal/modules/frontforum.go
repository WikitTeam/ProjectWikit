package modules

import (
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/wikinum"
)

func init() { module.Register("frontforum", renderFrontForum) }

const (
	frontForumLimit = 20

	// How much plain text %%description%% keeps. There is no published number
	// to match here, so this one is meant to be tuned.
	frontForumSummary = 200
)

const frontForumBody = `+ %%linked_title%%

by %%author%% %%date|%O ago (%e %b %Y, %H:%M %Z)%%

%%content%%

%%comments%% | category: %%category%%`

type frontForumItem struct {
	thread   db.ForumThread
	category *db.ForumCategory
	source   string
	posts    int
	author   *db.User
}

func renderFrontForum(env module.Env, params map[string]string, body string) (string, error) {
	if env.Data == nil || env.Render == nil {
		return "", forumFailed(env)
	}
	ids := frontForumCategoryIDs(params["category"])
	if len(ids) == 0 {
		return "", &module.Error{Message: env.Text("module-frontforum-no-category")}
	}

	body = strings.TrimSpace(body)
	if body == "" {
		body = frontForumBody
	}

	items, err := frontForumItems(env, ids, params)
	if err != nil {
		return "", err
	}
	return frontForumRender(env, items, body)
}

// An id that names no category is dropped rather than emptying the listing.
func frontForumCategoryIDs(raw string) []int64 {
	var out []int64
	for _, part := range strings.Split(raw, ";") {
		id, err := wikinum.Int(strings.TrimSpace(part))
		if err != nil || id <= 0 {
			continue
		}
		out = append(out, int64(id))
	}
	return out
}

func frontForumItems(env module.Env, ids []int64, params map[string]string) ([]frontForumItem, error) {
	subject, err := env.Data.Subject(env.User)
	if err != nil {
		return nil, err
	}
	if !perms.Resolve(subject, nil).Has(perms.ViewForumCategories) {
		return nil, nil
	}

	categories, err := env.Data.ForumCategories()
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]*db.ForumCategory, len(categories))
	for i := range categories {
		byID[categories[i].ID] = &categories[i]
	}
	wanted := make([]int64, 0, len(ids))
	for _, id := range ids {
		if _, ok := byID[id]; ok {
			wanted = append(wanted, id)
		}
	}
	if len(wanted) == 0 {
		return nil, nil
	}

	limit := frontForumLimit
	if n, err := wikinum.Int(params["limit"]); err == nil && n >= 0 {
		limit = n
	}
	offset := 0
	if n, err := wikinum.Int(params["offset"]); err == nil && n > 0 {
		offset = n
	}
	if limit == 0 {
		return nil, nil
	}

	threads, err := env.Data.ForumThreadsInCategories(wanted, offset, limit)
	if err != nil {
		return nil, err
	}
	if len(threads) == 0 {
		return nil, nil
	}
	return frontForumFill(env, threads, byID)
}

// Everything the items need is asked for in one query each, because a news
// column of twenty threads would otherwise issue sixty.
func frontForumFill(env module.Env, threads []db.ForumThread, byID map[int64]*db.ForumCategory) ([]frontForumItem, error) {
	threadIDs := make([]int64, len(threads))
	for i := range threads {
		threadIDs[i] = threads[i].ID
	}

	first, err := env.Data.ForumFirstPosts(threadIDs)
	if err != nil {
		return nil, err
	}
	counts, err := env.Data.ForumThreadPostCounts(threadIDs)
	if err != nil {
		return nil, err
	}

	postIDs := make([]int64, 0, len(first))
	for _, post := range first {
		postIDs = append(postIDs, post.ID)
	}
	contents := map[int64]db.ForumPostContent{}
	if len(postIDs) > 0 {
		contents, err = env.Data.ForumPostContents(postIDs)
		if err != nil {
			return nil, err
		}
	}

	authors, err := frontForumAuthors(env, threads)
	if err != nil {
		return nil, err
	}

	out := make([]frontForumItem, 0, len(threads))
	for _, thread := range threads {
		item := frontForumItem{thread: thread, posts: counts[thread.ID]}
		if thread.CategoryID != nil {
			item.category = byID[*thread.CategoryID]
		}
		if post, ok := first[thread.ID]; ok {
			item.source = contents[post.ID].Source
		}
		if thread.AuthorID != nil {
			item.author = authors[*thread.AuthorID]
		}
		out = append(out, item)
	}
	return out, nil
}

func frontForumAuthors(env module.Env, threads []db.ForumThread) (map[int64]*db.User, error) {
	var ids []int64
	seen := map[int64]bool{}
	for i := range threads {
		if id := threads[i].AuthorID; id != nil && !seen[*id] {
			seen[*id] = true
			ids = append(ids, *id)
		}
	}
	out := map[int64]*db.User{}
	if len(ids) == 0 {
		return out, nil
	}
	users, err := env.Data.UsersByIDs(ids)
	if err != nil {
		return nil, err
	}
	for i := range users {
		out[users[i].ID] = &users[i]
	}
	return out, nil
}

func frontForumRender(env module.Env, items []frontForumItem, body string) (string, error) {
	pc := env.Page
	if pc == nil {
		pc = page.NewContext(nil, nil, nil, env.User)
	}
	common := pc.CloneWith(pc.Article, pc.SourceArticle, pc.PathParams, pc.User)

	// The summary costs a render of its own, so it is only produced for a
	// format that asks for one.
	summarise := frontForumWantsSummary(body)

	var out strings.Builder
	out.WriteString(`<div class="front-forum-box">`)
	for i := range items {
		summary := ""
		if summarise {
			text, err := frontForumText(env, items[i].source)
			if err != nil {
				return "", err
			}
			summary = text
		}
		html, err := env.Render(frontForumVars(env, &items[i], summary, body)+"\n", common)
		if err != nil {
			return "", err
		}
		out.WriteString("<div>" + html + "</div>")
	}
	out.WriteString("</div>")
	pc.Merge(common)
	return out.String(), nil
}

var frontForumSummaryNames = []string{"description", "short", "summary"}

func frontForumWantsSummary(body string) bool {
	for _, name := range frontForumSummaryNames {
		if strings.Contains(body, "%%"+name+"%%") {
			return true
		}
	}
	return false
}

func frontForumText(env module.Env, source string) (string, error) {
	if source == "" || env.RenderMessageText == nil {
		return "", nil
	}
	text, err := env.RenderMessageText(source)
	if err != nil {
		return "", err
	}
	return truncateRunes(strings.TrimSpace(text), frontForumSummary), nil
}

func truncateRunes(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return string(runes[:limit])
}

func frontForumVars(env module.Env, item *frontForumItem, summary, body string) string {
	url := forumThreadURL(item.thread.ID, item.thread.Name)

	return page.ApplyTemplate(body, func(name string) (string, bool) {
		if format, ok := strings.CutPrefix(name, "date|"); ok {
			return dateTag(item.thread.CreatedAt.Unix(), format), true
		}
		switch name {
		case "title":
			return item.thread.Name, true
		case "linked_title", "title_linked":
			return "[" + url + " " + item.thread.Name + "]", true
		case "link":
			return url, true
		case "author":
			return frontForumAuthor(env, item.author), true
		case "date":
			return dateTag(item.thread.CreatedAt.Unix(), ""), true
		case "comments":
			return strconv.Itoa(max(item.posts-1, 0)), true
		case "category":
			return frontForumCategory(item.category), true
		case "description", "short", "summary":
			return summary, true
		case "content", "text", "long", "body":
			return item.source, true
		}
		return "", false
	})
}

func frontForumAuthor(env module.Env, user *db.User) string {
	if user == nil {
		return env.Text("user-system")
	}
	return "[[*user " + user.Username + "]]"
}

func frontForumCategory(category *db.ForumCategory) string {
	if category == nil {
		return ""
	}
	return "[" + forumCategoryURL(category.ID, category.Name) + " " + category.Name + "]"
}

func dateTag(timestamp int64, format string) string {
	if strings.TrimSpace(format) == "" {
		return "[[date " + strconv.FormatInt(timestamp, 10) + "]]"
	}
	return `[[date ` + strconv.FormatInt(timestamp, 10) + ` format="` + strings.TrimSpace(format) + `"]]`
}
