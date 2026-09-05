package webapi

import (
	"net/http"
	"strconv"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

const FavouritesPath = "/pw-api/favourites"

const favouritesPerPage = 20

type Favourites struct {
	deps     Deps
	upstream http.Handler
}

var _ http.Handler = (*Favourites)(nil)

func NewFavourites(d Deps, upstream http.Handler) *Favourites {
	return &Favourites{deps: d, upstream: upstream}
}

// A reader only ever gets their own rows. Nothing in the request names a user,
// so there is nothing to authorise beyond being signed in.
func (h *Favourites) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != FavouritesPath {
		h.upstream.ServeHTTP(w, r)
		return
	}
	if r.Method != http.MethodGet {
		h.upstream.ServeHTTP(w, r)
		return
	}
	loc := h.deps.Bundle.Localizer(i18n.DefaultLanguage)
	user := auth.FromContext(r.Context())
	if user == nil {
		writeJSON(w, http.StatusForbidden, field("error", loc.T("api-forbidden")))
		return
	}
	ctx := r.Context()

	total, err := h.deps.DB.FavouriteCountOf(ctx, user.ID)
	if err != nil {
		h.deps.log().Error("count favourites", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	pages := (total + favouritesPerPage - 1) / favouritesPerPage
	if pages == 0 {
		pages = 1
	}
	page := 1
	if n, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && n > 0 {
		page = n
	}
	if page > pages {
		page = pages
	}

	found, err := h.deps.DB.FavouritesOf(ctx, user.ID, (page-1)*favouritesPerPage, favouritesPerPage)
	if err != nil {
		h.deps.log().Error("list favourites", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}

	rendered := make(wikijson.Array, 0, len(found))
	for _, one := range found {
		rendered = append(rendered, favouriteJSON(one))
	}
	body, err := wikijson.Marshal(wikijson.Object{
		{Key: "page", Value: page},
		{Key: "pages", Value: pages},
		{Key: "total", Value: total},
		{Key: "favourites", Value: rendered},
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func favouriteJSON(one db.Favourite) wikijson.Object {
	title := one.Article.Title
	if title == "" {
		title = one.Article.FullName()
	}
	return wikijson.Object{
		{Key: "pageId", Value: one.Article.FullName()},
		{Key: "title", Value: title},
		{Key: "addedAt", Value: isoTime(one.AddedAt)},
	}
}
