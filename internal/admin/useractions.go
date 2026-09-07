package admin

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/password"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/token"
)

const (
	actionNew       = "new"
	actionMail      = "invite"
	actionInvite    = "invite-link"
	actionClaim     = "claim-link"
	actionBot       = "bot"
	actionResetVote = "reset-votes"
	actionActivate  = "activate"
)

const (
	inviteKindRegister = "register"
	inviteKindClaim    = "claim"
	inviteByLink       = "link"
	inviteByMail       = "email"
	acceptPrefix       = "/-/accept/"
)

var userActions = []string{actionNew, actionMail, actionInvite, actionClaim, actionBot}

func (h *Handler) userAction(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, what string) error {
	switch what {
	case actionNew:
		return h.newUser(w, r, loc, "")
	case actionInvite:
		return h.inviteLink(w, r, loc, "", "")
	case actionClaim:
		return h.claimLink(w, r, loc, "", "")
	case actionBot:
		return h.newBot(w, r, loc, "")
	case actionMail:
		return h.mailInvite(w, r, loc, nil, "", "")
	}
	notFound(w)
	return nil
}

func (h *Handler) newUser(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, problem string) error {
	roleList, err := h.deps.DB.AllRoles(r.Context())
	if err != nil {
		return err
	}
	return h.page(w, r, loc, loc.T("admin.new-user"), "user_new.html", map[string]any{
		"Roles":  h.grantableRoles(r, roleList),
		"CSRF":   csrf.Issue(w, r),
		"Error":  problem,
		"Action": Prefix + userSlug + "/" + actionNew,
		"Back":   Prefix + userSlug + "/",
	})
}

func (h *Handler) saveNewUser(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	ctx := r.Context()
	name := strings.TrimSpace(r.PostFormValue("username"))
	secret := r.PostFormValue("password")
	switch {
	case name == "":
		return h.newUser(w, r, loc, loc.T("admin.user-no-name"))
	case len(secret) < 8:
		return h.newUser(w, r, loc, loc.T("admin.password-too-short"))
	}
	taken, err := h.deps.DB.UsernameTaken(ctx, name)
	if err != nil {
		return err
	}
	if taken {
		return h.newUser(w, r, loc, loc.T("admin.name-taken"))
	}
	hash, err := password.Hash(secret)
	if err != nil {
		return err
	}
	id, err := h.deps.DB.CreateUser(ctx, name, strings.TrimSpace(r.PostFormValue("display_name")),
		hash, r.PostFormValue("is_active") != "", time.Now())
	if err != nil {
		return err
	}
	if err := h.grantPicked(r, id); err != nil {
		return err
	}
	h.noteID(r, db.AdminCreated, userSlug, id, name)
	redirect(w, Prefix+userSlug+"/"+strconv.FormatInt(id, 10))
	return nil
}

func (h *Handler) newBot(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, problem string) error {
	return h.page(w, r, loc, loc.T("admin.new-bot"), "user_bot.html", map[string]any{
		"CSRF":   csrf.Issue(w, r),
		"Error":  problem,
		"Action": Prefix + userSlug + "/" + actionBot,
		"Back":   Prefix + userSlug + "/",
	})
}

func (h *Handler) saveBot(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	ctx := r.Context()
	name := strings.TrimSpace(r.PostFormValue("username"))
	if name == "" {
		return h.newBot(w, r, loc, loc.T("admin.user-no-name"))
	}
	taken, err := h.deps.DB.UsernameTaken(ctx, name)
	if err != nil {
		return err
	}
	if taken {
		return h.newBot(w, r, loc, loc.T("admin.name-taken"))
	}
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return err
	}
	id, err := h.deps.DB.CreateBot(ctx, name, base64.RawURLEncoding.EncodeToString(raw[:]), time.Now())
	if err != nil {
		return err
	}
	h.noteID(r, db.AdminCreated, userSlug, id, name)
	redirect(w, Prefix+userSlug+"/"+strconv.FormatInt(id, 10))
	return nil
}

func (h *Handler) inviteLink(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, problem, made string) error {
	roleList, err := h.deps.DB.AllRoles(r.Context())
	if err != nil {
		return err
	}
	return h.page(w, r, loc, loc.T("admin.new-invite-link"), "user_invite.html", map[string]any{
		"Roles":  h.grantableRoles(r, roleList),
		"CSRF":   csrf.Issue(w, r),
		"Error":  problem,
		"Link":   made,
		"Action": Prefix + userSlug + "/" + actionInvite,
		"Back":   Prefix + inviteSlug + "/",
	})
}

func (h *Handler) saveInviteLink(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	ctx := r.Context()
	email := strings.TrimSpace(r.PostFormValue("email"))
	if email == "" {
		return h.inviteLink(w, r, loc, loc.T("admin.invite-no-email"), "")
	}
	if _, err := h.deps.DB.UserByEmail(ctx, email); err == nil {
		return h.inviteLink(w, r, loc, loc.T("admin.invite-email-taken"), "")
	} else if !errors.Is(err, db.ErrNotFound) {
		return err
	}
	now := time.Now()
	id, err := h.deps.DB.CreateInvitedUser(ctx, email, now)
	if err != nil {
		return err
	}
	if err := h.grantPicked(r, id); err != nil {
		return err
	}
	link, err := h.mintLink(w, r, inviteKindRegister, id, email, "", now)
	if err != nil {
		return err
	}
	h.noteID(r, db.AdminCreated, inviteSlug, id, email)
	return h.inviteLink(w, r, loc, "", link)
}

func (h *Handler) claimLink(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, problem, made string) error {
	ctx := r.Context()
	waiting, err := h.deps.DB.UnclaimedWikidotUsers(ctx)
	if err != nil {
		return err
	}
	roleList, err := h.deps.DB.AllRoles(ctx)
	if err != nil {
		return err
	}
	return h.page(w, r, loc, loc.T("admin.new-claim-link"), "user_claim.html", map[string]any{
		"Waiting": waiting,
		"Roles":   h.grantableRoles(r, roleList),
		"CSRF":    csrf.Issue(w, r),
		"Error":   problem,
		"Link":    made,
		"Action":  Prefix + userSlug + "/" + actionClaim,
		"Back":    Prefix + inviteSlug + "/",
	})
}

func (h *Handler) saveClaimLink(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) error {
	ctx := r.Context()
	id, err := strconv.ParseInt(r.PostFormValue("user"), 10, 64)
	if err != nil {
		return h.claimLink(w, r, loc, loc.T("admin.claim-no-user"), "")
	}
	row, err := h.deps.DB.AdminUser(ctx, id)
	if errors.Is(err, db.ErrNotFound) || row.Type != db.UserTypeWikidot || row.IsActive {
		return h.claimLink(w, r, loc, loc.T("admin.claim-no-user"), "")
	}
	if err != nil {
		return err
	}
	if err := h.grantPicked(r, id); err != nil {
		return err
	}
	link, err := h.mintLink(w, r, inviteKindClaim, id, "", row.WikidotUsername, time.Now())
	if err != nil {
		return err
	}
	h.noteID(r, db.AdminCreated, inviteSlug, id, row.WikidotUsername)
	return h.claimLink(w, r, loc, "", link)
}

func (h *Handler) mintLink(w http.ResponseWriter, r *http.Request, kind string, id int64, email, wikidotName string, now time.Time) (string, error) {
	return h.mintLinkAs(w, r, kind, inviteByLink, id, email, wikidotName, now)
}

func (h *Handler) mintLinkAs(w http.ResponseWriter, r *http.Request, kind, delivery string, id int64, email, wikidotName string, now time.Time) (string, error) {
	ctx := r.Context()
	current := site.FromContext(ctx)
	minted := h.deps.Tokens.Make(token.InviteValue(id, false), now)
	uid := base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(id, 10)))

	var owner *int64
	if by := auth.FromContext(ctx); by != nil {
		owner = &by.ID
	}
	if _, err := h.deps.DB.CreateInviteLink(ctx, kind, delivery, email, wikidotName,
		minted, uid, owner, id, now); err != nil {
		return "", err
	}
	return scheme(r) + "://" + current.Domain + acceptPrefix + uid + "/" + minted, nil
}

func (h *Handler) resetVotes(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, rest string) error {
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
	if r.Method != http.MethodPost {
		return h.page(w, r, loc, loc.T("admin.reset-votes"), "user_reset_votes.html", map[string]any{
			"User":   row,
			"CSRF":   csrf.Issue(w, r),
			"Action": Prefix + userSlug + "/" + rest + "/" + actionResetVote,
			"Back":   Prefix + userSlug + "/" + rest,
		})
	}
	if !h.verified(w, r) {
		return nil
	}
	if _, err := h.deps.DB.ResetUserVotes(ctx, id); err != nil {
		return err
	}
	h.noteID(r, db.AdminChanged, userSlug, id, row.Username)
	redirect(w, Prefix+userSlug+"/"+rest)
	return nil
}

func (h *Handler) grantableRoles(r *http.Request, all []db.RoleChoice) []db.RoleChoice {
	if !grantsFrom(r.Context()).Has(perms.ManagePermissions) {
		return nil
	}
	out := make([]db.RoleChoice, 0, len(all))
	for _, one := range all {
		if !contains(builtinRoles, one.Slug) {
			out = append(out, one)
		}
	}
	return out
}

func (h *Handler) grantPicked(r *http.Request, userID int64) error {
	if !grantsFrom(r.Context()).Has(perms.ManagePermissions) {
		return nil
	}
	for _, raw := range r.PostForm["roles"] {
		role, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			continue
		}
		if err := h.deps.DB.GrantRole(r.Context(), userID, role); err != nil {
			return err
		}
	}
	return nil
}

func scheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func (h *Handler) mailInvite(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer,
	target *db.AdminUserRow, problem, done string) error {

	roleList, err := h.deps.DB.AllRoles(r.Context())
	if err != nil {
		return err
	}
	action := Prefix + userSlug + "/" + actionMail
	name := ""
	if target != nil {
		action = Prefix + userSlug + "/" + strconv.FormatInt(target.ID, 10) + "/" + actionActivate
		name = target.WikidotUsername
		if name == "" {
			name = target.Username
		}
	}
	return h.page(w, r, loc, loc.T("admin.mail-invite"), "user_mail.html", map[string]any{
		"Roles":  h.grantableRoles(r, roleList),
		"Target": name,
		"Email":  strings.TrimSpace(r.PostFormValue("email")),
		"CSRF":   csrf.Issue(w, r),
		"Error":  problem,
		"Done":   done,
		"Action": action,
		"Back":   Prefix + userSlug + "/",
	})
}

func (h *Handler) sendInvite(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, target *db.AdminUserRow) error {
	ctx := r.Context()
	if h.deps.Mail == nil {
		return h.mailInvite(w, r, loc, target, loc.T("admin.no-mailer"), "")
	}
	email := strings.TrimSpace(r.PostFormValue("email"))
	if email == "" {
		return h.mailInvite(w, r, loc, target, loc.T("admin.invite-no-email"), "")
	}
	if held, err := h.deps.DB.UserByEmail(ctx, email); err == nil {
		if target == nil || held.ID != target.ID {
			return h.mailInvite(w, r, loc, target, loc.T("admin.invite-email-taken"), "")
		}
	} else if !errors.Is(err, db.ErrNotFound) {
		return err
	}

	now := time.Now()
	kind, wikidotName := inviteKindRegister, ""
	id := int64(0)
	if target == nil {
		fresh, err := h.deps.DB.CreateInvitedUser(ctx, email, now)
		if err != nil {
			return err
		}
		id = fresh
	} else {
		id = target.ID
		if err := h.deps.DB.SetEmail(ctx, id, email); err != nil {
			return err
		}
		if target.Type == db.UserTypeWikidot {
			kind, wikidotName = inviteKindClaim, target.WikidotUsername
		}
	}
	if err := h.grantPicked(r, id); err != nil {
		return err
	}

	link, err := h.mintLinkAs(w, r, kind, inviteByMail, id, email, wikidotName, now)
	if err != nil {
		return err
	}
	current := site.FromContext(ctx)
	err = h.deps.Mail.Send(ctx, []string{email},
		loc.T("email.invite-subject", "site", current.Title),
		loc.T("email.invite-body", "link", link, "site", current.Title))
	if err != nil {
		h.deps.logger().Error("send invitation", "err", err)
		return h.mailInvite(w, r, loc, target, loc.T("admin.invite-not-sent"), "")
	}
	h.noteID(r, db.AdminCreated, inviteSlug, id, email)
	return h.mailInvite(w, r, loc, target, "", email)
}

func (h *Handler) activate(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, rest string) error {
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		notFound(w)
		return nil
	}
	row, err := h.deps.DB.AdminUser(r.Context(), id)
	if errors.Is(err, db.ErrNotFound) {
		notFound(w)
		return nil
	}
	if err != nil {
		return err
	}
	if r.Method != http.MethodPost {
		return h.mailInvite(w, r, loc, &row, "", "")
	}
	if !h.verified(w, r) {
		return nil
	}
	return h.sendInvite(w, r, loc, &row)
}
