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
	"github.com/WikitTeam/ProjectWikit/internal/site"
)

const (
	sectionSlug  = "forum-sections"
	categorySlug = "forum-categories"
)

func init() {
	register(screen{slug: sectionSlug, label: "admin.forum-sections", need: perms.ManageForum, serve: (*Handler).forumSections})
	register(screen{slug: categorySlug, label: "admin.forum-categories", need: perms.ManageForum, serve: (*Handler).forumCategories})
}

func (h *Handler) forumSections(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, Prefix+sectionSlug), "/")
	ctx := r.Context()

	if r.Method == http.MethodPost {
		if !h.verified(w, r) {
			return nil
		}
		row := db.ForumSectionRow{
			Name:            strings.TrimSpace(r.PostFormValue("name")),
			Description:     r.PostFormValue("description"),
			Order:           atoi(r.PostFormValue("order")),
			IsHidden:        r.PostFormValue("is_hidden") != "",
			IsHiddenForUser: r.PostFormValue("is_hidden_for_users") != "",
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
			if err := h.deps.DB.DeleteForumSection(ctx, row.ID); err != nil {
				return err
			}
			h.noteID(r, db.AdminDeleted, sectionSlug, row.ID, row.Name)
			redirect(w, Prefix+sectionSlug+"/")
			return nil
		}
		if row.Name == "" {
			return h.forumSectionForm(w, r, loc, rest, loc.T("admin.forum-no-name"))
		}
		did := db.AdminChanged
		if row.ID == 0 {
			did = db.AdminCreated
		}
		if err := h.deps.DB.SaveForumSection(ctx, siteID(ctx), row); err != nil {
			return err
		}
		h.noteID(r, did, sectionSlug, row.ID, row.Name)
		redirect(w, Prefix+sectionSlug+"/")
		return nil
	}

	if rest == "" {
		found, err := h.deps.DB.AdminForumSections(ctx, siteID(ctx))
		if err != nil {
			return err
		}
		return h.page(w, r, loc, loc.T("admin.forum-sections"), "forum_section_list.html", map[string]any{
			"Sections": found,
			"New":      Prefix + sectionSlug + "/new",
			"Base":     Prefix + sectionSlug + "/",
		})
	}
	return h.forumSectionForm(w, r, loc, rest, "")
}

func (h *Handler) forumSectionForm(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, rest, problem string) error {
	row := db.ForumSectionRow{}
	if rest != "new" {
		id, err := strconv.ParseInt(rest, 10, 64)
		if err != nil {
			notFound(w)
			return nil
		}
		row, err = h.deps.DB.AdminForumSection(r.Context(), id)
		if errors.Is(err, db.ErrNotFound) {
			notFound(w)
			return nil
		}
		if err != nil {
			return err
		}
	}
	return h.page(w, r, loc, loc.T("admin.forum-sections"), "forum_section_form.html", map[string]any{
		"Section": row,
		"CSRF":    csrf.Issue(w, r),
		"Error":   problem,
		"Action":  Prefix + sectionSlug + "/" + rest,
		"Back":    Prefix + sectionSlug + "/",
	})
}

func (h *Handler) forumCategories(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, Prefix+categorySlug), "/")
	ctx := r.Context()

	if r.Method == http.MethodPost {
		if !h.verified(w, r) {
			return nil
		}
		section := optionalID(r.PostFormValue("section"))
		row := db.ForumCategoryRow{
			Name:          strings.TrimSpace(r.PostFormValue("name")),
			Description:   r.PostFormValue("description"),
			Order:         atoi(r.PostFormValue("order")),
			IsForComments: r.PostFormValue("is_for_comments") != "",
		}
		if section != nil {
			row.SectionID = *section
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
			if err := h.deps.DB.DeleteForumCategory(ctx, row.ID); err != nil {
				return err
			}
			h.noteID(r, db.AdminDeleted, categorySlug, row.ID, row.Name)
			redirect(w, Prefix+categorySlug+"/")
			return nil
		}
		if row.Name == "" {
			return h.forumCategoryForm(w, r, loc, rest, loc.T("admin.forum-no-name"))
		}
		if row.SectionID == 0 {
			return h.forumCategoryForm(w, r, loc, rest, loc.T("admin.forum-no-section"))
		}
		did := db.AdminChanged
		if row.ID == 0 {
			did = db.AdminCreated
		}
		if err := h.deps.DB.SaveForumCategory(ctx, row); err != nil {
			return err
		}
		h.noteID(r, did, categorySlug, row.ID, row.Name)
		redirect(w, Prefix+categorySlug+"/")
		return nil
	}

	if rest == "" {
		found, err := h.deps.DB.AdminForumCategories(ctx, siteID(ctx))
		if err != nil {
			return err
		}
		return h.page(w, r, loc, loc.T("admin.forum-categories"), "forum_category_list.html", map[string]any{
			"Categories": found,
			"New":        Prefix + categorySlug + "/new",
			"Base":       Prefix + categorySlug + "/",
		})
	}
	return h.forumCategoryForm(w, r, loc, rest, "")
}

func (h *Handler) forumCategoryForm(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, rest, problem string) error {
	ctx := r.Context()
	row := db.ForumCategoryRow{}
	if rest != "new" {
		id, err := strconv.ParseInt(rest, 10, 64)
		if err != nil {
			notFound(w)
			return nil
		}
		row, err = h.deps.DB.AdminForumCategory(ctx, id)
		if errors.Is(err, db.ErrNotFound) {
			notFound(w)
			return nil
		}
		if err != nil {
			return err
		}
	}
	sections, err := h.deps.DB.AdminForumSections(ctx, siteID(ctx))
	if err != nil {
		return err
	}
	return h.page(w, r, loc, loc.T("admin.forum-categories"), "forum_category_form.html", map[string]any{
		"Category": row,
		"Sections": sections,
		"CSRF":     csrf.Issue(w, r),
		"Error":    problem,
		"Action":   Prefix + categorySlug + "/" + rest,
		"Back":     Prefix + categorySlug + "/",
	})
}

func (h *Handler) verified(w http.ResponseWriter, r *http.Request) bool {
	current := site.FromContext(r.Context())
	if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return false
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return false
	}
	return true
}
