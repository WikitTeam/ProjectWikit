package userpage

import (
	"context"
	"net/http"
	"slices"
	"strconv"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/listpages"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/shell"
)

const (
	publicFavouritesRead = 2000
	folderParam          = "folder"
	folderPageParam      = "page"
	defaultFolderKey     = "default"
)

// A public folder is filtered through the viewer's permissions like the pins,
// so making a folder public never shows a page its reader may not open.
func (h *Handler) favouriteFolders(ctx context.Context, loc *i18n.Localizer, current *db.Site,
	profile *db.Profile, viewer *db.User, profileURL string) ([]shell.ProfileFolder, error) {

	defaultPublic, err := h.deps.DB.DefaultFolderPublic(ctx, profile.ID)
	if err != nil {
		return nil, err
	}
	folders, err := h.deps.DB.FavouriteFolders(ctx, profile.ID)
	if err != nil {
		return nil, err
	}
	var out []shell.ProfileFolder
	ids := map[int64]int{}
	if defaultPublic {
		ids[db.DefaultFolder] = 0
		out = append(out, shell.ProfileFolder{Key: defaultFolderKey, Name: loc.T("profile.favourites-default")})
	}
	for _, f := range folders {
		if f.Public {
			ids[f.ID] = len(out)
			out = append(out, shell.ProfileFolder{Key: strconv.FormatInt(f.ID, 10), Name: f.Name})
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	for i := range out {
		out[i].URL = profileURL + "?" + folderParam + "=" + out[i].Key
	}

	found, err := h.deps.DB.PublicFavourites(ctx, profile.ID, defaultPublic, publicFavouritesRead)
	if err != nil {
		return nil, err
	}
	sites, err := h.siteSet(ctx, current)
	if err != nil {
		return nil, err
	}
	hidden := map[int64][]string{}
	for _, one := range found {
		at, ok := ids[one.FolderID]
		if !ok {
			continue
		}
		on := sites.of(one.SiteID)
		if _, ok := hidden[on.ID]; !ok {
			if hidden[on.ID], err = repo.HiddenCategoriesOn(ctx, h.deps.DB, on, viewer); err != nil {
				return nil, err
			}
		}
		if slices.Contains(hidden[on.ID], one.Article.Category) {
			continue
		}
		title := one.Article.Title
		if title == "" {
			title = one.Article.FullName()
		}
		out[at].Pages = append(out[at].Pages, shell.ProfilePage{
			ArticleID: one.Article.ID, Title: title, URL: sites.href(on, "/"+one.Article.FullName()),
			Site: on.Title, At: one.AddedAt,
		})
	}
	for i := range out {
		out[i].Count = len(out[i].Pages)
	}
	return out, nil
}

// A folder that is private or belongs to someone else reads as missing, the
// same as one that never existed.
func openFolder(r *http.Request, loc *i18n.Localizer, folders []shell.ProfileFolder) (*shell.ProfileFolder, string, bool) {
	key := r.URL.Query().Get(folderParam)
	if key == "" {
		return nil, "", false
	}
	for i := range folders {
		if folders[i].Key != key {
			continue
		}
		f := folders[i]
		page, pages := pageOf(r, folderPageParam, len(f.Pages))
		f.Pages = f.Pages[min(len(f.Pages), (page-1)*historyPerPage):min(len(f.Pages), page*historyPerPage)]
		return &f, listpages.PaginationLinks(loc, pageHref(r, folderPageParam), page, pages), true
	}
	return nil, "", true
}
