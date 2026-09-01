package pageconfig

import (
	"strconv"

	"github.com/WikitTeam/ProjectWikit/internal/article"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

type Options struct {
	PageID         string
	NormalizedName string
	HasArticle     bool
	Anonymous      bool

	Perms  perms.Set
	Rating page.Rating

	PathParams      article.Params
	CommentCount    int
	CommentThreadID int64
	CommentSlug     string
	CanCreateTags   bool
	IsWatching      bool
	Preferences     Preferences
}

// Preferences is every preference the registry carries, which today is one. The
// frontend reads them under the section__name key.
type Preferences struct {
	AdvancedSourceEditor bool
}

const (
	PreferenceSection              = "qol"
	PreferenceAdvancedSourceEditor = "advanced_source_editor_enabled"
	preferenceKey                  = PreferenceSection + "__" + PreferenceAdvancedSourceEditor
)

func PreferenceEnabled(raw string) bool { return raw == "True" }

func (o Options) JSON() (string, error) {
	return wikijson.Marshal(wikijson.Object{
		{Key: "optionsEnabled", Value: o.HasArticle},
		{Key: "editable", Value: o.Perms.Has(perms.EditArticles)},
		{Key: "lockable", Value: o.Perms.Has(perms.LockArticles)},
		{Key: "tagable", Value: o.Perms.Has(perms.TagArticles)},
		{Key: "pageId", Value: o.PageID},
		{Key: "rating", Value: o.Rating.Value},
		{Key: "ratingMode", Value: o.Rating.Mode},
		{Key: "ratingVotes", Value: o.Rating.Votes},
		{Key: "ratingPopularity", Value: o.Rating.Popularity},
		{Key: "pathParams", Value: pathParams(o.PathParams)},
		{Key: "canRate", Value: o.Perms.Has(perms.RateArticles)},
		{Key: "canComment", Value: o.HasArticle && o.Perms.Has(perms.CommentArticles)},
		{Key: "canViewComments", Value: o.HasArticle && o.Perms.Has(perms.ViewArticleComments)},
		{Key: "commentThread", Value: o.commentThread()},
		{Key: "commentCount", Value: o.CommentCount},
		{Key: "canDelete", Value: o.Perms.Has(perms.DeleteArticles)},
		{Key: "canCreateTags", Value: o.CanCreateTags},
		{Key: "canManageFiles", Value: o.Perms.Has(perms.ManageArticleFiles)},
		{Key: "canRename", Value: o.Perms.Has(perms.MoveArticles)},
		{Key: "canCreateHere", Value: o.Perms.Has(perms.CreateArticles)},
		{Key: "canManageAuthors", Value: o.Perms.Has(perms.ManageArticleAuthors)},
		{Key: "canResetVotes", Value: o.Perms.Has(perms.ResetArticleVotes)},
		{Key: "canWatch", Value: !o.Anonymous},
		{Key: "preferences", Value: o.preferences()},
		{Key: "isWatching", Value: o.IsWatching},
	})
}

// Linking the thread directly is what wikidot does. A page whose thread was
// never opened has no id to link, so it goes through the path that opens one.
func (o Options) commentThread() any {
	if !o.HasArticle {
		return nil
	}
	if o.CommentThreadID == 0 {
		return "/" + o.NormalizedName + "/comments/show"
	}
	return "/forum/t-" + strconv.FormatInt(o.CommentThreadID, 10) + "/" + o.CommentSlug
}

func (o Options) preferences() wikijson.Object {
	if o.Anonymous {
		return wikijson.Object{}
	}
	return wikijson.Object{{Key: preferenceKey, Value: o.Preferences.AdvancedSourceEditor}}
}

func pathParams(params article.Params) wikijson.Object {
	out := make(wikijson.Object, 0, len(params))
	for _, param := range params {
		var value any
		if !param.Bare {
			value = param.Value
		}
		out = append(out, wikijson.Field{Key: param.Key, Value: value})
	}
	return out
}
