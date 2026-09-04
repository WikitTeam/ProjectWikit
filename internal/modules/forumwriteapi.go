package modules

import (
	"errors"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/module"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

func init() {
	module.RegisterWriteAPI("forumnewpost", "submit", forumPostSubmit)
	module.RegisterWriteAPI("forumnewthread", "submit", forumThreadSubmit)
	module.RegisterWriteAPI("forumpost", "update", forumPostUpdate)
	module.RegisterWriteAPI("forumpost", "delete", forumPostDelete)
	module.RegisterWriteAPI("forumthread", "update", forumThreadUpdate)
}

// A reply chain is walked no further than this, which caps how many people one
// post can reach.
const replyDepth = 100

func forumPostSubmit(env module.Env, params map[string]string) (wikijson.Object, error) {
	title := strings.TrimSpace(params["name"])
	source := strings.TrimSpace(params["source"])
	if source == "" {
		return nil, &module.Error{Message: env.Text("module-forumpost-no-source")}
	}
	thread, err := threadByParam(env, params["threadid"])
	if err != nil {
		return nil, err
	}

	var replyTo *db.ForumThreadPost
	if id, err := strconv.ParseInt(params["replyto"], 10, 64); err == nil {
		found, err := env.Data.ForumPost(id)
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			return nil, err
		}
		if found != nil && found.ThreadID != thread.ID {
			return nil, &module.Error{Message: env.Text("module-forumpost-other-thread")}
		}
		replyTo = found
	}

	allowed, err := threadPermissions(env, thread)
	if err != nil {
		return nil, err
	}
	if !allowed.Has(perms.CreateForumPosts) && !allowed.Has(perms.CommentArticles) {
		return nil, &module.Error{Message: env.Text("module-forumpost-cannot-create")}
	}

	var authorID *int64
	if env.User != nil {
		authorID = &env.User.ID
	}
	var replyID *int64
	if replyTo != nil {
		replyID = &replyTo.ID
	}
	id, err := env.Data.CreateForumPost(db.ForumPostWrite{
		ThreadID:  thread.ID,
		Name:      title,
		Source:    source,
		AuthorID:  authorID,
		ReplyToID: replyID,
		At:        time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}

	post, err := env.Data.ForumPost(id)
	if err != nil {
		return nil, err
	}
	if err := announcePost(env, thread, post, title, source, replyTo); err != nil {
		return nil, err
	}
	return wikijson.Object{{Key: "postId", Value: id}}, nil
}

func forumThreadSubmit(env module.Env, params map[string]string) (wikijson.Object, error) {
	title := strings.TrimSpace(params["name"])
	source := strings.TrimSpace(params["source"])
	description := strings.TrimSpace(params["description"])
	if len(description) > 1000 {
		description = description[:1000]
	}
	if title == "" {
		return nil, &module.Error{Message: env.Text("module-forumthread-no-title")}
	}
	if source == "" {
		return nil, &module.Error{Message: env.Text("module-forumthread-no-source")}
	}

	category, err := categoryByParam(env, params["categoryid"])
	if err != nil {
		return nil, err
	}
	subject, err := env.Data.Subject(env.User)
	if err != nil {
		return nil, err
	}
	if !perms.Resolve(subject, nil).Has(perms.CreateForumThreads) {
		return nil, &module.Error{Message: env.Text("module-forumthread-cannot-create")}
	}

	var authorID *int64
	if env.User != nil {
		authorID = &env.User.ID
	}
	threadID, postID, err := env.Data.CreateForumThread(db.ForumThreadWrite{
		CategoryID:  category.ID,
		Name:        title,
		Description: description,
		AuthorID:    authorID,
		At:          time.Now().UTC(),
	}, source)
	if err != nil {
		return nil, err
	}
	if env.User != nil {
		if err := env.Data.SubscribeToThread(env.User.ID, threadID); err != nil {
			return nil, err
		}
	}

	thread, err := env.Data.ForumThread(threadID)
	if err != nil {
		return nil, err
	}
	post, err := env.Data.ForumPost(postID)
	if err != nil {
		return nil, err
	}
	if err := announcePost(env, thread, post, title, source, nil); err != nil {
		return nil, err
	}
	return wikijson.Object{
		{Key: "threadId", Value: threadID},
		{Key: "postId", Value: postID},
	}, nil
}

func forumPostUpdate(env module.Env, params map[string]string) (wikijson.Object, error) {
	title := strings.TrimSpace(params["name"])
	source := strings.TrimSpace(params["source"])
	if source == "" {
		return nil, &module.Error{Message: env.Text("module-forumpost-no-source")}
	}
	post, err := postByParam(env, params)
	if err != nil {
		return nil, err
	}
	allowed, err := postPermissions(env, post)
	if err != nil {
		return nil, err
	}
	if !allowed.Has(perms.EditForumPosts) {
		return nil, &module.Error{Message: env.Text("module-forumpost-cannot-edit")}
	}

	previous, err := env.Data.ForumPostSource(post.ID, nil)
	if err != nil {
		return nil, err
	}
	var authorID *int64
	if env.User != nil {
		authorID = &env.User.ID
	}
	if err := env.Data.UpdateForumPost(post.ID, title, source, previous, authorID); err != nil {
		return nil, err
	}

	fresh, err := env.Data.ForumPost(post.ID)
	if err != nil {
		return nil, err
	}
	thread, err := env.Data.ForumThread(post.ThreadID)
	if err != nil {
		return nil, err
	}
	if err := announceMentions(env, thread, fresh, title, source, previous); err != nil {
		return nil, err
	}

	content, err := env.RenderMessage(source)
	if err != nil {
		return nil, err
	}
	return wikijson.Object{
		{Key: "postId", Value: fresh.ID},
		{Key: "name", Value: fresh.Name},
		{Key: "createdAt", Value: isoTime(fresh.CreatedAt)},
		{Key: "updatedAt", Value: isoTime(fresh.UpdatedAt)},
		{Key: "source", Value: source},
		{Key: "content", Value: content},
	}, nil
}

func forumPostDelete(env module.Env, params map[string]string) (wikijson.Object, error) {
	post, err := postByParam(env, params)
	if err != nil {
		return nil, err
	}
	allowed, err := postPermissions(env, post)
	if err != nil {
		return nil, err
	}
	if !allowed.Has(perms.DeleteForumPosts) {
		return nil, &module.Error{Message: env.Text("module-forumpost-cannot-delete")}
	}
	if err := env.Data.DeleteForumPost(post.ID); err != nil {
		return nil, err
	}
	return wikijson.Object{{Key: "status", Value: "ok"}}, nil
}

func forumThreadUpdate(env module.Env, params map[string]string) (wikijson.Object, error) {
	thread, err := threadByParam(env, params["threadid"])
	if err != nil {
		return nil, err
	}
	allowed, err := threadPermissions(env, thread)
	if err != nil {
		return nil, err
	}

	var edit db.ForumThreadEdit
	if name, ok := params["name"]; ok {
		if strings.TrimSpace(name) == "" {
			return nil, &module.Error{Message: env.Text("module-forumthread-no-title")}
		}
		if !allowed.Has(perms.EditForumThreads) {
			return nil, &module.Error{Message: env.Text("module-forumthread-cannot-edit")}
		}
		edit.Name = &name
	}
	if description, ok := params["description"]; ok {
		if !allowed.Has(perms.EditForumThreads) {
			return nil, &module.Error{Message: env.Text("module-forumthread-cannot-edit")}
		}
		edit.Description = &description
	}
	if raw, ok := params["islocked"]; ok {
		if !allowed.Has(perms.LockForumThreads) {
			return nil, &module.Error{Message: env.Text("module-forumthread-cannot-lock")}
		}
		locked := truthy(raw)
		edit.Locked = &locked
	}
	if raw, ok := params["ispinned"]; ok {
		if !allowed.Has(perms.PinForumThreads) {
			return nil, &module.Error{Message: env.Text("module-forumthread-cannot-pin")}
		}
		pinned := truthy(raw)
		edit.Pinned = &pinned
	}
	if raw, ok := params["categoryid"]; ok {
		if !allowed.Has(perms.MoveForumThreads) {
			return nil, &module.Error{Message: env.Text("module-forumthread-cannot-move")}
		}
		category, err := categoryByParam(env, raw)
		if err != nil {
			return nil, err
		}
		edit.CategoryID = &category.ID
	}

	if err := env.Data.UpdateForumThread(thread.ID, edit); err != nil {
		return nil, err
	}
	fresh, err := env.Data.ForumThread(thread.ID)
	if err != nil {
		return nil, err
	}
	return wikijson.Object{
		{Key: "threadId", Value: fresh.ID},
		{Key: "name", Value: fresh.Name},
		{Key: "description", Value: fresh.Description},
		{Key: "isLocked", Value: fresh.IsLocked},
		{Key: "isPinned", Value: fresh.IsPinned},
		{Key: "categoryId", Value: pointedID(fresh.CategoryID)},
	}, nil
}

// The call arrives with its values already flattened to text, so a boolean gets
// here spelled out and has to be read back rather than tested for emptiness.
func pointedID(id *int64) any {
	if id == nil {
		return nil
	}
	return *id
}

func truthy(raw string) bool {
	return raw != "" && raw != "false" && raw != "0"
}

func threadByParam(env module.Env, raw string) (*db.ForumThread, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, &module.Error{Message: env.Text("module-forumthread-not-given")}
	}
	thread, err := env.Data.ForumThread(id)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return nil, err
	}
	if thread == nil {
		return nil, &module.Error{Message: env.Text("module-forumthread-not-given")}
	}
	return thread, nil
}

func categoryByParam(env module.Env, raw string) (*db.ForumCategory, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, &module.Error{Message: env.Text("module-forumcategory-not-given")}
	}
	category, err := env.Data.ForumCategory(id)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return nil, err
	}
	if category == nil {
		return nil, &module.Error{Message: env.Text("module-forumcategory-not-given")}
	}
	return category, nil
}

func threadPermissions(env module.Env, thread *db.ForumThread) (perms.Set, error) {
	subject, err := env.Data.Subject(env.User)
	if err != nil {
		return perms.Set{}, err
	}
	object, err := env.Data.ForumThreadObject(thread, env.User)
	if err != nil {
		return perms.Set{}, err
	}
	return perms.Resolve(subject, object), nil
}

var notifyMention = regexp.MustCompile(`@([\p{L}\p{N}_.-]+)`)

func mentionedNames(source string) []string {
	var out []string
	for _, m := range notifyMention.FindAllStringSubmatch(source, -1) {
		name := strings.ToLower(m[1])
		if !slices.Contains(out, name) {
			out = append(out, name)
		}
	}
	return out
}

// The post reaches three sets of people, and nobody hears about it twice. Whoever
// is up the reply chain wins over whoever merely watches the thread.
func announcePost(env module.Env, thread *db.ForumThread, post *db.ForumThreadPost,
	title, source string, replyTo *db.ForumThreadPost) error {

	meta, err := postMeta(env, thread, post, title, source)
	if err != nil {
		return err
	}
	watchers, err := env.Data.ThreadSubscribers(thread.ID)
	if err != nil {
		return err
	}
	watchers = withoutAuthor(env, watchers)

	if replyTo != nil {
		chain, err := replyChain(env, replyTo)
		if err != nil {
			return err
		}
		chain = withoutAuthor(env, chain)
		watchers = without(watchers, chain)

		chain, err = readersOfPost(env, thread, post, chain)
		if err != nil {
			return err
		}
		origin, err := replyMeta(env, thread, replyTo, meta)
		if err != nil {
			return err
		}
		encoded, err := wikijson.Marshal(origin)
		if err != nil {
			return err
		}
		if err := env.Data.SendNotification(db.NotifyNewPostReply, encoded, chain); err != nil {
			return err
		}
	}

	watchers, err = readersOfPost(env, thread, post, watchers)
	if err != nil {
		return err
	}
	encoded, err := wikijson.Marshal(meta)
	if err != nil {
		return err
	}
	if err := env.Data.SendNotification(db.NotifyNewThreadPost, encoded, watchers); err != nil {
		return err
	}
	return announceMentions(env, thread, post, title, source, "")
}

// Editing a post only reaches names it did not already carry, so fixing a typo
// does not tell everyone again.
func announceMentions(env module.Env, thread *db.ForumThread, post *db.ForumThreadPost,
	title, source, previous string) error {

	names := mentionedNames(source)
	if previous != "" {
		names = without2(names, mentionedNames(previous))
	}
	if len(names) == 0 {
		return nil
	}
	users, err := env.Data.ActiveUsersByNames(names)
	if err != nil {
		return err
	}
	ids := make([]int64, 0, len(users))
	for i := range users {
		if env.User != nil && users[i].ID == env.User.ID {
			continue
		}
		ids = append(ids, users[i].ID)
	}
	ids, err = readersOfPost(env, thread, post, ids)
	if err != nil {
		return err
	}
	meta, err := postMeta(env, thread, post, title, source)
	if err != nil {
		return err
	}
	encoded, err := wikijson.Marshal(meta)
	if err != nil {
		return err
	}
	return env.Data.SendNotification(db.NotifyForumMention, encoded, ids)
}

func replyChain(env module.Env, from *db.ForumThreadPost) ([]int64, error) {
	var out []int64
	post := from
	for i := 0; i < replyDepth && post != nil; i++ {
		if post.AuthorID != nil && !slices.Contains(out, *post.AuthorID) {
			out = append(out, *post.AuthorID)
		}
		if post.ReplyToID == nil {
			break
		}
		next, err := env.Data.ForumPost(*post.ReplyToID)
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			return nil, err
		}
		post = next
	}
	return out, nil
}

func readersOfPost(env module.Env, thread *db.ForumThread, post *db.ForumThreadPost,
	ids []int64) ([]int64, error) {

	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		reader, err := env.Data.UserByID(id)
		if err != nil {
			return nil, err
		}
		subject, err := env.Data.Subject(reader)
		if err != nil {
			return nil, err
		}
		object, err := env.Data.ForumThreadObject(thread, reader)
		if err != nil {
			return nil, err
		}
		allowed := perms.Resolve(subject, env.Data.ForumPostObject(post, object, reader))
		if allowed.Has(perms.ViewForumPosts) {
			out = append(out, id)
		}
	}
	return out, nil
}

func withoutAuthor(env module.Env, ids []int64) []int64 {
	if env.User == nil {
		return ids
	}
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id != env.User.ID {
			out = append(out, id)
		}
	}
	return out
}

func without(ids, drop []int64) []int64 {
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if !slices.Contains(drop, id) {
			out = append(out, id)
		}
	}
	return out
}

func without2(names, drop []string) []string {
	out := make([]string, 0, len(names))
	for _, name := range names {
		if !slices.Contains(drop, name) {
			out = append(out, name)
		}
	}
	return out
}

func postMeta(env module.Env, thread *db.ForumThread, post *db.ForumThreadPost,
	title, source string) (wikijson.Object, error) {

	category, section, err := threadHome(env, thread)
	if err != nil {
		return nil, err
	}
	name, err := threadTitle(env, thread)
	if err != nil {
		return nil, err
	}
	threadURL := "/forum/t-" + strconv.FormatInt(thread.ID, 10) + "/" + wikidot.Normalize(name)

	author, err := env.Data.UserJSON(env.User)
	if err != nil {
		return nil, err
	}
	return wikijson.Object{
		{Key: "thread", Value: named(thread.ID, name, threadURL)},
		{Key: "category", Value: named(category.ID, category.Name,
			"/forum/c-"+strconv.FormatInt(category.ID, 10)+"/"+wikidot.Normalize(category.Name))},
		{Key: "section", Value: named(section.ID, section.Name,
			"/forum/s-"+strconv.FormatInt(section.ID, 10)+"/"+wikidot.Normalize(section.Name))},
		{Key: "post", Value: named(post.ID, title,
			threadURL+"#post-"+strconv.FormatInt(post.ID, 10))},
		{Key: "author", Value: author},
		{Key: "message_source", Value: source},
	}, nil
}

func replyMeta(env module.Env, thread *db.ForumThread, replyTo *db.ForumThreadPost,
	base wikijson.Object) (wikijson.Object, error) {

	name, err := threadTitle(env, thread)
	if err != nil {
		return nil, err
	}
	threadURL := "/forum/t-" + strconv.FormatInt(thread.ID, 10) + "/" + wikidot.Normalize(name)
	origin := named(replyTo.ID, replyTo.Name,
		threadURL+"#post-"+strconv.FormatInt(replyTo.ID, 10))
	return append(slices.Clone(base), wikijson.Field{Key: "origin", Value: origin}), nil
}

// A comment thread carries the page's title rather than a name of its own.
func threadTitle(env module.Env, thread *db.ForumThread) (string, error) {
	if thread.CategoryID != nil || thread.ArticleID == nil {
		return thread.Name, nil
	}
	article, err := env.Data.ArticleByID(*thread.ArticleID)
	if err != nil {
		return "", err
	}
	return article.Title, nil
}

func named(id int64, name, url string) wikijson.Object {
	return wikijson.Object{
		{Key: "id", Value: id},
		{Key: "name", Value: name},
		{Key: "url", Value: url},
	}
}

// A comment thread has no category of its own, so the first one that takes
// comments stands in for it.
func threadHome(env module.Env, thread *db.ForumThread) (*db.ForumCategory, *db.ForumSection, error) {
	var category *db.ForumCategory
	if thread.CategoryID != nil {
		found, err := env.Data.ForumCategory(*thread.CategoryID)
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			return nil, nil, err
		}
		category = found
	}
	if category == nil {
		all, err := env.Data.ForumCategories()
		if err != nil {
			return nil, nil, err
		}
		for i := range all {
			if all[i].IsForComments {
				category = &all[i]
				break
			}
		}
	}
	if category == nil {
		return nil, nil, &module.Error{Message: env.Text("module-forumcategory-not-given")}
	}
	section, err := env.Data.ForumSection(category.SectionID)
	if err != nil {
		return nil, nil, err
	}
	return category, section, nil
}
