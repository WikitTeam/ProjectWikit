package webapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

const FavouritesPath = "/pw-api/favourites"

const (
	favouritesPerPage = 20
	folderNameLimit   = 64
)

type Favourites struct {
	deps Deps
	next http.Handler
}

var _ http.Handler = (*Favourites)(nil)

func NewFavourites(d Deps, next http.Handler) *Favourites {
	return &Favourites{deps: d, next: next}
}

// A reader only ever gets and changes their own rows. Nothing in the request
// names a user, so there is nothing to authorise beyond being signed in.
func (h *Favourites) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != FavouritesPath || (r.Method != http.MethodGet && r.Method != http.MethodPost) {
		h.next.ServeHTTP(w, r)
		return
	}
	loc := h.deps.Bundle.For(r.Context())
	user := auth.FromContext(r.Context())
	if user == nil {
		writeJSON(w, http.StatusForbidden, field("error", loc.T("api-forbidden")))
		return
	}
	if r.Method == http.MethodPost {
		h.change(w, r, loc, user)
		return
	}
	h.list(w, r, loc, user)
}

func (h *Favourites) list(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, user *db.User) {
	ctx := r.Context()
	folders, err := h.folders(r, user)
	if err != nil {
		h.fail(w, loc, "list favourite folders", err)
		return
	}
	folder := db.DefaultFolder
	if n, err := strconv.ParseInt(r.URL.Query().Get("folder"), 10, 64); err == nil {
		for _, f := range folders {
			if f.ID == n {
				folder = n
			}
		}
	}

	total, err := h.deps.DB.FavouriteCountOf(ctx, user.ID, folder)
	if err != nil {
		h.fail(w, loc, "count favourites", err)
		return
	}
	pages := max((total+favouritesPerPage-1)/favouritesPerPage, 1)
	page := 1
	if n, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && n > 0 {
		page = min(n, pages)
	}

	found, err := h.deps.DB.FavouritesOf(ctx, user.ID, folder, (page-1)*favouritesPerPage, favouritesPerPage)
	if err != nil {
		h.fail(w, loc, "list favourites", err)
		return
	}
	sites, err := loadSiteLinks(ctx, h.deps.DB)
	if err != nil {
		h.fail(w, loc, "list sites", err)
		return
	}

	rendered := make(wikijson.Array, 0, len(found))
	for _, one := range found {
		rendered = append(rendered, favouriteJSON(sites, one))
	}
	listed := make(wikijson.Array, 0, len(folders))
	for _, f := range folders {
		listed = append(listed, wikijson.Object{
			{Key: "id", Value: f.ID},
			{Key: "name", Value: f.Name},
			{Key: "public", Value: f.Public},
			{Key: "count", Value: f.Count},
		})
	}
	body, err := wikijson.Marshal(wikijson.Object{
		{Key: "folders", Value: listed},
		{Key: "folder", Value: folder},
		{Key: "page", Value: page},
		{Key: "pages", Value: pages},
		{Key: "total", Value: total},
		{Key: "favourites", Value: rendered},
	})
	if err != nil {
		h.fail(w, loc, "encode favourites", err)
		return
	}
	writeJSON(w, http.StatusOK, body)
}

// The default folder heads the list with ID 0. Its name is left to the page,
// which knows the reader's language.
func (h *Favourites) folders(r *http.Request, user *db.User) ([]db.FavouriteFolder, error) {
	ctx := r.Context()
	public, err := h.deps.DB.DefaultFolderPublic(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	count, err := h.deps.DB.FavouriteCountOf(ctx, user.ID, db.DefaultFolder)
	if err != nil {
		return nil, err
	}
	own, err := h.deps.DB.FavouriteFolders(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return append([]db.FavouriteFolder{{ID: db.DefaultFolder, Public: public, Count: count}}, own...), nil
}

type favouriteChange struct {
	Action  string `json:"action"`
	Name    string `json:"name"`
	Public  bool   `json:"public"`
	Folder  int64  `json:"folder"`
	Article int64  `json:"article"`
}

var errFolderName = errors.New("favourite folder name is empty or too long")

func (h *Favourites) change(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, user *db.User) {
	current := site.FromContext(r.Context())
	if current == nil {
		h.fail(w, loc, "change favourites", errors.New("the request carries no site"))
		return
	}
	if err := csrf.Verify(r, []string{current.Domain}); err != nil {
		refuseCSRF(w, r, loc)
		return
	}
	raw, err := readBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	var input favouriteChange
	if json.Unmarshal(raw, &input) != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-json")))
		return
	}

	err = h.apply(r, user, input)
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, field("status", "ok"))
	case errors.Is(err, errFolderName):
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-favourite-folder-name", "count", folderNameLimit)))
	case errors.Is(err, db.ErrFolderExists):
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-favourite-folder-exists")))
	case errors.Is(err, db.ErrTooManyFolders):
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-favourite-folder-limit", "count", db.MaxFavouriteFolders)))
	case errors.Is(err, db.ErrNotFound):
		writeJSON(w, http.StatusNotFound, field("error", loc.T("api-favourite-folder-missing")))
	case errors.Is(err, errBadAction):
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
	default:
		h.fail(w, loc, "change favourites", err)
	}
}

var errBadAction = errors.New("unknown favourite action")

func (h *Favourites) apply(r *http.Request, user *db.User, in favouriteChange) error {
	ctx := r.Context()
	switch in.Action {
	case "create":
		name := strings.TrimSpace(in.Name)
		if name == "" || utf8.RuneCountInString(name) > folderNameLimit {
			return errFolderName
		}
		_, err := h.deps.DB.CreateFavouriteFolder(ctx, user.ID, name, in.Public, time.Now().UTC())
		return err
	case "visibility":
		if in.Folder == db.DefaultFolder {
			return h.deps.DB.SetDefaultFolderPublic(ctx, user.ID, in.Public)
		}
		return h.deps.DB.SetFavouriteFolderPublic(ctx, user.ID, in.Folder, in.Public)
	case "delete":
		if in.Folder == db.DefaultFolder {
			return errBadAction
		}
		return h.deps.DB.DeleteFavouriteFolder(ctx, user.ID, in.Folder)
	case "move":
		return h.deps.DB.MoveFavourite(ctx, user.ID, in.Article, in.Folder)
	}
	return errBadAction
}

func (h *Favourites) fail(w http.ResponseWriter, loc *i18n.Localizer, what string, err error) {
	h.deps.log().Error(what, "err", err)
	writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
}

func favouriteJSON(sites siteLinks, one db.Favourite) wikijson.Object {
	title := one.Article.Title
	if title == "" {
		title = one.Article.FullName()
	}
	return wikijson.Object{
		{Key: "articleId", Value: one.Article.ID},
		{Key: "pageId", Value: one.Article.FullName()},
		{Key: "site", Value: sites.title(one.SiteID)},
		{Key: "url", Value: sites.href(one.SiteID, "/"+one.Article.FullName())},
		{Key: "title", Value: title},
		{Key: "addedAt", Value: isoTime(one.AddedAt)},
	}
}
