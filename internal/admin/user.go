package admin

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/account"
	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/timezone"
)

const (
	userSlug = "users"
	perPage  = 50
)

var userTypes = []string{"normal", "wikidot", "bot", "system"}

const stateBanned = "banned"

func init() {
	register(screen{slug: userSlug, label: "admin.users", need: perms.ManageUsers, serve: (*Handler).users})
}

func (h *Handler) users(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, Prefix+userSlug), "/")
	if id, tail, ok := strings.Cut(rest, "/"); ok {
		switch tail {
		case actionResetVote:
			if !h.allowed(w, r, perms.ResetMemberVotes) {
				return nil
			}
			return h.resetVotes(w, r, loc, id)
		case actionActivate:
			if !h.allowed(w, r, perms.InviteMembers) {
				return nil
			}
			return h.activate(w, r, loc, id)
		case actionActivity:
			return h.userActivity(w, r, loc, id)
		}
		notFound(w)
		return nil
	}
	if contains(userActions, rest) {
		if !h.allowed(w, r, neededForAction(rest)) {
			return nil
		}
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
	roleList, err := h.deps.DB.AllRoles(ctx, siteID(ctx))
	if err != nil {
		return err
	}
	var roles []db.RoleChoice
	var roleID int64
	wanted := atoi(r.URL.Query().Get("role"))
	for _, one := range roleList {
		if slices.Contains(builtinRoles, one.Slug) {
			continue
		}
		roles = append(roles, one)
		if int64(wanted) == one.ID {
			roleID = one.ID
		}
	}
	state := r.URL.Query().Get("state")
	if state != stateBanned {
		state = ""
	}
	sensitive := seesSensitive(ctx)
	found, total, err := h.deps.DB.AdminUsers(ctx, siteID(ctx), query, kind, roleID, state == stateBanned, sensitive, perPage, (page-1)*perPage)
	if err != nil {
		return err
	}
	return h.page(w, r, loc, loc.T("admin.users"), "user_list.html", map[string]any{
		"Users":    found,
		"Query":    query,
		"Kind":     kind,
		"State":    state,
		"Banned":   stateBanned,
		"Now":      time.Now(),
		"Types":    userTypes,
		"Roles":    roles,
		"Role":     roleID,
		"Page":     page,
		"Pages":    (total + perPage - 1) / perPage,
		"Total":    total,
		"SeeEmail": sensitive,
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
	row, err := h.deps.DB.AdminUser(ctx, siteID(ctx), id)
	if errors.Is(err, db.ErrNotFound) {
		notFound(w)
		return nil
	}
	if err != nil {
		return err
	}
	lower, err := h.mayEdit(ctx, row)
	if err != nil {
		return err
	}

	roleList, err := h.deps.DB.AllRoles(ctx, siteID(ctx))
	if err != nil {
		return err
	}
	choices := assignableRoles(roleList)
	granted, _, err := h.access(ctx)
	if err != nil {
		return err
	}
	mine := auth.FromContext(ctx)
	here, err := h.superHere(ctx)
	if err != nil {
		return err
	}
	sensitive := seesSensitive(ctx)
	var hold db.AccountEmail
	if sensitive {
		h.accounts().SettleHolds(ctx)
		if hold, err = h.deps.DB.AccountEmail(ctx, row.ID); err != nil {
			return err
		}
	}

	sanctions, err := h.deps.DB.MemberSanctions(ctx, siteID(ctx), row.ID)
	if err != nil {
		return err
	}
	var sanctionList []sanctionRow
	if lower {
		sanctionList = sanctionRows(sanctions, granted)
	}
	var lastLogin *time.Time
	lastAddress := ""
	if sensitive {
		seen, err := h.deps.DB.LastSeen(ctx, row.ID)
		if err != nil {
			return err
		}
		lastLogin = seen.Login
		lastAddress = seen.Address
	}
	return h.page(w, r, loc, loc.T("admin.users"), "user_form.html", map[string]any{
		"User":         row,
		"Now":          time.Now(),
		"Zone":         site.Zone(ctx),
		"ZoneName":     site.Zone(ctx).String(),
		"Roles":        choices,
		"Held":         row.Roles,
		"MaySetRoles":  h.maySetRoles(mine, here, granted, row),
		"Sanctions":    sanctionList,
		"MaySanction":  len(sanctionList) > 0,
		"LockSelf":     mine != nil && !here && mine.ID == row.ID && holdsStaff(roleList, row.Roles),
		"MayAccount":   mine != nil && mine.IsSuperuser,
		"ResetVotes":   Prefix + userSlug + "/" + rest + "/" + actionResetVote,
		"MayReset":     lower && granted.Has(perms.ResetMemberVotes),
		"MayActivate":  row.Unclaimed && granted.Has(perms.InviteMembers),
		"Activity":     Prefix + userSlug + "/" + rest + "/" + actionActivity,
		"Activate":     Prefix + userSlug + "/" + rest + "/" + actionActivate,
		"MaySuper":     mine != nil && mine.IsSuperuser,
		"MaySiteSuper": here && !row.IsSuperuser,
		"DropSelf":     mine != nil && !mine.IsSuperuser && mine.ID == row.ID && !holdsStaff(roleList, row.Roles),
		"Hold":         hold,
		"HoldDays":     int(account.HoldPeriod / (24 * time.Hour)),
		"Done":         userDone(loc, r.URL.Query().Get("said")),
		"SeeEmail":     sensitive,
		"SeeKey":       sensitive || (row.BotOfSite && granted.Has(perms.ManageBots)),
		"LastLogin":    lastLogin,
		"LastAddress":  lastAddress,
		"CSRF":         csrf.Issue(w, r),
		"Error":        problem,
		"Action":       Prefix + userSlug + "/" + rest,
		"Back":         Prefix + userSlug + "/",
		"Access":       h.accessHref(ctx, "?user="+url.QueryEscape(row.Username)),
	})
}

func (h *Handler) saveUser(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, rest string) error {
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
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		notFound(w)
		return nil
	}
	stored, err := h.deps.DB.AdminUser(ctx, siteID(ctx), id)
	if errors.Is(err, db.ErrNotFound) {
		notFound(w)
		return nil
	}
	if err != nil {
		return err
	}
	lower, err := h.mayEdit(ctx, stored)
	if err != nil {
		return err
	}
	granted, _, err := h.access(ctx)
	if err != nil {
		return err
	}
	mine := auth.FromContext(ctx)
	here, err := h.superHere(ctx)
	if err != nil {
		return err
	}
	maySuper := mine != nil && mine.IsSuperuser
	if maySuper && r.PostFormValue("hold") == "cancel" {
		target, err := h.deps.DB.UserByID(ctx, stored.ID)
		if err != nil {
			return err
		}
		if _, err := h.accounts().CancelHold(ctx, target, mine, account.HoldCancelled); err != nil {
			return err
		}
		redirect(w, Prefix+userSlug+"/"+rest+"?said="+saidHoldCancelled)
		return nil
	}

	next := stored
	// The account itself is shared by every site of the instance, so only a
	// superuser changes it.
	wantedEmail := ""
	if maySuper {
		zone := editorZone(r)
		next.Username = strings.TrimSpace(r.PostFormValue("username"))
		next.WikidotUsername = strings.TrimSpace(r.PostFormValue("wikidot_username"))
		next.DisplayName = strings.TrimSpace(r.PostFormValue("display_name"))
		next.Bio = r.PostFormValue("bio")
		next.IsActive = r.PostFormValue("is_active") != ""
		next.IsForumActive = r.PostFormValue("is_forum_active") != ""
		next.CanSendDM = r.PostFormValue("can_send_direct_messages") != ""
		next.InactiveUntil = untilUnless(next.IsActive, r.PostFormValue("inactive_until"), zone)
		next.ForumInactiveUntil = untilUnless(next.IsForumActive, r.PostFormValue("forum_inactive_until"), zone)
		next.DMUntil = untilUnless(next.CanSendDM, r.PostFormValue("direct_messages_until"), zone)
		next.IsSuperuser = r.PostFormValue("is_superuser") != ""
		if seesSensitive(ctx) {
			wantedEmail = strings.TrimSpace(r.PostFormValue("email"))
		}
	}
	maySetRoles := h.maySetRoles(mine, here, granted, stored)
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
	held := false
	if wantedEmail != "" && !strings.EqualFold(wantedEmail, stored.Email) {
		state, err := h.deps.DB.AccountEmail(ctx, stored.ID)
		if err != nil {
			return err
		}
		if !strings.EqualFold(wantedEmail, state.Held) {
			target, err := h.deps.DB.UserByID(ctx, stored.ID)
			if err != nil {
				return err
			}
			switch err := h.accounts().HoldEmail(r, loc, target, wantedEmail, mine); {
			case errors.Is(err, account.ErrHoldInvalid):
				return h.userForm(w, r, loc, rest, loc.T("admin.email-invalid"))
			case errors.Is(err, account.ErrHoldTaken):
				return h.userForm(w, r, loc, rest, loc.T("admin.invite-email-taken"))
			case err != nil && !errors.Is(err, account.ErrHoldSame):
				return err
			case err == nil:
				held = true
			}
		}
	}
	err = h.deps.DB.SaveAdminUser(ctx, siteID(ctx), next, builtinRoles, maySetRoles, maySuper)
	if err != nil {
		return err
	}
	if here && !stored.IsSuperuser {
		on := r.PostFormValue("site_superuser") != ""
		changed, err := h.deps.DB.SetSiteSuperuser(ctx, siteID(ctx), stored.ID, on, mine.ID, time.Now())
		if err != nil {
			return err
		}
		if changed {
			action := actionSiteSuperOn
			if !on {
				action = actionSiteSuperOff
			}
			h.noteID(r, action, userSlug, stored.ID, stored.Username)
		}
	}
	if lower {
		if err := h.saveSanctions(r, next.ID); err != nil {
			return err
		}
	}
	h.noteID(r, db.AdminChanged, userSlug, next.ID, next.Username)
	if held {
		redirect(w, Prefix+userSlug+"/"+rest+"?said="+saidHoldSent)
		return nil
	}
	redirect(w, Prefix+userSlug+"/")
	return nil
}

const (
	actionSiteSuperOn  = "site-super-on"
	actionSiteSuperOff = "site-super-off"

	saidHoldSent      = "hold-sent"
	saidHoldCancelled = "hold-cancelled"
)

func userDone(loc *i18n.Localizer, said string) string {
	switch said {
	case saidHoldSent:
		return loc.T("admin.email-hold-sent", "days", int(account.HoldPeriod/(24*time.Hour)))
	case saidHoldCancelled:
		return loc.T("admin.email-hold-cancelled")
	}
	return ""
}

func (h *Handler) accounts() account.Deps {
	return account.Deps{DB: h.deps.DB, Tokens: h.deps.Tokens, Mail: h.deps.Mail,
		Trust: h.deps.Trust, Bundle: h.deps.Bundle, Log: h.deps.Log}
}

// Addresses and keys belong to the account, which every site of the instance
// shares, so a site's own administrators never see them.
func seesSensitive(ctx context.Context) bool {
	mine := auth.FromContext(ctx)
	return mine != nil && mine.IsSuperuser && mine.ActiveAt(time.Now())
}

func (h *Handler) superHere(ctx context.Context) (bool, error) {
	mine := auth.FromContext(ctx)
	if mine == nil || !mine.ActiveAt(time.Now()) {
		return false, nil
	}
	if mine.IsSuperuser {
		return true, nil
	}
	return h.deps.DB.IsSiteSuperuser(ctx, siteID(ctx), mine.ID)
}

func holdsStaff(all []db.RoleChoice, held []int64) bool {
	for _, one := range all {
		if one.IsStaff && slices.Contains(held, one.ID) {
			return true
		}
	}
	return false
}

func (h *Handler) mayEdit(ctx context.Context, target db.AdminUserRow) (bool, error) {
	mine := auth.FromContext(ctx)
	if mine == nil {
		return false, nil
	}
	if mine.IsSuperuser {
		return true, nil
	}
	if target.IsSuperuser {
		return false, nil
	}
	here, err := h.superHere(ctx)
	if err != nil || here {
		return here, err
	}
	if target.SiteSuperuser {
		return false, nil
	}
	rank, err := h.deps.DB.OperationIndex(ctx, siteID(ctx), mine.ID)
	if err != nil {
		return false, err
	}
	return target.OperationIndex > rank, nil
}

func (h *Handler) maySetRoles(mine *db.User, here bool, granted perms.Set, target db.AdminUserRow) bool {
	if mine == nil {
		return false
	}
	if mine.IsSuperuser {
		return true
	}
	if target.IsSuperuser {
		return false
	}
	if here {
		return true
	}
	if target.SiteSuperuser {
		return false
	}
	return granted.Has(perms.ManagePermissions)
}

func editorZone(r *http.Request) *time.Location {
	if name := r.PostFormValue("editor_zone"); timezone.Valid(name) {
		return timezone.Load(name)
	}
	return site.Zone(r.Context())
}

// Ticking the box drops the deadline. Left behind, a deadline would keep the
// account switched off whatever the box says.
func untilUnless(on bool, raw string, zone *time.Location) *time.Time {
	if on {
		return nil
	}
	return optionalTime(raw, zone)
}

func optionalTime(raw string, zone *time.Location) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04", "2006-01-02"} {
		if at, err := time.ParseInLocation(layout, raw, zone); err == nil {
			return &at
		}
	}
	return nil
}
