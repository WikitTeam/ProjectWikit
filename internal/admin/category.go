package admin

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
)

const pageCategorySlug = "page-categories"

func init() {
	register(screen{slug: pageCategorySlug, label: "admin.page-categories", need: perms.ManageCategories, serve: (*Handler).pageCategories})
}

func (h *Handler) pageCategories(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, Prefix+pageCategorySlug), "/")
	if r.Method == http.MethodPost {
		return h.savePageCategory(w, r, loc, rest)
	}
	if rest == "" {
		found, err := h.deps.DB.AdminCategories(r.Context(), siteID(r.Context()))
		if err != nil {
			return err
		}
		return h.page(w, r, loc, loc.T("admin.page-categories"), "page_category_list.html", map[string]any{
			"Categories": found,
			"New":        Prefix + pageCategorySlug + "/new",
			"Base":       Prefix + pageCategorySlug + "/",
		})
	}
	return h.pageCategoryForm(w, r, loc, rest, "")
}

func (h *Handler) pageCategoryForm(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, rest, problem string) error {
	ctx := r.Context()
	row := db.CategoryRow{IsIndexed: true, Settings: db.SiteSettings{RatingMode: "default", CreateTags: "default"}}
	if rest != "new" {
		id, err := strconv.ParseInt(rest, 10, 64)
		if err != nil {
			notFound(w)
			return nil
		}
		row, err = h.deps.DB.AdminCategory(ctx, siteID(ctx), id)
		if errors.Is(err, db.ErrNotFound) {
			notFound(w)
			return nil
		}
		if err != nil {
			return err
		}
		if row.Settings.RatingMode == "" {
			row.Settings = db.SiteSettings{RatingMode: "default", CreateTags: "default"}
		}
	}
	roleList, err := h.deps.DB.AllRoles(ctx, siteID(ctx))
	if err != nil {
		return err
	}
	catalog, err := h.deps.DB.PermissionCatalog(ctx)
	if err != nil {
		return err
	}
	granted, _, err := h.access(ctx)
	if err != nil {
		return err
	}

	type cell struct {
		Role    db.RoleChoice
		Grants  []grantGroup
		Present bool
	}
	byRole := map[int64]db.CategoryOverride{}
	for _, one := range row.Overrides {
		byRole[one.RoleID] = one
	}
	cells := make([]cell, 0, len(roleList))
	for _, one := range roleList {
		override, present := byRole[one.ID]
		cells = append(cells, cell{
			Role:    one,
			Grants:  grantGroups(catalog, override.Allow, override.Deny),
			Present: present,
		})
	}

	return h.page(w, r, loc, loc.T("admin.page-categories"), "page_category_form.html", map[string]any{
		"Category":    row,
		"Cells":       cells,
		"RatingModes": ratingModes,
		"TagModes":    tagModes,
		"MayGrant":    granted.Has(perms.ManagePermissions),
		"CSRF":        csrf.Issue(w, r),
		"Error":       problem,
		"Action":      Prefix + pageCategorySlug + "/" + rest,
		"Back":        Prefix + pageCategorySlug + "/",
	})
}

func (h *Handler) savePageCategory(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, rest string) error {
	ctx := r.Context()
	if !h.verified(w, r) {
		return nil
	}
	row := db.CategoryRow{
		Name:      strings.TrimSpace(r.PostFormValue("name")),
		IsIndexed: r.PostFormValue("is_indexed") != "",
		Settings: db.SiteSettings{
			RatingMode: r.PostFormValue("rating_mode"),
			CreateTags: r.PostFormValue("can_user_create_tags"),
		},
	}
	if rest != "new" {
		id, err := strconv.ParseInt(rest, 10, 64)
		if err != nil {
			notFound(w)
			return nil
		}
		row.ID = id
	}
	if r.PostFormValue("delete") != "" && row.ID != 0 {
		h.noteID(r, db.AdminDeleted, pageCategorySlug, row.ID, row.Name)
		if err := h.deps.DB.DeleteCategory(ctx, siteID(ctx), row.ID); err != nil {
			return err
		}
		redirect(w, Prefix+pageCategorySlug+"/")
		return nil
	}
	if row.Name == "" {
		return h.pageCategoryForm(w, r, loc, rest, loc.T("admin.category-no-name"))
	}
	if !contains(ratingModes, row.Settings.RatingMode) || !contains(tagModes, row.Settings.CreateTags) {
		return h.pageCategoryForm(w, r, loc, rest, loc.T("admin.site-bad-mode"))
	}
	did := db.AdminChanged
	if row.ID == 0 {
		did = db.AdminCreated
	}
	if err := h.deps.DB.SaveCategory(ctx, siteID(ctx), row); err != nil {
		return err
	}

	granted, _, err := h.access(ctx)
	if err != nil {
		return err
	}
	if !granted.Has(perms.ManagePermissions) {
		redirect(w, Prefix+pageCategorySlug+"/")
		return nil
	}
	if row.ID == 0 {
		found, err := h.deps.DB.AdminCategories(ctx, siteID(ctx))
		if err != nil {
			return err
		}
		for _, one := range found {
			if one.Name == row.Name {
				row.ID = one.ID
			}
		}
	}
	roleList, err := h.deps.DB.AllRoles(ctx, siteID(ctx))
	if err != nil {
		return err
	}
	catalog, err := h.deps.DB.PermissionCatalog(ctx)
	if err != nil {
		return err
	}
	var overrides []db.CategoryOverride
	for _, one := range roleList {
		if r.PostFormValue("override_"+strconv.FormatInt(one.ID, 10)) == "" {
			continue
		}
		next := db.CategoryOverride{RoleID: one.ID}
		for _, name := range catalog {
			switch r.PostFormValue("perm_" + strconv.FormatInt(one.ID, 10) + "_" + name) {
			case "allow":
				next.Allow = append(next.Allow, name)
			case "deny":
				next.Deny = append(next.Deny, name)
			}
		}
		overrides = append(overrides, next)
	}
	if err := h.deps.DB.SaveCategoryOverrides(ctx, row.ID, overrides); err != nil {
		return err
	}
	h.noteID(r, did, pageCategorySlug, row.ID, row.Name)
	redirect(w, Prefix+pageCategorySlug+"/")
	return nil
}
