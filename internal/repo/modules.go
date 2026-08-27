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
