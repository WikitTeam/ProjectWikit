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

const (
	tagSlug         = "tags"
	tagCategorySlug = "tag-categories"
)

func init() {
	register(screen{slug: tagSlug, label: "admin.tags", need: perms.ManageTags, serve: (*Handler).tags})
	register(screen{slug: tagCategorySlug, label: "admin.tag-categories", need: perms.ManageTags, serve: (*Handler).tagCategories})
}

func (h *Handler) tags(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, Prefix+tagSlug), "/")
	ctx := r.Context()

	if r.Method == http.MethodPost {
		if !h.verified(w, r) {
			return nil
		}
		row := db.TagRow{
			Name:       strings.TrimSpace(r.PostFormValue("name")),
			CategoryID: optionalID(r.PostFormValue("category")),
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
			if err := h.deps.DB.DeleteTag(ctx, row.ID); err != nil {
				return err
			}
			h.noteID(r, db.AdminDeleted, tagSlug, row.ID, row.Name)
			redirect(w, Prefix+tagSlug+"/")
			return nil
		}
		if row.Name == "" {
			return h.tagForm(w, r, loc, rest, loc.T("admin.tag-no-name"))
		}
		did := db.AdminChanged
		if row.ID == 0 {
			did = db.AdminCreated
		}
		if err := h.deps.DB.SaveTag(ctx, row); err != nil {
			return err
		}
		h.noteID(r, did, tagSlug, row.ID, row.Name)
		redirect(w, Prefix+tagSlug+"/")
		return nil
	}

	if rest == "" {
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		page := atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		found, total, err := h.deps.DB.AdminTags(ctx, query, perPage, (page-1)*perPage)
		if err != nil {
			return err
		}
		return h.page(w, r, loc, loc.T("admin.tags"), "tag_list.html", map[string]any{
			"Tags":   found,
			"Query":  query,
			"Page":   page,
			"Pages":  (total + perPage - 1) / perPage,
			"Total":  total,
			"New":    Prefix + tagSlug + "/new",
			"Base":   Prefix + tagSlug + "/",
			"Action": Prefix + tagSlug + "/",
		})
	}
	return h.tagForm(w, r, loc, rest, "")
}

func (h *Handler) tagForm(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, rest, problem string) error {
	ctx := r.Context()
	row := db.TagRow{}
	if rest != "new" {
		id, err := strconv.ParseInt(rest, 10, 64)
		if err != nil {
			notFound(w)
			return nil
		}
		row, err = h.deps.DB.AdminTag(ctx, id)
		if errors.Is(err, db.ErrNotFound) {
			notFound(w)
			return nil
		}
		if err != nil {
			return err
		}
	}
	categories, err := h.deps.DB.AdminTagCategories(ctx)
	if err != nil {
		return err
	}
	return h.page(w, r, loc, loc.T("admin.tags"), "tag_form.html", map[string]any{
		"Tag":        row,
		"Categories": categories,
		"CSRF":       csrf.Issue(w, r),
		"Error":      problem,
		"Action":     Prefix + tagSlug + "/" + rest,
		"Back":       Prefix + tagSlug + "/",
	})
}

func (h *Handler) tagCategories(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, Prefix+tagCategorySlug), "/")
	ctx := r.Context()

	if r.Method == http.MethodPost {
		if !h.verified(w, r) {
			return nil
		}
		row := db.TagCategoryRow{
			Name:        strings.TrimSpace(r.PostFormValue("name")),
			Slug:        strings.TrimSpace(r.PostFormValue("slug")),
			Description: r.PostFormValue("description"),
			Priority:    optionalInt(r.PostFormValue("priority")),
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
			if err := h.deps.DB.DeleteTagCategory(ctx, row.ID); err != nil {
				return err
			}
			h.noteID(r, db.AdminDeleted, tagCategorySlug, row.ID, row.Name)
			redirect(w, Prefix+tagCategorySlug+"/")
			return nil
		}
		if row.Name == "" {
			return h.tagCategoryForm(w, r, loc, rest, loc.T("admin.tag-no-name"))
		}
		if !slugPattern.MatchString(row.Slug) {
			return h.tagCategoryForm(w, r, loc, rest, loc.T("admin.tag-bad-slug"))
		}
		did := db.AdminChanged
		if row.ID == 0 {
			did = db.AdminCreated
		}
		if err := h.deps.DB.SaveTagCategory(ctx, row); err != nil {
			return err
		}
		h.noteID(r, did, tagCategorySlug, row.ID, row.Name)
		redirect(w, Prefix+tagCategorySlug+"/")
		return nil
	}

	if rest == "" {
		found, err := h.deps.DB.AdminTagCategories(ctx)
		if err != nil {
			return err
		}
		return h.page(w, r, loc, loc.T("admin.tag-categories"), "tag_category_list.html", map[string]any{
			"Categories": found,
			"New":        Prefix + tagCategorySlug + "/new",
			"Base":       Prefix + tagCategorySlug + "/",
		})
	}
	return h.tagCategoryForm(w, r, loc, rest, "")
}

func optionalInt(raw string) *int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return nil
	}
	return &n
}

func (h *Handler) tagCategoryForm(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, rest, problem string) error {
	row := db.TagCategoryRow{}
	if rest != "new" {
		id, err := strconv.ParseInt(rest, 10, 64)
		if err != nil {
			notFound(w)
			return nil
		}
		row, err = h.deps.DB.AdminTagCategory(r.Context(), id)
		if errors.Is(err, db.ErrNotFound) {
			notFound(w)
			return nil
		}
		if err != nil {
			return err
		}
	}
	return h.page(w, r, loc, loc.T("admin.tag-categories"), "tag_category_form.html", map[string]any{
		"Category": row,
		"CSRF":     csrf.Issue(w, r),
		"Error":    problem,
		"Action":   Prefix + tagCategorySlug + "/" + rest,
		"Back":     Prefix + tagCategorySlug + "/",
	})
}
