package modules

import (
	"slices"
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

const forumCommentsScope = "comments"

func forumSearchAPI(env module.Env, params map[string]string) (wikijson.Object, error) {
	nothing := wikijson.Object{{Key: "results", Value: wikijson.Array{}}, {Key: "hasMore", Value: false}}
	filter := db.ForumSearchFilter{
		From: searchDate(params["datefrom"], false),
		To:   searchDate(params["dateto"], true),
	}
	if q := strings.TrimSpace(params["q"]); q != "" {
		filter.Words = searchSpaces.Split(q, -1)
	}
	if author := strings.TrimSpace(params["author"]); author != "" {
		found, err := searchAuthor(env, author)
		if err != nil {
			return nil, err
		}
		if found == nil {
			return nothing, nil
		}
		filter.AuthorID = &found.ID
	}
	switch place := strings.TrimSpace(params["category"]); place {
	case "":
	case forumCommentsScope:
		filter.Comments = true
	default:
		id, err := strconv.ParseInt(place, 10, 64)
		if err != nil {
			return nothing, nil
		}
		filter.CategoryID = &id
	}
	subject, err := env.Data.Subject(env.User)
	if err != nil {
		return nil, err
	}
	if filter.Sections, err = searchableSections(env, subject); err != nil {
		return nil, err
	}

	if len(filter.Words) == 0 && filter.AuthorID == nil && filter.From == nil && filter.To == nil &&
		filter.CategoryID == nil && !filter.Comments {
		return nothing, nil
	}
	words := filter.Words
	if words == nil {
		words = []string{}
	}
	offset, _ := strconv.Atoi(strings.TrimSpace(params["offset"]))
	offset = max(offset, 0)
	hits, err := env.Data.SearchForum(filter, offset, searchLimit+1)
	if err != nil {
		return nil, err
	}
	more := len(hits) > searchLimit
	hits = hits[:min(len(hits), searchLimit)]

	out := wikijson.Array{}
	for _, hit := range hits {
		visible, err := forumHitVisible(env, subject, hit)
		if err != nil {
			return nil, err
		}
		if !visible {
			continue
		}
		author := any(nil)
		if hit.AuthorID != nil {
			if u, err := env.Data.UserByID(*hit.AuthorID); err == nil && u != nil {
				author = searchAuthorField([]db.User{*u})
			}
		}
		out = append(out, wikijson.Object{
			{Key: "title", Value: hit.ThreadName},
			{Key: "post", Value: hit.PostName},
			{Key: "where", Value: hit.Where},
			{Key: "url", Value: "/forum/t-" + strconv.FormatInt(hit.ThreadID, 10) + "#post-" + strconv.FormatInt(hit.PostID, 10)},
			{Key: "excerpt", Value: searchExcerpt(hit.Source, words)},
			{Key: "words", Value: words},
			{Key: "author", Value: author},
			{Key: "createdAt", Value: searchTime(hit.CreatedAt)},
		})
	}
	return wikijson.Object{{Key: "results", Value: out}, {Key: "hasMore", Value: more}}, nil
}

func searchableSections(env module.Env, subject perms.Subject) ([]int64, error) {
	sections, err := env.Data.ForumSections()
	if err != nil {
		return nil, err
	}
	out := []int64{}
	for i := range sections {
		granted := perms.Resolve(subject, env.Data.ForumSectionObject(&sections[i]))
		if sections[i].IsIndexed && granted.Has(perms.ViewForumThreads) && granted.Has(perms.ViewForumPosts) {
			out = append(out, sections[i].ID)
		}
	}
	return out, nil
}

// A page's discussion follows the page, so a reader who may not see the page
// does not see what was said about it either. A thread of its own was already
// kept to the sections the reader may search.
func forumHitVisible(env module.Env, subject perms.Subject, hit db.ForumSearchHit) (bool, error) {
	if hit.ArticleID == nil {
		return true, nil
	}
	article, err := env.Data.ArticleByID(*hit.ArticleID)
	if err != nil || article == nil {
		return false, nil
	}
	object, err := env.Data.ArticleObject(article, env.User)
	if err != nil {
		return false, err
	}
	granted := perms.Resolve(subject, object)
	return granted.Has(perms.ViewArticles) && granted.Has(perms.ViewArticleComments), nil
}

// The places offered are the categories a reader could browse to, plus the
// page discussions as one place of their own.
func forumPlacesAPI(env module.Env, _ map[string]string) (wikijson.Object, error) {
	out := wikijson.Array{}
	subject, err := env.Data.Subject(env.User)
	if err != nil {
		return nil, err
	}
	sections, err := env.Data.ForumSections()
	if err != nil {
		return nil, err
	}
	searchable, err := searchableSections(env, subject)
	if err != nil {
		return nil, err
	}
	names := map[int64]string{}
	for _, s := range sections {
		if slices.Contains(searchable, s.ID) {
			names[s.ID] = s.Name
		}
	}
	categories, err := env.Data.ForumCategories()
	if err != nil {
		return nil, err
	}
	for _, c := range categories {
		section, ok := names[c.SectionID]
		if !ok || c.IsForComments {
			continue
		}
		out = append(out, wikijson.Object{
			{Key: "value", Value: strconv.FormatInt(c.ID, 10)},
			{Key: "name", Value: c.Name},
			{Key: "group", Value: section},
		})
	}
	out = append(out, wikijson.Object{{Key: "value", Value: forumCommentsScope}, {Key: "name", Value: env.Text("search-forum-comments")}, {Key: "group", Value: ""}})
	return wikijson.Object{{Key: "places", Value: out}}, nil
}
