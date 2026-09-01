package repo

import (
	"context"
	"errors"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/form"
)

const templateName = "_template"

// A listing renders one page per row and every row asks its category the same
// question, so the answer is kept for the rest of the request.
type formLoader struct {
	ctx   context.Context
	db    *db.DB
	known map[string]*form.Definition
}

func newFormLoader(ctx context.Context, d *db.DB) *formLoader {
	return &formLoader{ctx: ctx, db: d, known: map[string]*form.Definition{}}
}

// A category with no template, no revision, or no form block is an ordinary
// category, which is why none of those is an error.
func (l *formLoader) CategoryForm(category string) (*form.Definition, error) {
	if def, ok := l.known[category]; ok {
		return def, nil
	}
	def, err := l.load(category)
	if err != nil {
		return nil, err
	}
	l.known[category] = def
	return def, nil
}

func (l *formLoader) load(category string) (*form.Definition, error) {
	article, err := l.db.ArticleByName(l.ctx, category+":"+templateName)
	if errors.Is(err, db.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	source, err := l.db.LatestSource(l.ctx, article.ID)
	if errors.Is(err, db.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	def, found, err := form.Parse(source)
	if err != nil || !found {
		return nil, err
	}
	return def, nil
}
