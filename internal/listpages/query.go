package listpages

import (
	"errors"
	"slices"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

const maxPerPage = 250

// PageIndex is what the first row's %%index%% counts from, so it moves with
// the pagination rather than with the offset.
type Result struct {
	Pages      []db.Article
	PageIndex  int
	Page       int
	TotalPages int
	Total      int
}

func Run(src Source, q Query, viewer *db.User, paginate bool) (Result, error) {
	// The cap belongs here rather than in the parsing, because what the reader
	// asked for is what the frontend gets handed back.
	q.PerPage = min(q.PerPage, maxPerPage)
	empty := Result{Page: 1, TotalPages: 1}
	if q.Invalid {
		return empty, nil
	}
	hidden, err := src.HiddenCategories(viewer)
	if err != nil {
		return Result{}, err
	}

	if q.HasOnly {
		if containsFold(hidden, q.Only.Category) {
			return empty, nil
		}
		return Result{Pages: []db.Article{*q.Only}, Page: 1, TotalPages: 1, Total: 1}, nil
	}
	if q.HasFullName {
		found, err := src.ArticleByRef(strings.ToLower(q.FullName))
		if errors.Is(err, db.ErrNotFound) {
			return empty, nil
		}
		if err != nil {
			return Result{}, err
		}
		if containsFold(hidden, found.Category) {
			return empty, nil
		}
		return Result{Pages: []db.Article{*found}, Page: 1, TotalPages: 1, Total: 1}, nil
	}

	filter := q.Filter
	filter.Hidden = hidden

	if q.hasFormWork() {
		return runWithForm(src, q, filter, paginate)
	}

	total, err := src.CountArticles(filter, q.Offset, q.Limit)
	if err != nil {
		return Result{}, err
	}

	if !paginate {
		pages, err := src.ListArticles(filter, q.Offset, q.Limit)
		if err != nil {
			return Result{}, err
		}
		return Result{Pages: pages, Page: 1, TotalPages: 1, Total: total}, nil
	}

	offset, limit, skip := window(q)
	var pages []db.Article
	if !skip {
		pages, err = src.ListArticles(filter, offset, limit)
		if err != nil {
			return Result{}, err
		}
	}
	return Result{
		Pages:      pages,
		PageIndex:  (q.Page - 1) * q.PerPage,
		Page:       q.Page,
		TotalPages: totalPages(total, q.PerPage),
		Total:      total,
	}, nil
}

func window(q Query) (offset int, limit *int, skip bool) {
	if q.PerPage <= 0 {
		return 0, nil, true
	}
	start := (q.Page - 1) * q.PerPage
	offset = q.Offset + start
	perPage := q.PerPage
	if q.Limit == nil {
		return offset, &perPage, false
	}
	remaining := *q.Limit - start
	if remaining <= 0 {
		return offset, nil, true
	}
	bounded := min(perPage, remaining)
	return offset, &bounded, false
}

// A per-page of zero reports no pages at all rather than dividing by zero and
// failing the request.
func totalPages(total, perPage int) int {
	if perPage <= 0 {
		return 0
	}
	return (total + perPage - 1) / perPage
}

func containsFold(values []string, want string) bool {
	return slices.ContainsFunc(values, func(v string) bool { return strings.EqualFold(v, want) })
}
