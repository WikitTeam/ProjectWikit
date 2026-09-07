package admin

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/site"
)

const (
	userSlug = "users"
	perPage  = 50
)

var userTypes = []string{"normal", "wikidot", "bot", "system"}

func init() {
	register(screen{slug: userSlug, label: "admin.users", need: perms.ManageUsers, serve: (*Handler).users})
}

func (h *Handler) users(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, Prefix+userSlug), "/")
	if id, tail, ok := strings.Cut(rest, "/"); ok {
		switch tail {
		case actionResetVote:
			return h.resetVotes(w, r, loc, id)
		case actionActivate:
			return h.activate(w, r, loc, id)
		case actionActivity:
			return h.userActivity(w, r, loc, id)
		}
		notFound(w)
		return nil
	}
	if contains(userActions, rest) {
		if r.Method != http.MethodPost {
			return h.userAction(w, r, loc, rest)
		}
		if !h.verified(w, r) {
			return nil
		}
		switch rest {
		case actionNew:
			return h.saveNewUser(w, r, loc)
		case actionInvite:
			return h.saveInviteLink(w, r, loc)
		case actionClaim:
			return h.saveClaimLink(w, r, loc)
		case actionBot:
			return h.saveBot(w, r, loc)
		case actionMail:
			return h.sendInvite(w, r, loc, nil)
		}
	}
	if r.Method == http.MethodPost {
		return h.saveUser(w, r, loc, rest)
	}
	if rest == "" {
		return h.userList(w, r, loc)
	}
	return h.userForm(w, r, loc, rest, "")
}

func (h *Handler) userList(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	ctx := r.Context()
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	kind := r.URL.Query().Get("type")
	if !contains(userTypes, kind) {
		kind = ""
	}
	page := atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	found, total, err := h.deps.DB.AdminUsers(ctx, query, kind, perPage, (page-1)*perPage)
	if err != nil {
		return err
	}
	granted, _, err := h.access(ctx)
	if err != nil {
		return err
	}
	return h.page(w, r, loc, loc.T("admin.users"), "user_list.html", map[string]any{
		"Users":    found,
		"Query":    query,
		"Kind":     kind,
		"Types":    userTypes,
		"Page":     page,
		"Pages":    (total + perPage - 1) / perPage,
		"Total":    total,
		"SeeEmail": granted.Has(perms.ViewSensitiveInfo),
		"Base":     Prefix + userSlug + "/",
		"Action":   Prefix + userSlug + "/",
	})
}

func (h *Handler) userForm(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, rest, problem string) error {
	ctx := r.Context()
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		notFound(w)
		return nil
	}
	row, err := h.deps.DB.AdminUser(ctx, id)
	if errors.Is(err, db.ErrNotFound) {
		notFound(w)
		return nil
	}
	if err != nil {
		return err
	}
	allowed, err := h.mayEdit(ctx, row)
	if err != nil {
		return err
	}
	if !allowed {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return nil
	}

	roleList, err := h.deps.DB.AllRoles(ctx)
	if err != nil {
		return err
	}
	choices := make([]db.RoleChoice, 0, len(roleList))
	for _, one := range roleList {
		if !slices.Contains(builtinRoles, one.Slug) {
			choices = append(choices, one)
		}
	}
	granted, _, err := h.access(ctx)
	if err != nil {
		return err
	}
	mine := auth.FromContext(ctx)

	return h.page(w, r, loc, loc.T("admin.users"), "user_form.html", map[string]any{
		"User":        row,
		"Roles":       choices,
		"Held":        row.Roles,
		"MaySetRoles": h.maySetRoles(mine, granted, row),
		"ResetVotes":  Prefix + userSlug + "/" + rest + "/" + actionResetVote,
		"Activity":    Prefix + userSlug + "/" + rest + "/" + actionActivity,
		"Activate":    Prefix + userSlug + "/" + rest + "/" + actionActivate,
		"MaySuper":    mine != nil && mine.IsSuperuser,
		"SeeEmail":    granted.Has(perms.ViewSensitiveInfo),
		"CSRF":        csrf.Issue(w, r),
		"Error":       problem,
		"Action":      Prefix + userSlug + "/" + rest,
		"Back":        Prefix + userSlug + "/",
	})
}

func (h *Handler) saveUser(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, rest string) error {
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
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		notFound(w)
		return nil
	}
	stored, err := h.deps.DB.AdminUser(ctx, id)
	if errors.Is(err, db.ErrNotFound) {
		notFound(w)
		return nil
	}
	if err != nil {
		return err
	}
	allowed, err := h.mayEdit(ctx, stored)
	if err != nil {
		return err
	}
	if !allowed {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return nil
	}
	granted, _, err := h.access(ctx)
	if err != nil {
		return err
	}
	mine := auth.FromContext(ctx)

	next := stored
	next.Username = strings.TrimSpace(r.PostFormValue("username"))
	next.WikidotUsername = strings.TrimSpace(r.PostFormValue("wikidot_username"))
	next.DisplayName = strings.TrimSpace(r.PostFormValue("display_name"))
	next.Bio = r.PostFormValue("bio")
	next.IsActive = r.PostFormValue("is_active") != ""
	next.IsForumActive = r.PostFormValue("is_forum_active") != ""
	next.CanSendDM = r.PostFormValue("can_send_direct_messages") != ""
	next.InactiveUntil = optionalTime(r.PostFormValue("inactive_until"))
	next.ForumInactiveUntil = optionalTime(r.PostFormValue("forum_inactive_until"))
	if granted.Has(perms.ViewSensitiveInfo) {
		next.Email = strings.TrimSpace(r.PostFormValue("email"))
	}

	maySuper := mine != nil && mine.IsSuperuser
	if maySuper {
		next.IsSuperuser = r.PostFormValue("is_superuser") != ""
	}
	maySetRoles := h.maySetRoles(mine, granted, stored)
	if maySetRoles {
		next.Roles = nil
		for _, raw := range r.PostForm["roles"] {
			if got := optionalID(raw); got != nil {
				next.Roles = append(next.Roles, *got)
			}
		}
	}

	if next.Username == "" {
		return h.userForm(w, r, loc, rest, loc.T("admin.user-no-name"))
	}
	err = h.deps.DB.SaveAdminUser(ctx, next, builtinRoles, maySetRoles, maySuper)
	if err != nil {
		return err
	}
	h.noteID(r, db.AdminChanged, userSlug, next.ID, next.Username)
	redirect(w, Prefix+userSlug+"/")
	return nil
}

func (h *Handler) mayEdit(ctx context.Context, target db.AdminUserRow) (bool, error) {
	mine := auth.FromContext(ctx)
	if mine == nil {
		return false, nil
	}
	if mine.IsSuperuser {
		return true, nil
	}
	rank, err := h.deps.DB.OperationIndex(ctx, mine.ID)
	if err != nil {
		return false, err
	}
	return target.OperationIndex > rank, nil
}

func (h *Handler) maySetRoles(mine *db.User, granted perms.Set, target db.AdminUserRow) bool {
	if mine == nil {
		return false
	}
	if mine.IsSuperuser {
		return true
	}
	if target.IsSuperuser {
		return false
	}
	return granted.Has(perms.ManagePermissions)
}

func optionalTime(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04", "2006-01-02"} {
		if at, err := time.Parse(layout, raw); err == nil {
			return &at
		}
	}
	return nil
}
