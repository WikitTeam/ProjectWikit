package webapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

const (
	RatingsPath    = "/pw-api/ratings"
	LikedPostsPath = "/pw-api/liked-posts"
)

const ownRowsPerPage = 20

type OwnRows struct {
	deps     Deps
	upstream http.Handler
}

var _ http.Handler = (*OwnRows)(nil)

func NewOwnRows(d Deps, upstream http.Handler) *OwnRows {
	return &OwnRows{deps: d, upstream: upstream}
}

func (h *OwnRows) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet || (r.URL.Path != RatingsPath && r.URL.Path != LikedPostsPath) {
		h.upstream.ServeHTTP(w, r)
		return
	}
	loc := h.deps.Bundle.Localizer(i18n.DefaultLanguage)
	user := auth.FromContext(r.Context())
	if user == nil {
		writeJSON(w, http.StatusForbidden, field("error", loc.T("api-forbidden")))
		return
	}
	if r.URL.Path == RatingsPath {
		h.ratings(w, r, loc, user)
		return
	}
	h.likedPosts(w, r, loc, user)
}

func (h *OwnRows) ratings(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, user *db.User) {
	ctx := r.Context()
	total, err := h.deps.DB.RatedByCountOf(ctx, user.ID)
	if err != nil {
		h.deps.log().Error("count ratings", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	page, pages := pageOf(r, total)

	found, err := h.deps.DB.RatedBy(ctx, user.ID, (page-1)*ownRowsPerPage, ownRowsPerPage)
	if err != nil {
		h.deps.log().Error("list ratings", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}

	rendered := make(wikijson.Array, 0, len(found))
	for _, one := range found {
		title := one.Article.Title
		if title == "" {
			title = one.Article.FullName()
		}
		votedAt := any(nil)
		if one.VotedAt != nil {
			votedAt = isoTime(*one.VotedAt)
		}
		rendered = append(rendered, wikijson.Object{
			{Key: "pageId", Value: one.Article.FullName()},
			{Key: "title", Value: title},
			{Key: "rate", Value: one.Rate},
			{Key: "votedAt", Value: votedAt},
		})
	}
	h.writePage(w, loc, page, pages, total, "ratings", rendered)
}

func (h *OwnRows) likedPosts(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, user *db.User) {
	ctx := r.Context()
	total, err := h.deps.DB.LikedPostCountOf(ctx, user.ID)
	if err != nil {
		h.deps.log().Error("count liked posts", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	page, pages := pageOf(r, total)

	found, err := h.deps.DB.LikedPostsOf(ctx, user.ID, (page-1)*ownRowsPerPage, ownRowsPerPage)
	if err != nil {
		h.deps.log().Error("list liked posts", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}

	rendered := make(wikijson.Array, 0, len(found))
	for _, one := range found {
		thread := "/forum/t-" + strconv.FormatInt(one.Post.ThreadID, 10) + "/" + wikidot.Normalize(one.ThreadName)
		rendered = append(rendered, wikijson.Object{
			{Key: "postId", Value: one.Post.ID},
			{Key: "name", Value: strings.TrimSpace(one.Post.Name)},
			{Key: "threadName", Value: one.ThreadName},
			{Key: "url", Value: thread + "#post-" + strconv.FormatInt(one.Post.ID, 10)},
			{Key: "likedAt", Value: isoTime(one.LikedAt)},
		})
	}
	h.writePage(w, loc, page, pages, total, "posts", rendered)
}

func (h *OwnRows) writePage(w http.ResponseWriter, loc *i18n.Localizer, page, pages, total int, key string, rows wikijson.Array) {
	body, err := wikijson.Marshal(wikijson.Object{
		{Key: "page", Value: page},
		{Key: "pages", Value: pages},
		{Key: "total", Value: total},
		{Key: key, Value: rows},
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func pageOf(r *http.Request, total int) (page, pages int) {
	pages = (total + ownRowsPerPage - 1) / ownRowsPerPage
	if pages == 0 {
		pages = 1
	}
	page = 1
	if n, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && n > 0 {
		page = n
	}
	if page > pages {
		page = pages
	}
	return page, pages
}
