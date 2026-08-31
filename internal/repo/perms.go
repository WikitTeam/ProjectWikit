package repo

import (
	"context"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
)

const (
	slugEveryone   = "everyone"
	slugRegistered = "registered"
)

// Perms assembles what internal/perms needs out of the database.
type Perms struct {
	ctx context.Context
	db  *db.DB
}

func NewPerms(ctx context.Context, d *db.DB) *Perms {
	return &Perms{ctx: ctx, db: d}
}

// Subject collects the roles one visitor carries. A nil user is the anonymous
// visitor, who still holds the default role.
func (p *Perms) Subject(u *db.User, now time.Time) (perms.Subject, error) {
	slugs := []string{slugEveryone}
	if u != nil {
		slugs = append(slugs, slugRegistered)
	}
	bySlug, err := p.db.RoleIDsBySlug(p.ctx, slugs)
	if err != nil {
		return perms.Subject{}, err
	}

	var ids []int64
	if id, ok := bySlug[slugEveryone]; ok {
		ids = append(ids, id)
	}
	if u != nil {
		if id, ok := bySlug[slugRegistered]; ok {
			ids = append(ids, id)
		}
		own, err := p.db.RoleIDsForUser(p.ctx, u.ID)
		if err != nil {
			return perms.Subject{}, err
		}
		ids = append(ids, own...)
	}

	roles, err := p.db.RolePermissions(p.ctx, ids)
	if err != nil {
		return perms.Subject{}, err
	}

	subject := perms.Subject{Anonymous: u == nil, Roles: roles}
	if u != nil {
		subject.Active = u.ActiveAt(now)
		subject.ForumActive = u.ForumActiveAt(now)
		subject.Superuser = u.IsSuperuser
	}
	return subject, nil
}

// Article is the object side for a page that exists. Its category is the only
// step taken before the page's own overrides.
func (p *Perms) Article(a *db.Article, u *db.User) (*perms.Object, error) {
	overrides, err := p.db.CategoryOverrides(p.ctx, a.Category)
	if err != nil {
		return nil, err
	}
	object := &perms.Object{Overrides: overrides, Locked: a.Locked}
	if u != nil {
		author, err := p.db.ArticleHasAuthor(p.ctx, a.ID, u.ID)
		if err != nil {
			return nil, err
		}
		object.Author = author
	}
	return object, nil
}

// Category is the object side for a page that does not exist. A category with
// no row of its own answers the same as no object at all.
func (p *Perms) Category(name string) (*perms.Object, error) {
	overrides, err := p.db.CategoryOverrides(p.ctx, name)
	if err != nil {
		return nil, err
	}
	return &perms.Object{Overrides: overrides}, nil
}

// The forum models sit outside the per-role override table, so nothing but the
// hidden flag reaches the answer.
func (p *Perms) ForumSection(s *db.ForumSection) *perms.Object {
	return &perms.Object{Kind: perms.KindForumSection, HiddenForUsers: s.IsHiddenForUsers}
}

// A thread reads its role overrides through the article it comments on, so a
// thread of its own carries none.
func (p *Perms) ForumThread(t *db.ForumThread, u *db.User) (*perms.Object, error) {
	object := &perms.Object{Kind: perms.KindForumThread, Locked: t.IsLocked}
	if t.ArticleID != nil {
		article, err := p.db.ArticleByID(p.ctx, *t.ArticleID)
		if err != nil {
			return nil, err
		}
		overrides, err := p.db.CategoryOverrides(p.ctx, article.Category)
		if err != nil {
			return nil, err
		}
		object.Overrides = overrides
	}
	if u != nil && t.AuthorID != nil {
		object.Author = *t.AuthorID == u.ID
	}
	return object, nil
}

func (p *Perms) ForumPost(post *db.ForumThreadPost, thread *perms.Object, u *db.User) *perms.Object {
	object := &perms.Object{Kind: perms.KindForumPost, Thread: thread}
	if u != nil && post.AuthorID != nil {
		object.Author = *post.AuthorID == u.ID
	}
	return object
}
