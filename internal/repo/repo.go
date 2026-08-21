// Package repo composes the data layer into the renderer's Repository.
package repo

import (
	"context"
	"errors"

	"github.com/WikitTeam/ProjectWikit/internal/callbacks"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
)

var ErrNotPorted = errors.New("repo: not ported yet")

type Repository struct {
	ctx context.Context
	db  *db.DB
}

var _ callbacks.Repository = (*Repository)(nil)

func New(ctx context.Context, d *db.DB) *Repository {
	return &Repository{ctx: ctx, db: d}
}

func (r *Repository) PageInfo(refs []string) ([]renderer.PartialPageInfo, error) {
	titles, err := r.db.ArticleTitles(r.ctx, refs)
	if err != nil {
		return nil, err
	}
	out := make([]renderer.PartialPageInfo, 0, len(titles))
	for _, ref := range refs {
		title, ok := titles[ref]
		if !ok {
			// fetch_internal_links omits missing pages instead of reporting
			// exists=false; ftml treats an absent entry as a red link.
			continue
		}
		out = append(out, renderer.PartialPageInfo{FullName: ref, Title: &title, Exists: true})
	}
	return out, nil
}

func (r *Repository) IncludeSources(refs []renderer.IncludeRef) ([]renderer.FetchedPage, error) {
	names := make([]string, len(refs))
	for i, ref := range refs {
		names[i] = ref.FullName
	}
	sources, err := r.db.ArticleSources(r.ctx, names)
	if err != nil {
		return nil, err
	}
	out := make([]renderer.FetchedPage, 0, len(refs))
	for _, ref := range refs {
		page := renderer.FetchedPage{FullName: ref.FullName}
		if source, ok := sources[ref.FullName]; ok {
			page.Content = &source
		}
		out = append(out, page)
	}
	return out, nil
}

func (r *Repository) RenderModule(name string, params map[string]string, body string) (string, error) {
	return "", ErrNotPorted
}

func (r *Repository) RenderUser(username string, avatar bool) (string, error) {
	return "", ErrNotPorted
}
