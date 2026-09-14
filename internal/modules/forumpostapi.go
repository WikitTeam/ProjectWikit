package modules

import (
	"errors"
	"strconv"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

func init() {
	module.RegisterAPI("forumpost", "fetch", forumPostFetch)
	module.RegisterAPI("forumpost", "fetchversions", forumPostVersions)
	module.RegisterAPI("forumthread", "for_article", forumThreadForArticle)
	module.RegisterAPI("forumnewpost", "preview", forumPostPreview)
}

func forumPostPreview(env module.Env, params map[string]string) (wikijson.Object, error) {
	source, ok := params["source"]
	if !ok {
		return nil, &module.Error{Message: env.Text("module-forumpost-no-source")}
	}
	html, err := env.RenderMessage(source)
	if err != nil {
		return nil, err
	}
	return wikijson.Object{{Key: "result", Value: html}}, nil
}

func forumThreadForArticle(env module.Env, _ map[string]string) (wikijson.Object, error) {
	if env.Page == nil || env.Page.Article == nil {
		return nil, &module.Error{Message: env.Text("module-forumthread-no-page")}
	}
	info, err := env.Data.CommentInfo(env.Page.Article.ID)
	if err != nil {
		return nil, err
	}
	return wikijson.Object{{Key: "threadId", Value: idOrNothing(info.ThreadID)}}, nil
}

func forumPostFetch(env module.Env, params map[string]string) (wikijson.Object, error) {
	post, err := readablePost(env, params)
	if err != nil {
		return nil, err
	}
	at, err := postDate(params)
	if err != nil {
		return nil, err
	}
	source, err := env.Data.ForumPostSource(post.ID, at)
	if err != nil {
		return nil, err
	}
	content, err := env.RenderMessage(source)
	if err != nil {
		return nil, err
	}
	return wikijson.Object{
		{Key: "postId", Value: post.ID},
		{Key: "createdAt", Value: isoTime(post.CreatedAt)},
		{Key: "updatedAt", Value: isoTime(post.UpdatedAt)},
		{Key: "name", Value: post.Name},
		{Key: "source", Value: source},
		{Key: "content", Value: content},
	}, nil
}

func forumPostVersions(env module.Env, params map[string]string) (wikijson.Object, error) {
	post, err := readablePost(env, params)
	if err != nil {
		return nil, err
	}
	found, err := env.Data.ForumPostVersions(post.ID)
	if err != nil {
		return nil, err
	}

	out := make(wikijson.Array, 0, len(found))
	for _, version := range found {
		author, err := postVersionAuthor(env, version.AuthorID)
		if err != nil {
			return nil, err
		}
		out = append(out, wikijson.Object{
			{Key: "createdAt", Value: isoTime(version.CreatedAt)},
			{Key: "author", Value: author},
		})
	}
	return wikijson.Object{{Key: "versions", Value: out}}, nil
}

func postVersionAuthor(env module.Env, id *int64) (wikijson.Object, error) {
	if id == nil {
		return env.Data.UserJSON(nil)
	}
	user, err := env.Data.UserByID(*id)
	if err != nil {
		return nil, err
	}
	return env.Data.UserJSON(user)
}

func readablePost(env module.Env, params map[string]string) (*db.ForumThreadPost, error) {
	post, err := postByParam(env, params)
	if err != nil {
		return nil, err
	}
	allowed, err := postPermissions(env, post)
	if err != nil {
		return nil, err
	}
	if !allowed.Has(perms.ViewForumPosts) {
		return nil, &module.Error{Message: env.Text("module-forumpost-cannot-view")}
	}
	return post, nil
}

func postByParam(env module.Env, params map[string]string) (*db.ForumThreadPost, error) {
	id, err := strconv.ParseInt(params["postid"], 10, 64)
	if err != nil {
		return nil, &module.Error{Message: env.Text("module-forumpost-missing", "id", params["postid"])}
	}
	post, err := env.Data.ForumPost(id)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return nil, err
	}
	if post == nil {
		return nil, &module.Error{Message: env.Text("module-forumpost-missing", "id", params["postid"])}
	}
	return post, nil
}

func postPermissions(env module.Env, post *db.ForumThreadPost) (perms.Set, error) {
	subject, err := env.Data.Subject(env.User)
	if err != nil {
		return perms.Set{}, err
	}
	thread, err := env.Data.ForumThread(post.ThreadID)
	if err != nil {
		return perms.Set{}, err
	}
	object, err := env.Data.ForumThreadObject(thread, env.User)
	if err != nil {
		return perms.Set{}, err
	}
	return perms.Resolve(subject, env.Data.ForumPostObject(post, object, env.User)), nil
}

// A date the caller could not parse asks for the newest version, which is what
// an absent one asks for too.
func postDate(params map[string]string) (*time.Time, error) {
	raw, ok := params["atdate"]
	if !ok || raw == "" {
		return nil, nil
	}
	at, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, nil
	}
	return &at, nil
}

func idOrNothing(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

func isoTime(at time.Time) string {
	return at.UTC().Format("2006-01-02T15:04:05.999999-07:00")
}
