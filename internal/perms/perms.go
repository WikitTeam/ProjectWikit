// Package perms answers what one user may do to one page or forum object.
package perms

import "sort"

const (
	ViewArticles         = "view_articles"
	RateArticles         = "rate_articles"
	CreateArticles       = "create_articles"
	EditArticles         = "edit_articles"
	TagArticles          = "tag_articles"
	MoveArticles         = "move_articles"
	LockArticles         = "lock_articles"
	ManageArticleFiles   = "manage_article_files"
	DeleteArticles       = "delete_articles"
	ResetArticleVotes    = "reset_article_votes"
	CommentArticles      = "comment_articles"
	ViewArticleComments  = "view_article_comments"
	ManageArticleAuthors = "manage_article_authors"

	ViewForumPosts          = "view_forum_posts"
	CreateForumPosts        = "create_forum_posts"
	EditForumPosts          = "edit_forum_posts"
	DeleteForumPosts        = "delete_forum_posts"
	ViewForumThreads        = "view_forum_threads"
	CreateForumThreads      = "create_forum_threads"
	EditForumThreads        = "edit_forum_threads"
	PinForumThreads         = "pin_forum_threads"
	LockForumThreads        = "lock_forum_threads"
	MoveForumThreads        = "move_forum_threads"
	ViewForumSections       = "view_forum_sections"
	ViewHiddenForumSections = "view_hidden_forum_sections"
	ViewForumCategories     = "view_forum_categories"
)

// lockable is what a locked page takes away from anyone who cannot unlock it.
var lockable = []string{EditArticles, ManageArticleAuthors, ManageArticleFiles, TagArticles, MoveArticles, DeleteArticles}

// Commenting on an article is in this list because an article's comments are
// forum posts underneath.
var silenced = []string{CommentArticles, CreateForumPosts, EditForumPosts, DeleteForumPosts, EditForumThreads, PinForumThreads, MoveForumThreads}

// Set answers one question at a time. A superuser gets a set that says yes to
// every name, which is the shortcut Django takes before any backend runs.
type Set struct {
	all   bool
	named map[string]bool
}

func (s Set) Has(name string) bool { return s.all || s.named[name] }

func (s Set) All() bool { return s.all }

// Names is for tests and for anything that prints a whole answer.
func (s Set) Names() []string {
	out := make([]string, 0, len(s.named))
	for name := range s.named {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

type Role struct {
	ID           int64
	Permissions  []string
	Restrictions []string
}

// Override is one of a category's overrides. A role reads only the first one
// carrying its own id.
type Override struct {
	RoleID       int64
	Permissions  []string
	Restrictions []string
}

// Subject is the user asking. Roles arrive already assembled: the default role
// first, then the registered one and the user's own.
type Subject struct {
	Anonymous   bool
	Active      bool
	ForumActive bool
	Superuser   bool
	Roles       []Role
}

type Kind int

const (
	KindArticle Kind = iota
	KindForumSection
	KindForumThread
	KindForumPost
)

// Object is the thing being asked about. A category fills in Overrides alone,
// which is why the article kind is the zero value.
type Object struct {
	Overrides []Override
	Kind      Kind

	Locked bool
	Author bool

	HiddenForUsers bool

	// Django reaches this through the post's override pipeline, so a thread's
	// rules run after the post's own.
	Thread *Object
}

// Resolve answers for one user against one object. A nil object is the
// site-wide question, which no override reaches.
func Resolve(s Subject, o *Object) Set {
	if s.Active && s.Superuser {
		return Set{all: true}
	}
	// An anonymous visitor is never inactive, so the flag does not apply.
	if !s.Active && !s.Anonymous {
		return Set{named: map[string]bool{}}
	}

	granted := make(map[string]bool)
	for _, role := range s.Roles {
		final := make(map[string]bool, len(role.Permissions))
		for _, name := range role.Permissions {
			final[name] = true
		}
		for _, name := range role.Restrictions {
			delete(final, name)
		}
		if o != nil {
			applyOverride(final, o.Overrides, role.ID)
		}
		for name := range final {
			granted[name] = true
		}
	}
	if o != nil {
		applyObject(granted, o, s)
	}
	return Set{named: granted}
}

func applyOverride(final map[string]bool, overrides []Override, roleID int64) {
	for _, override := range overrides {
		if override.RoleID != roleID {
			continue
		}
		for _, name := range override.Permissions {
			final[name] = true
		}
		for _, name := range override.Restrictions {
			delete(final, name)
		}
		return
	}
}

func applyObject(granted map[string]bool, o *Object, s Subject) {
	switch o.Kind {
	case KindForumSection:
		if o.HiddenForUsers && granted[ViewForumSections] && !granted[ViewHiddenForumSections] {
			delete(granted, ViewForumSections)
		}
		return
	case KindForumThread:
		applyThread(granted, o, s)
		return
	case KindForumPost:
		// The grant reads the site-wide answer rather than what this thread
		// has left, so a locked thread still lets its author keep the post.
		if o.Author && Resolve(s, nil).Has(CreateForumPosts) {
			granted[EditForumPosts] = true
		}
		if o.Thread != nil {
			applyThread(granted, o.Thread, s)
		}
		return
	}
	if o.Locked {
		if !granted[LockArticles] {
			for _, name := range lockable {
				delete(granted, name)
			}
		}
		return
	}
	if o.Author {
		granted[ManageArticleAuthors] = true
	}
}

func applyThread(granted map[string]bool, o *Object, s Subject) {
	if o.Author {
		granted[EditForumThreads] = true
	}
	barred := !s.Anonymous && !s.ForumActive
	locked := o.Locked && !granted[LockForumThreads]
	if !barred && !locked {
		return
	}
	for _, name := range silenced {
		delete(granted, name)
	}
}
