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
		subject.Superuser = u.IsSuperuser
	}
	return subject, nil
}

// Article is the object side for a page that exists. The overrides come from
// its category, which is the only step Django takes before the page's own.
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
