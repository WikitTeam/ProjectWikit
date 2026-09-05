package webapi

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

const AllArticlesPath = "/pw-api/articles"

type AllArticles struct {
	deps     Deps
	upstream http.Handler
}

var _ http.Handler = (*AllArticles)(nil)

func NewAllArticles(d Deps, upstream http.Handler) *AllArticles {
	return &AllArticles{deps: d, upstream: upstream}
}

func (h *AllArticles) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != AllArticlesPath || r.Method != http.MethodGet {
		h.upstream.ServeHTTP(w, r)
		return
	}
	ctx := r.Context()
	loc := h.deps.Bundle.Localizer(i18n.DefaultLanguage)
	current := site.FromContext(ctx)
	if current == nil {
		h.deps.log().Error("all articles", "err", errors.New("the request carries no site"))
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}

	body, err := h.listing(ctx, current, auth.FromContext(ctx))
	if err != nil {
		h.deps.log().Error("list articles", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func (h *AllArticles) listing(ctx context.Context, current *db.Site, viewer *db.User) (string, error) {
	hidden, err := repo.HiddenCategories(ctx, h.deps.DB, viewer)
	if err != nil {
		return "", err
	}
	articles, err := h.deps.DB.ListArticles(ctx, db.ListFilter{}, 0, nil)
	if err != nil {
		return "", err
	}
	kept := make([]db.Article, 0, len(articles))
	ids := make([]int64, 0, len(articles))
	for i := range articles {
		if slices.Contains(hidden, articles[i].Category) {
			continue
		}
		kept = append(kept, articles[i])
		ids = append(ids, articles[i].ID)
	}

	authors, err := h.deps.DB.AuthorsOfArticles(ctx, ids)
	if err != nil {
		return "", err
	}
	editors, err := h.deps.DB.LatestEditorsOfArticles(ctx, ids)
	if err != nil {
		return "", err
	}
	tags, err := h.deps.DB.TagsOfArticles(ctx, ids)
	if err != nil {
		return "", err
	}
	votes, err := h.deps.DB.VoteStatsOfArticles(ctx, ids)
	if err != nil {
		return "", err
	}
	siteMode, err := h.siteMode(ctx, current)
	if err != nil {
		return "", err
	}
	categoryModes, err := h.deps.DB.CategoryRatingModes(ctx)
	if err != nil {
		return "", err
	}

	rendered := map[int64]wikijson.Object{}
	userJSON := func(u *db.User) (wikijson.Object, error) {
		if u == nil {
			return repo.UserJSON(ctx, h.deps.DB, nil)
		}
		if known, ok := rendered[u.ID]; ok {
			return known, nil
		}
		one, err := repo.UserJSON(ctx, h.deps.DB, u)
		if err != nil {
			return nil, err
		}
		rendered[u.ID] = one
		return one, nil
	}

	out := make(wikijson.Array, 0, len(kept))
	for i := range kept {
		article := &kept[i]

		written := make(wikijson.Array, 0, 1)
		for _, one := range authors[article.ID] {
			body, err := userJSON(&one)
			if err != nil {
				return "", err
			}
			written = append(written, body)
		}
		if len(written) == 0 {
			body, err := userJSON(nil)
			if err != nil {
				return "", err
			}
			written = append(written, body)
		}
		var editor wikijson.Object
		if known, ok := editors[article.ID]; ok {
			editor, err = userJSON(&known)
		} else {
			editor, err = userJSON(nil)
		}
		if err != nil {
			return "", err
		}

		names := make([]string, 0, len(tags[article.ID]))
		for _, tag := range tags[article.ID] {
			names = append(names, strings.ToLower(tag.FullName()))
		}
		slices.Sort(names)

		rating := page.RatingOf(page.RatingMode(siteMode, categoryModes[article.Category]), votes[article.ID])
		out = append(out, wikijson.Object{
			{Key: "uid", Value: article.ID},
			{Key: "pageId", Value: article.FullName()},
			{Key: "title", Value: article.Title},
			{Key: "canonicalUrl", Value: "//" + current.Domain + "/" + article.FullName()},
			{Key: "createdAt", Value: isoTime(article.CreatedAt)},
			{Key: "updatedAt", Value: isoTime(article.UpdatedAt)},
			{Key: "createdBy", Value: written[0]},
			{Key: "updatedBy", Value: editor},
			{Key: "authors", Value: written},
			{Key: "rating", Value: wikijson.Object{
				{Key: "value", Value: rating.Value},
				{Key: "votes", Value: rating.Votes},
				{Key: "popularity", Value: rating.Popularity},
				{Key: "mode", Value: rating.Mode},
			}},
			{Key: "tags", Value: names},
		})
	}
	return wikijson.Marshal(out)
}

func (h *AllArticles) siteMode(ctx context.Context, current *db.Site) (string, error) {
	mode, err := h.deps.DB.SiteRatingMode(ctx, current.ID)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return "", err
	}
	return mode, nil
}
