package repo

import (
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
)

type moduleData struct{ repo *Repository }

func (m moduleData) TagCategory(slug string) (db.TagCategory, error) {
	return m.repo.db.TagCategoryBySlug(m.repo.ctx, slug)
}

func (m moduleData) TagArticles(categorySlug, name string, hidden []string) ([]db.Article, error) {
	return m.repo.db.ArticlesByTag(m.repo.ctx, categorySlug, name, hidden)
}

// HiddenCategories asks the same question once per category, the way the page
// list does. There is no query for it because permissions are resolved in Go.
func (m moduleData) HiddenCategories(user *db.User) ([]string, error) {
	names, err := m.repo.db.CategoryNames(m.repo.ctx)
	if err != nil {
		return nil, err
	}
	resolver := NewPerms(m.repo.ctx, m.repo.db)
	subject, err := resolver.Subject(user, time.Now())
	if err != nil {
		return nil, err
	}

	var hidden []string
	for _, name := range names {
		object, err := resolver.Category(name)
		if err != nil {
			return nil, err
		}
		if !perms.Resolve(subject, object).Has(perms.ViewArticles) {
			hidden = append(hidden, name)
		}
	}
	return hidden, nil
}

func (m moduleData) TagIDsByName(categorySlug, name string) ([]int64, error) {
	return m.repo.db.TagIDsByName(m.repo.ctx, categorySlug, name)
}

func (m moduleData) ArticleTagIDs(articleID int64) ([]int64, error) {
	return m.repo.db.ArticleTagIDs(m.repo.ctx, articleID)
}

func (m moduleData) ArticleByRef(ref string) (*db.Article, error) {
	return m.repo.db.ArticleByName(m.repo.ctx, ref)
}

func (m moduleData) UserByUsername(name string) (*db.User, error) {
	return m.repo.db.UserByUsername(m.repo.ctx, name)
}

func (m moduleData) UserByWikidotName(name string) (*db.User, error) {
	return m.repo.db.UserByWikidotName(m.repo.ctx, name)
}

func (m moduleData) SiteRatingMode() (string, error) {
	if m.repo.opts.Site == nil {
		return "", db.ErrNotFound
	}
	return m.repo.db.SiteRatingMode(m.repo.ctx, m.repo.opts.Site.ID)
}

func (m moduleData) CategoryRatingMode(category string) (string, error) {
	return m.repo.db.CategoryRatingMode(m.repo.ctx, category)
}

func (m moduleData) VoteStats(articleID int64) (db.VoteStats, error) {
	return m.repo.db.VoteStats(m.repo.ctx, articleID)
}

func (m moduleData) ListArticles(f db.ListFilter, offset int, limit *int) ([]db.Article, error) {
	return m.repo.db.ListArticles(m.repo.ctx, f, offset, limit)
}

func (m moduleData) CountArticles(f db.ListFilter, offset int, limit *int) (int, error) {
	return m.repo.db.CountArticles(m.repo.ctx, f, offset, limit)
}
