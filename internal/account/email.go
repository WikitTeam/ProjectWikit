package account

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

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

type EmailHandler struct {
	deps Deps
}

var _ http.Handler = (*EmailHandler)(nil)

func NewEmail(d Deps) *EmailHandler { return &EmailHandler{deps: d} }

func (h *EmailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rest, ok := strings.CutPrefix(r.URL.Path, EmailPrefix)
	if !ok {
		notFound(w)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	purpose, rest, _ := strings.Cut(rest, "/")
	uid, secret, found := strings.Cut(strings.TrimSuffix(rest, "/"), "/")
	if !found || strings.Contains(secret, "/") {
		notFound(w)
		return
	}
	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	loc := h.deps.Bundle.For(ctx)

	outcome, err := h.act(r, purpose, uid, secret)
	if err != nil {
		h.deps.logger().Error("act on email link", "purpose", purpose, "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	body, err := h.deps.page(r, loc, current, loc.T("email."+outcome+"-title"),
		mustRender(shell.New(loc, h.deps.Assets).Notice(shell.Notice{
			AuthIcon:  authIcon(current),
			SiteTitle: current.Title,
			Heading:   loc.T("email." + outcome + "-title"),
			Body:      loc.T("email." + outcome),
			LinkURL:   noticeLink(outcome),
			LinkText:  loc.T(noticeLinkKey(outcome)),
		})))
	if err != nil {
		h.deps.logger().Error("render email notice", "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write([]byte(body))
	}
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

func mustRender(body string, err error) string {
	if err != nil {
		return ""
	}
	return body
}

func (h *EmailHandler) act(r *http.Request, purpose, uid, secret string) (string, error) {
	ctx := r.Context()
	user, state, err := h.target(ctx, uid)
	if err != nil || user == nil {
		return "dead", err
	}
	now := time.Now()

	switch purpose {
	case purposeVerify:
		if !h.deps.Tokens.Check(secret, verifyValue(user.ID, state), now) {
			return "dead", nil
		}
		done, err := h.deps.DB.MarkEmailVerified(ctx, user.ID, state.Email, now)
		if err != nil || !done {
			return "dead", err
		}
		return "verified", nil

	case purposeApprove:
		if state.Pending == "" || !h.deps.Tokens.Check(secret, token.Custom(purposeApprove, id(user.ID), lower(state.Pending)), now) {
			return "dead", nil
		}
		return "approved", h.deps.sendActivation(r, h.loc(ctx), user, state.Pending)

	case purposeActivate:
		if state.Pending == "" || !h.deps.Tokens.Check(secret, token.Custom(purposeActivate, id(user.ID), lower(state.Pending)), now) {
			return "dead", nil
		}
		taken, err := h.deps.DB.VerifiedEmailTaken(ctx, state.Pending, user.ID)
		if err != nil {
			return "", err
		}
		if taken {
			return "taken", nil
		}
		pending := state.Pending
		done, err := h.deps.DB.ApplyPendingEmail(ctx, user.ID, pending, now)
		if err != nil || !done {
			return "dead", err
		}
		if state.VerifiedAt != nil {
			if err := h.deps.warnPrevious(r, h.loc(ctx), user, state.Email, pending); err != nil {
				return "", err
			}
		}
		return "changed", nil

	case purposeRevert:
		if state.Previous == "" || !h.deps.Tokens.Check(secret, token.Custom(purposeRevert, id(user.ID), lower(state.Email), lower(state.Previous)), now) {
			return "dead", nil
		}
		done, err := h.deps.DB.RevertEmail(ctx, user.ID, state.Previous, now)
		if err != nil || !done {
			return "dead", err
		}
		if err := h.deps.retirePassword(ctx, user.ID); err != nil {
			return "", err
		}
		return "reverted", nil
	}
	return "dead", nil
}

func (h *EmailHandler) target(ctx context.Context, uid string) (*db.User, db.AccountEmail, error) {
	raw, err := base64.RawURLEncoding.DecodeString(uid)
	if err != nil {
		return nil, db.AccountEmail{}, nil
	}
	parsed, err := strconv.ParseInt(string(raw), 10, 64)
	if err != nil {
		return nil, db.AccountEmail{}, nil
	}
	user, err := h.deps.DB.UserByID(ctx, parsed)
	if errors.Is(err, db.ErrNotFound) {
		return nil, db.AccountEmail{}, nil
	}
	if err != nil {
		return nil, db.AccountEmail{}, err
	}
	state, err := h.deps.DB.AccountEmail(ctx, user.ID)
	if err != nil {
		return nil, db.AccountEmail{}, err
	}
	return user, state, nil
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
