package account

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/shell"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/token"
)

const EmailPrefix = "/-/email/"

const (
	purposeVerify   = "verify"
	purposeApprove  = "approve"
	purposeActivate = "activate"
	purposeRevert   = "revert"
)

const bindingCooldown = 7 * 24 * time.Hour

const settingsEmailPath = profilePath + "?tab=email"

type EmailHandler struct {
	deps Deps
}

var _ http.Handler = (*EmailHandler)(nil)

func NewEmail(d Deps) *EmailHandler { return &EmailHandler{deps: d} }

// Mail scanners open every link they find, so opening one only asks and the
// button underneath is what acts.
func (h *EmailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rest, ok := strings.CutPrefix(r.URL.Path, EmailPrefix)
	if !ok {
		h.deps.notFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, HEAD, POST")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	purpose, rest, _ := strings.Cut(rest, "/")
	uid, secret, found := strings.Cut(strings.TrimSuffix(rest, "/"), "/")
	if !found || strings.Contains(secret, "/") {
		h.deps.notFound(w, r)
		return
	}
	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	loc := h.deps.Bundle.For(ctx)

	if r.Method != http.MethodPost {
		n, err := h.ask(w, r, loc, purpose, uid, secret)
		if err != nil {
			h.deps.logger().Error("read email link", "purpose", purpose, "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		h.deps.notice(w, r, loc, current, http.StatusOK, n)
		return
	}

	if err := csrf.Verify(r, []string{current.Domain}); err != nil {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	done, err := h.act(w, r, purpose, uid, secret)
	if err != nil {
		h.deps.logger().Error("act on email link", "purpose", purpose, "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	outcome := done.outcome
	me := int64(0)
	if viewer := auth.FromContext(ctx); viewer != nil {
		me = viewer.ID
	} else if done.signedIn {
		me = done.user.ID
	}
	if outcome == "verified" && me != 0 && me == done.user.ID {
		redirect(w, homePath, http.StatusSeeOther)
		return
	}

	when := ""
	if outcome == "held" {
		if user, state, err := h.deps.userByUID(ctx, uid); err == nil && user != nil && state.HeldUntil != nil {
			when = h.deps.When(ctx, user.ID, *state.HeldUntil)
		}
	}
	n := shell.Notice{
		Heading:  loc.T("email." + outcome + "-title"),
		Body:     loc.T("email."+outcome, "when", when),
		LinkURL:  noticeLink(outcome),
		LinkText: loc.T(noticeLinkKey(outcome)),
	}
	if outcome == "approved" || outcome == "changed" {
		n.GoURL, n.ActionText, n.LinkURL = LoginPath+"?to="+url.QueryEscape(settingsEmailPath), loc.T("account.continue"), ""
		if me != 0 && me == done.user.ID {
			n.GoURL = settingsEmailPath
		}
	}
	h.deps.notice(w, r, loc, current, http.StatusOK, n)
}

type acted struct {
	outcome  string
	user     *db.User
	signedIn bool
}

func (h *EmailHandler) ask(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, purpose, uid, secret string) (shell.Notice, error) {
	ctx := r.Context()
	h.deps.SettleHolds(ctx)
	dead := shell.Notice{
		Heading: loc.T("email.dead-title"), Body: loc.T("email.dead"),
		LinkURL: LoginPath, LinkText: loc.T("reset.back-login"),
	}
	user, state, err := h.deps.userByUID(ctx, uid)
	if err != nil || user == nil {
		return dead, err
	}
	g, value, shown := h.expected(purpose, user, state)
	if value == nil || !g.Check(secret, value, time.Now()) {
		return dead, nil
	}
	return shell.Notice{
		Heading:    loc.T("email." + purpose + "-ask-title"),
		Body:       loc.T("email."+purpose+"-ask", "email", shown, "days", holdDays()),
		ActionURL:  r.URL.Path,
		ActionText: loc.T("account.continue"),
		CSRF:       csrf.Issue(w, r),
	}, nil
}

// Each purpose is bound to the state it was sent for, so a link stops working
// once that state has moved on.
func (h *EmailHandler) expected(purpose string, user *db.User, state db.AccountEmail) (token.Generator, token.Value, string) {
	g := h.deps.Tokens
	switch purpose {
	case purposeVerify:
		return g, verifyValue(user.ID, state), state.Email
	case purposeApprove:
		if state.Pending != "" {
			return g, token.Custom(purposeApprove, id(user.ID), lower(state.Pending)), state.Pending
		}
	case purposeActivate:
		if state.Pending != "" {
			return g, token.Custom(purposeActivate, id(user.ID), lower(state.Pending)), state.Pending
		}
	case purposeRevert:
		if state.Previous != "" {
			return g, token.Custom(purposeRevert, id(user.ID), lower(state.Email), lower(state.Previous)), state.Previous
		}
	case purposeHold:
		if state.Held != "" && state.HeldUntil == nil {
			return g, token.Custom(purposeHold, id(user.ID), lower(state.Held)), state.Held
		}
	case purposeCancel:
		if state.Held != "" {
			return h.deps.holdTokens(), token.Custom(purposeCancel, id(user.ID), lower(state.Held), lower(state.Email)), masked(state.Held)
		}
	}
	return g, nil, ""
}

func noticeLink(outcome string) string {
	if outcome == "reverted" {
		return ResetPath
	}
	return LoginPath
}

func noticeLinkKey(outcome string) string {
	if outcome == "reverted" {
		return "email.reverted-link"
	}
	return "reset.back-login"
}

func (h *EmailHandler) act(w http.ResponseWriter, r *http.Request, purpose, uid, secret string) (acted, error) {
	ctx := r.Context()
	h.deps.SettleHolds(ctx)
	user, state, err := h.deps.userByUID(ctx, uid)
	if err != nil || user == nil {
		return acted{outcome: "dead"}, err
	}
	now := time.Now()
	g, value, _ := h.expected(purpose, user, state)
	if value == nil || !g.Check(secret, value, now) {
		return acted{outcome: "dead", user: user}, nil
	}
	outcome, signedIn, err := h.apply(w, r, purpose, user, state, secret, now)
	return acted{outcome: outcome, user: user, signedIn: signedIn}, err
}

func (h *EmailHandler) apply(w http.ResponseWriter, r *http.Request, purpose string, user *db.User,
	state db.AccountEmail, secret string, now time.Time) (string, bool, error) {

	ctx := r.Context()
	plain := func(outcome string, err error) (string, bool, error) { return outcome, false, err }

	switch purpose {
	case purposeHold:
		return plain(h.deps.confirmHold(r, user, state, secret, now))

	case purposeCancel:
		return plain(h.deps.cancelHold(ctx, user, state, secret, now))

	case purposeVerify:
		done, err := h.deps.DB.MarkEmailVerified(ctx, user.ID, state.Email, now)
		if err != nil || !done {
			return plain("dead", err)
		}
		signedIn, err := h.deps.carryOn(w, r, user)
		return "verified", signedIn, err

	case purposeApprove:
		return plain("approved", h.deps.sendActivation(r, h.loc(ctx), user, state.Pending))

	case purposeActivate:
		taken, err := h.deps.DB.VerifiedEmailTaken(ctx, state.Pending, user.ID)
		if err != nil || taken {
			return plain("taken", err)
		}
		pending := state.Pending
		done, err := h.deps.DB.ApplyPendingEmail(ctx, user.ID, pending, now)
		if err != nil || !done {
			return plain("dead", err)
		}
		h.deps.holdOverridden(ctx, user, state, HoldDroppedChanged)
		if state.VerifiedAt != nil {
			if err := h.deps.warnPrevious(r, h.loc(ctx), user, state.Email, pending); err != nil {
				return plain("", err)
			}
		}
		signedIn, err := h.deps.carryOn(w, r, user)
		return "changed", signedIn, err

	case purposeRevert:
		done, err := h.deps.DB.RevertEmail(ctx, user.ID, state.Previous, now)
		if err != nil || !done {
			return plain("dead", err)
		}
		h.deps.holdOverridden(ctx, user, state, HoldDroppedReverted)
		return plain("reverted", h.deps.retirePassword(ctx, user.ID))
	}
	return plain("dead", nil)
}

func verifyValue(userID int64, state db.AccountEmail) token.Value {
	stamped := ""
	if state.VerifiedAt != nil {
		stamped = state.VerifiedAt.UTC().Format(time.RFC3339)
	}
	return token.Custom(purposeVerify, id(userID), lower(state.Email), stamped)
}

func (h *EmailHandler) loc(ctx context.Context) *i18n.Localizer {
	return h.deps.Bundle.For(ctx)
}

func id(n int64) string { return strconv.FormatInt(n, 10) }

func lower(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
