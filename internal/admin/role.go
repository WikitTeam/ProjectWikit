package admin

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/site"
)

const (
	roleSlug         = "roles"
	roleCategorySlug = "role-categories"
)

var (
	builtinRoles = []string{"everyone", "registered"}
	inlineModes  = []string{"hidden", "badge", "icon"}
	profileModes = []string{"hidden", "badge", "status"}
)

func init() {
	register(screen{slug: roleSlug, label: "admin.roles", need: perms.ManageRoles, serve: (*Handler).roles})
	register(screen{slug: roleCategorySlug, label: "admin.role-categories", need: perms.ManageRoles, serve: (*Handler).roleCategories})
}

func (h *Handler) roles(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, Prefix+roleSlug), "/")
	if r.Method == http.MethodPost {
		return h.saveRole(w, r, loc, rest)
	}
	if rest == "" {
		found, err := h.deps.DB.AdminRoles(r.Context())
		if err != nil {
			return err
		}
		return h.page(w, r, loc, loc.T("admin.roles"), "role_list.html", map[string]any{
			"Roles": found,
			"New":   Prefix + roleSlug + "/new",
			"Base":  Prefix + roleSlug + "/",
		})
	}
	return h.roleForm(w, r, loc, rest, "")
}

func (h *Handler) roleForm(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, rest, problem string) error {
	ctx := r.Context()
	row := db.RoleRow{InlineVisualMode: "hidden", ProfileVisualMode: "hidden"}
	if rest != "new" {
		id, err := strconv.ParseInt(rest, 10, 64)
		if err != nil {
			notFound(w)
			return nil
		}
		row, err = h.deps.DB.AdminRole(ctx, id)
		if errors.Is(err, db.ErrNotFound) {
			notFound(w)
			return nil
		}
		if err != nil {
			return err
		}
	}
	outranked, err := h.outranked(ctx, row)
	if err != nil {
		return err
	}
	if outranked {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return nil
	}

	catalog, err := h.deps.DB.PermissionCatalog(ctx)
	if err != nil {
		return err
	}
	categories, err := h.deps.DB.RoleCategories(ctx)
	if err != nil {
		return err
	}
	granted, _, err := h.access(ctx)
	if err != nil {
		return err
	}

	type grant struct {
		Name  string
		Allow bool
		Deny  bool
	}
	grants := make([]grant, 0, len(catalog))
	for _, name := range catalog {
		grants = append(grants, grant{
			Name:  name,
			Allow: slices.Contains(row.Allow, name),
			Deny:  slices.Contains(row.Deny, name),
		})
	}

	return h.page(w, r, loc, loc.T("admin.roles"), "role_form.html", map[string]any{
		"Role":         row,
		"Grants":       grants,
		"Categories":   categories,
		"InlineModes":  inlineModes,
		"ProfileModes": profileModes,
		"MayGrant":     granted.Has(perms.ManagePermissions),
		"Builtin":      slices.Contains(builtinRoles, row.Slug),
		"CSRF":         csrf.Issue(w, r),
		"Error":        problem,
		"Action":       Prefix + roleSlug + "/" + rest,
		"Back":         Prefix + roleSlug + "/",
	})
}

func (h *Handler) saveRole(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, rest string) error {
	ctx := r.Context()
	current := site.FromContext(ctx)
	if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return nil
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return nil
	}

	var stored db.RoleRow
	if rest != "new" {
		id, err := strconv.ParseInt(rest, 10, 64)
		if err != nil {
			notFound(w)
			return nil
		}
		stored, err = h.deps.DB.AdminRole(ctx, id)
		if errors.Is(err, db.ErrNotFound) {
			notFound(w)
			return nil
		}
		if err != nil {
			return err
		}
	}
	outranked, err := h.outranked(ctx, stored)
	if err != nil {
		return err
	}
	if outranked {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return nil
	}
	builtin := slices.Contains(builtinRoles, stored.Slug)

	if r.PostFormValue("delete") != "" && stored.ID != 0 {
		if builtin {
			return h.roleForm(w, r, loc, rest, loc.T("admin.role-builtin"))
		}
		if err := h.deps.DB.DeleteRole(ctx, stored.ID); err != nil {
			return err
		}
		redirect(w, Prefix+roleSlug+"/")
		return nil
	}

	next := stored
	next.Slug = strings.TrimSpace(r.PostFormValue("slug"))
	if builtin {
		next.Slug = stored.Slug
	}
	next.Name = strings.TrimSpace(r.PostFormValue("name"))
	next.ShortName = strings.TrimSpace(r.PostFormValue("short_name"))
	next.CategoryID = optionalID(r.PostFormValue("category"))
	next.Index = atoi(r.PostFormValue("index"))
	next.IsStaff = r.PostFormValue("is_staff") != ""
	next.GroupVotes = r.PostFormValue("group_votes") != ""
	next.VotesTitle = strings.TrimSpace(r.PostFormValue("votes_title"))
	next.InlineVisualMode = r.PostFormValue("inline_visual_mode")
	next.ProfileVisualMode = r.PostFormValue("profile_visual_mode")
	next.Color = strings.TrimSpace(r.PostFormValue("color"))
	next.Icon = strings.TrimSpace(r.PostFormValue("icon"))
	next.BadgeText = strings.TrimSpace(r.PostFormValue("badge_text"))
	next.BadgeBg = strings.TrimSpace(r.PostFormValue("badge_bg"))
	next.BadgeTextColor = strings.TrimSpace(r.PostFormValue("badge_text_color"))
	next.BadgeShowBorder = r.PostFormValue("badge_show_border") != ""

	granted, _, err := h.access(ctx)
	if err != nil {
		return err
	}
	mayGrant := granted.Has(perms.ManagePermissions)
	if mayGrant {
		catalog, err := h.deps.DB.PermissionCatalog(ctx)
		if err != nil {
			return err
		}
		next.Allow, next.Deny = nil, nil
		for _, name := range catalog {
			switch r.PostFormValue("perm_" + name) {
			case "allow":
				next.Allow = append(next.Allow, name)
			case "deny":
				next.Deny = append(next.Deny, name)
			}
		}
	}

	if problem := checkRole(loc, next); problem != "" {
		return h.roleForm(w, r, loc, rest, problem)
	}
	if _, err := h.deps.DB.SaveRole(ctx, next, mayGrant); err != nil {
		return err
	}
	redirect(w, Prefix+roleSlug+"/")
	return nil
}

func (h *Handler) outranked(ctx context.Context, row db.RoleRow) (bool, error) {
	if row.ID == 0 {
		return false, nil
	}
	user := auth.FromContext(ctx)
	if user == nil {
		return true, nil
	}
	if user.IsSuperuser {
		return false, nil
	}
	mine, err := h.deps.DB.OperationIndex(ctx, user.ID)
	if err != nil {
		return false, err
	}
	return row.Index < mine, nil
}

func checkRole(loc *i18n.Localizer, r db.RoleRow) string {
	switch {
	case !slugPattern.MatchString(r.Slug):
		return loc.T("admin.role-bad-slug")
	case !contains(inlineModes, r.InlineVisualMode) || !contains(profileModes, r.ProfileVisualMode):
		return loc.T("admin.role-bad-mode")
	}
	return ""
}

func (h *Handler) roleCategories(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	ctx := r.Context()
	if r.Method == http.MethodPost {
		current := site.FromContext(ctx)
		if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return nil
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return nil
		}
		id := optionalID(r.PostFormValue("id"))
		if r.PostFormValue("delete") != "" && id != nil {
			if err := h.deps.DB.DeleteRoleCategory(ctx, *id); err != nil {
				return err
			}
			redirect(w, Prefix+roleCategorySlug+"/")
			return nil
		}
		name := strings.TrimSpace(r.PostFormValue("name"))
		if name == "" {
			return h.roleCategoryList(w, r, loc, loc.T("admin.role-category-no-name"))
		}
		row := db.RoleCategoryRow{Name: name}
		if id != nil {
			row.ID = *id
		}
		if err := h.deps.DB.SaveRoleCategory(ctx, row); err != nil {
			return err
		}
		redirect(w, Prefix+roleCategorySlug+"/")
		return nil
	}
	return h.roleCategoryList(w, r, loc, "")
}

func (h *Handler) roleCategoryList(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, problem string) error {
	found, err := h.deps.DB.RoleCategories(r.Context())
	if err != nil {
		return err
	}
	return h.page(w, r, loc, loc.T("admin.role-categories"), "role_category.html", map[string]any{
		"Categories": found,
		"CSRF":       csrf.Issue(w, r),
		"Error":      problem,
		"Action":     Prefix + roleCategorySlug + "/",
	})
}

func atoi(raw string) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0
	}
	return n
}
