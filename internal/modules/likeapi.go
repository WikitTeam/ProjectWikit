package modules

import (
	"strconv"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

const likersPerPage = 20

func init() {
	module.RegisterWriteAPI("forumpost", "like", forumPostLike)
	module.RegisterWriteAPI("forumpost", "unlike", forumPostUnlike)
	module.RegisterAPI("forumpost", "likes", forumPostLikes)
}

func forumPostLike(env module.Env, params map[string]string) (wikijson.Object, error) {
	post, err := likeablePost(env, params)
	if err != nil {
		return nil, err
	}
	fresh, err := env.Data.LikePost(post.ID, env.User.ID)
	if err != nil {
		return nil, err
	}
	if fresh {
		if err := tellTheLikedAuthor(env, post); err != nil {
			return nil, err
		}
	}
	return likeState(env, post.ID, true)
}

func forumPostUnlike(env module.Env, params map[string]string) (wikijson.Object, error) {
	post, err := likeablePost(env, params)
	if err != nil {
		return nil, err
	}
	if err := env.Data.UnlikePost(post.ID, env.User.ID); err != nil {
		return nil, err
	}
	return likeState(env, post.ID, false)
}

func forumPostLikes(env module.Env, params map[string]string) (wikijson.Object, error) {
	post, err := readablePost(env, params)
	if err != nil {
		return nil, err
	}
	total, err := env.Data.PostLikeCount(post.ID)
	if err != nil {
		return nil, err
	}

	page := 1
	if n, err := strconv.Atoi(params["page"]); err == nil && n > 0 {
		page = n
	}
	pages := (total + likersPerPage - 1) / likersPerPage
	if pages == 0 {
		pages = 1
	}
	if page > pages {
		page = pages
	}

	users, err := env.Data.PostLikers(post.ID, (page-1)*likersPerPage, likersPerPage)
	if err != nil {
		return nil, err
	}
	rendered := make(wikijson.Array, 0, len(users))
	for i := range users {
		one, err := env.Data.UserJSON(&users[i])
		if err != nil {
			return nil, err
		}
		rendered = append(rendered, one)
	}
	return wikijson.Object{
		{Key: "postId", Value: post.ID},
		{Key: "count", Value: total},
		{Key: "page", Value: page},
		{Key: "pages", Value: pages},
		{Key: "perPage", Value: likersPerPage},
		{Key: "users", Value: rendered},
	}, nil
}

func likeablePost(env module.Env, params map[string]string) (*db.ForumThreadPost, error) {
	if env.User == nil {
		return nil, &module.Error{Message: env.Text("forum-like-signed-out")}
	}
	return readablePost(env, params)
}

func likeState(env module.Env, postID int64, liked bool) (wikijson.Object, error) {
	count, err := env.Data.PostLikeCount(postID)
	if err != nil {
		return nil, err
	}
	return wikijson.Object{
		{Key: "postId", Value: postID},
		{Key: "count", Value: count},
		{Key: "liked", Value: liked},
	}, nil
}

// Liking your own post tells nobody, and neither does liking one that has no
// author left.
func tellTheLikedAuthor(env module.Env, post *db.ForumThreadPost) error {
	if post.AuthorID == nil || *post.AuthorID == env.User.ID {
		return nil
	}
	thread, err := env.Data.ForumThread(post.ThreadID)
	if err != nil {
		return err
	}
	object, err := env.Data.ForumThreadObject(thread, nil)
	if err != nil {
		return err
	}
	reader, err := env.Data.UserByID(*post.AuthorID)
	if err != nil {
		return err
	}
	subject, err := env.Data.Subject(reader)
	if err != nil {
		return err
	}
	if !perms.Resolve(subject, env.Data.ForumPostObject(post, object, reader)).Has(perms.ViewForumPosts) {
		return nil
	}

	name, err := threadTitle(env, thread)
	if err != nil {
		return err
	}
	who, err := env.Data.UserJSON(env.User)
	if err != nil {
		return err
	}
	threadURL := "/forum/t-" + strconv.FormatInt(thread.ID, 10) + "/" + wikidot.Normalize(name)
	meta, err := wikijson.Marshal(wikijson.Object{
		{Key: "thread", Value: named(thread.ID, name, threadURL)},
		{Key: "post", Value: named(post.ID, post.Name,
			threadURL+"#post-"+strconv.FormatInt(post.ID, 10))},
		{Key: "author", Value: who},
	})
	if err != nil {
		return err
	}
	return env.Data.SendNotification(db.NotifyPostLike, meta, []int64{*post.AuthorID})
}
