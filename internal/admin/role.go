package admin

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

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
		found, err := h.deps.DB.AdminRoles(r.Context(), siteID(r.Context()))
		if err != nil {
			return err
		}
		managing, err := h.managingRoles(r.Context(), found)
		if err != nil {
			return err
		}
		warnings, err := h.roleWarnings(r.Context(), loc)
		if err != nil {
			return err
		}
		return h.page(w, r, loc, loc.T("admin.roles"), "role_list.html", map[string]any{
			"Roles":     found,
			"Managing":  managing,
			"Warnings":  warnings,
			"Builtin":   builtinIDs(found),
			"UsersHref": Prefix + userSlug + "/?role=",
			"New":       Prefix + roleSlug + "/new",
			"Base":      Prefix + roleSlug + "/",
		})
	}
	return h.roleForm(w, r, loc, rest, "")
}

var managingGroups = []string{"tickets", "members", "admin"}

func (h *Handler) managingRoles(ctx context.Context, roles []db.RoleRow) (map[int64]bool, error) {
	ids := make([]int64, len(roles))
	for i, one := range roles {
		ids[i] = one.ID
	}
	granted, err := h.deps.DB.RolePermissions(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]bool, len(granted))
	for _, role := range granted {
		for _, name := range role.Permissions {
			if slices.Contains(managingGroups, perms.GroupOf(name)) {
				out[role.ID] = true
				break
			}
		}
	}
	return out, nil
}

// Only the roles everyone holds are flagged. A Deny on any other role is
// usually the point of that role.
func (h *Handler) roleWarnings(ctx context.Context, loc *i18n.Localizer) (map[int64][]finding, error) {
	x, err := h.newExplainer(ctx, loc)
	if err != nil {
		return nil, err
	}
	findings, err := h.lint(ctx, x)
	if err != nil {
		return nil, err
	}
	out := map[int64][]finding{}
	for _, f := range findings {
		if choice, ok := x.roles[f.RoleID]; ok && slices.Contains(builtinRoles, choice.Slug) {
			out[f.RoleID] = append(out[f.RoleID], f)
		}
	}
	return out, nil
}

// Everyone and registered are added to every visitor when permissions are
// worked out, so a stored link to them would only skew the member count.
func assignableRoles(all []db.RoleChoice) []db.RoleChoice {
	out := make([]db.RoleChoice, 0, len(all))
	for _, one := range all {
		if !slices.Contains(builtinRoles, one.Slug) {
			out = append(out, one)
		}
	}
	return out
}

// A role the form did not offer is dropped rather than refused, so a setting
// that already points at a built-in role clears on the next save.
func offered(id *int64, choices []db.RoleChoice) *int64 {
	if id == nil {
		return nil
	}
	for _, one := range choices {
		if one.ID == *id {
			return id
		}
	}
	return nil
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
		row, err = h.deps.DB.AdminRole(ctx, siteID(ctx), id)
		if errors.Is(err, db.ErrNotFound) {
			notFound(w)
			return nil
		}
		if err != nil {
			return err
		}
	}

	catalog, err := h.deps.DB.PermissionCatalog(ctx)
	if err != nil {
		return err
	}
	categories, err := h.deps.DB.RoleCategories(ctx, siteID(ctx))
	if err != nil {
		return err
	}
	granted, _, err := h.access(ctx)
	if err != nil {
		return err
	}
	warnings, err := h.roleWarnings(ctx, loc)
	if err != nil {
		return err
	}
	own := warnings[row.ID]
	for i := range own {
		if own[i].Href == Prefix+roleSlug+"/"+rest {
			own[i].Href = ""
		}
	}

	return h.page(w, r, loc, loc.T("admin.roles"), "role_form.html", map[string]any{
		"Role":         row,
		"Grants":       grantGroups(catalog, row.Allow, row.Deny),
		"Categories":   categories,
		"InlineModes":  inlineModes,
		"ProfileModes": profileModes,
		"MayGrant":     granted.Has(perms.ManagePermissions),
		"Builtin":      slices.Contains(builtinRoles, row.Slug),
		"CSRF":         csrf.Issue(w, r),
		"Error":        problem,
		"Warnings":     own,
		"Action":       Prefix + roleSlug + "/" + rest,
		"Back":         Prefix + roleSlug + "/",
		"Access":       h.accessHref(ctx, "role?role="+strconv.FormatInt(row.ID, 10)),
	})
}

func (h *Handler) saveRole(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, rest string) error {
	ctx := r.Context()
	current := site.FromContext(ctx)
	if err := csrf.Verify(r, []string{current.Domain}); err != nil {
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
		stored, err = h.deps.DB.AdminRole(ctx, siteID(ctx), id)
		if errors.Is(err, db.ErrNotFound) {
			notFound(w)
			return nil
		}
		if err != nil {
			return err
		}
	}
	builtin := slices.Contains(builtinRoles, stored.Slug)

	if r.PostFormValue("delete") != "" && stored.ID != 0 {
		if builtin {
			return h.roleForm(w, r, loc, rest, loc.T("admin.role-builtin"))
		}
		if err := h.deps.DB.DeleteRole(ctx, siteID(ctx), stored.ID); err != nil {
			return err
		}
		h.noteID(r, db.AdminDeleted, roleSlug, stored.ID, stored.Slug)
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
	icon, err := h.pickIcon(r, "icon", stored.Icon, roleIcon)
	if err != nil {
		return h.roleForm(w, r, loc, rest, iconMessage(loc, err))
	}
	next.Icon = icon
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
	did := db.AdminChanged
	if next.ID == 0 {
		did = db.AdminCreated
	}
	if _, err := h.deps.DB.SaveRole(ctx, siteID(ctx), next, mayGrant); err != nil {
		return err
	}
	h.noteID(r, did, roleSlug, next.ID, next.Slug)
	redirect(w, Prefix+roleSlug+"/")
	return nil
}

func validRoleSlug(slug string) bool {
	return slug != "" && utf8.ValidString(slug) && !strings.ContainsFunc(slug, unicode.IsControl)
}

func checkRole(loc *i18n.Localizer, r db.RoleRow) string {
	switch {
	case !validRoleSlug(r.Slug):
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
		if err := csrf.Verify(r, []string{current.Domain}); err != nil {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return nil
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return nil
		}
		id := optionalID(r.PostFormValue("id"))
		if r.PostFormValue("delete") != "" && id != nil {
			if err := h.deps.DB.DeleteRoleCategory(ctx, siteID(ctx), *id); err != nil {
				return err
			}
			h.noteID(r, db.AdminDeleted, roleCategorySlug, *id, "")
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
		did := db.AdminChanged
		if row.ID == 0 {
			did = db.AdminCreated
		}
		if err := h.deps.DB.SaveRoleCategory(ctx, siteID(ctx), row); err != nil {
			return err
		}
		h.noteID(r, did, roleCategorySlug, row.ID, row.Name)
		redirect(w, Prefix+roleCategorySlug+"/")
		return nil
	}
	return h.roleCategoryList(w, r, loc, "")
}

func (h *Handler) roleCategoryList(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, problem string) error {
	found, err := h.deps.DB.RoleCategories(r.Context(), siteID(r.Context()))
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

func builtinIDs(roles []db.RoleRow) map[int64]bool {
	out := map[int64]bool{}
	for _, one := range roles {
		if slices.Contains(builtinRoles, one.Slug) {
			out[one.ID] = true
		}
	}
	return out
}
