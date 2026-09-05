package account

import (
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
	"github.com/WikitTeam/ProjectWikit/internal/shell"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/token"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
)

const AcceptPrefix = "/-/accept/"

type AcceptHandler struct {
	deps Deps
}

var _ http.Handler = (*AcceptHandler)(nil)

func NewAccept(d Deps) *AcceptHandler { return &AcceptHandler{deps: d} }

func (h *AcceptHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rest, ok := strings.CutPrefix(r.URL.Path, AcceptPrefix)
	if !ok {
		notFound(w)
		return
	}
	uid, secret, found := strings.Cut(strings.TrimSuffix(rest, "/"), "/")
	if !found || strings.Contains(secret, "/") {
		notFound(w)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, HEAD, POST")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if auth.FromContext(ctx) != nil {
		redirect(w, homePath, http.StatusFound)
		return
	}
	loc := h.deps.Bundle.Localizer(i18n.DefaultLanguage)
	csrfToken := csrf.Issue(w, r)

	form := shell.Accept{CSRF: csrfToken}
	invited, err := h.invited(r, uid, secret)
	if err != nil {
		h.deps.logger().Error("read invitation", "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	switch {
	case invited == nil:
		form.Error, form.Fatal = loc.T("accept.error-invalid"), true
	case invited.Type == db.UserTypeWikidot:
		form.IsWikidot, form.Username = true, invited.WikidotUsername
	}

	if r.Method == http.MethodPost && !form.Fatal {
		if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		done, err := h.accept(w, r, loc, invited, secret, &form)
		if err != nil {
			h.deps.logger().Error("accept invitation", "user", invited.ID, "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if done {
			redirect(w, homePath, http.StatusFound)
			return
		}
	}

	render := shell.New(loc, h.deps.Assets, h.deps.TimeZone)
	content, err := render.Accept(form)
	if err == nil {
		var body string
		body, err = h.deps.page(r, loc, current, loc.T("accept.title"), content)
		if err == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			if r.Method != http.MethodHead {
				_, _ = w.Write([]byte(body))
			}
			return
		}
	}
	h.deps.logger().Error("render invitation", "err", err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (h *AcceptHandler) invited(r *http.Request, uid, secret string) (*db.User, error) {
	ctx := r.Context()
	raw, err := base64.RawURLEncoding.DecodeString(uid)
	if err != nil {
		return nil, nil
	}
	id, err := strconv.ParseInt(string(raw), 10, 64)
	if err != nil {
		return nil, nil
	}
	user, err := h.deps.DB.UserByID(ctx, id)
	if errors.Is(err, db.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	used, err := h.deps.DB.TokenUsed(ctx, secret)
	if err != nil {
		return nil, err
	}
	if used || !h.deps.Tokens.Check(secret, token.InviteValue(user.ID, user.IsActive), time.Now()) {
		return nil, nil
	}
	return user, nil
}

func (h *AcceptHandler) accept(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer,
	invited *db.User, secret string, form *shell.Accept) (bool, error) {

	ctx := r.Context()
	name := invited.WikidotUsername
	display := (*string)(nil)
	if invited.Type != db.UserTypeWikidot {
		raw := wikidot.NormalizeDisplayName(r.PostFormValue("username"))
		form.Username = raw
		if problem := wikidot.ValidateDisplayName(raw); problem != wikidot.DisplayNameOK {
			form.Error = displayNameError(loc, problem)
			return false, nil
		}
		name = wikidot.CanonicalizeUsername(raw)
		if wikidot.ReservedUsername(name) {
			form.Error = loc.T("signup.error-name-reserved")
			return false, nil
		}
		if name == "" {
			free, err := h.freeFallback(r, invited.ID)
			if err != nil {
				return false, err
			}
			name = free
		}
		if raw != name {
			display = &raw
		}
	}
	form.Username = name

	if err := h.available(r, name, invited.ID); err != nil {
		if errors.Is(err, errNameTaken) {
			form.Error = loc.T("accept.error-name-taken")
			return false, nil
		}
		return false, err
	}

	plain := r.PostFormValue("password")
	if plain == "" {
		form.Error = loc.T("accept.error-password-required")
		return false, nil
	}
	if plain != r.PostFormValue("password2") {
		form.Error = loc.T("accept.error-password-mismatch")
		return false, nil
	}
	if err := password.Validate(plain, password.Attributes{Username: name}); err != nil {
		form.Error = passwordError(loc, err)
		return false, nil
	}
	hash, err := password.Hash(plain)
	if err != nil {
		return false, err
	}

	if err := h.deps.DB.ActivateUser(ctx, invited.ID, name, display, hash); err != nil {
		return false, err
	}
	if err := h.deps.DB.MarkTokenUsed(ctx, secret, true); err != nil {
		return false, err
	}
	if err := h.deps.DB.ActivateInviteLink(ctx, secret, name, time.Now()); err != nil {
		return false, err
	}
	if err := h.deps.welcome(ctx, invited.ID); err != nil {
		return false, err
	}
	user, err := h.deps.DB.UserByID(ctx, invited.ID)
	if err != nil {
		return false, err
	}
	return true, h.deps.signIn(ctx, w, r, user)
}

var errNameTaken = errors.New("account: that name belongs to somebody else")

func (h *AcceptHandler) available(r *http.Request, name string, own int64) error {
	found, err := h.deps.DB.UserByName(r.Context(), name)
	if errors.Is(err, db.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if found.ID != own {
		return errNameTaken
	}
	return nil
}

func (h *AcceptHandler) freeFallback(r *http.Request, id int64) (string, error) {
	for suffix := 1; suffix < fallbackAttempts; suffix++ {
		candidate := wikidot.FallbackUsername(id, suffix)
		taken, err := h.deps.DB.UsernameTaken(r.Context(), candidate)
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
	}
	return "", errors.New("account: no fallback username is free")
}
