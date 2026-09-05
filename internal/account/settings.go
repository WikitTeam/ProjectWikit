package account

import (
	"context"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/password"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
)

const (
	SettingsPrefix = "/-/profile/"
	profilePath    = "/-/profile/edit"

	emailPath    = "/-/profile/email"
	passwordPath = "/-/profile/password"
	namePath     = "/-/profile/name"
)

const RenameCooldown = 30 * 24 * time.Hour

var outcomes = []string{
	"email-sent", "email-approval-sent", "email-verified-sent", "email-taken",
	"email-invalid", "email-same", "email-needs-password",
	"password-changed", "password-wrong", "password-unusable", "password-mismatch", "password-weak",
	"name-changed", "name-cooldown", "name-taken", "name-invalid",
}

type SettingsHandler struct {
	deps Deps
}

var _ http.Handler = (*SettingsHandler)(nil)

func NewSettings(d Deps) *SettingsHandler { return &SettingsHandler{deps: d} }

func Outcome(raw string) string {
	if slices.Contains(outcomes, raw) {
		return raw
	}
	return ""
}

func (h *SettingsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	current := site.FromContext(ctx)
	user := auth.FromContext(ctx)
	if current == nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if user == nil {
		redirect(w, LoginPath+"?to="+url.QueryEscape(profilePath), http.StatusFound)
		return
	}
	if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	loc := h.deps.Bundle.Localizer(i18n.DefaultLanguage)

	var outcome string
	var err error
	switch r.URL.Path {
	case emailPath:
		outcome, err = h.email(r, loc, user)
	case passwordPath:
		outcome, err = h.password(w, r, loc, user)
	case namePath:
		outcome, err = h.name(r, user)
	default:
		notFound(w)
		return
	}
	if err != nil {
		h.deps.logger().Error("change account settings", "path", r.URL.Path, "user", user.ID, "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	redirect(w, profilePath+"?said="+url.QueryEscape(outcome), http.StatusSeeOther)
}

func (h *SettingsHandler) email(r *http.Request, loc *i18n.Localizer, user *db.User) (string, error) {
	ctx := r.Context()
	state, err := h.deps.DB.AccountEmail(ctx, user.ID)
	if err != nil {
		return "", err
	}
	wanted := lower(r.PostFormValue("email"))
	if !plausibleEmail(wanted) {
		return "email-invalid", nil
	}

	if wanted == lower(state.Email) {
		if state.VerifiedAt != nil {
			return "email-same", nil
		}
		return "email-verified-sent", h.deps.sendVerification(ctx, loc, user, state)
	}

	taken, err := h.deps.DB.VerifiedEmailTaken(ctx, wanted, user.ID)
	if err != nil {
		return "", err
	}
	if taken {
		return "email-taken", nil
	}

	proven, err := h.knowsPassword(ctx, user, r.PostFormValue("password"))
	if err != nil {
		return "", err
	}
	if !proven && state.VerifiedAt == nil {
		return "email-needs-password", nil
	}
	if err := h.deps.DB.SetPendingEmail(ctx, user.ID, wanted); err != nil {
		return "", err
	}
	if proven {
		return "email-sent", h.deps.sendActivation(ctx, loc, user, wanted)
	}
	return "email-approval-sent", h.deps.sendApproval(ctx, loc, user, state.Email, wanted)
}

func (h *SettingsHandler) password(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, user *db.User) (string, error) {
	ctx := r.Context()
	_, stored, err := h.deps.DB.UserForSession(ctx, user.ID)
	if err != nil {
		return "", err
	}
	if !password.IsUsable(stored) {
		return "password-unusable", nil
	}
	ok, err := password.Verify(r.PostFormValue("current"), stored)
	if err != nil || !ok {
		return "password-wrong", nil
	}
	fresh := r.PostFormValue("new1")
	if fresh != r.PostFormValue("new2") {
		return "password-mismatch", nil
	}
	if err := password.Validate(fresh, aboutUser(user)); err != nil {
		return "password-weak", nil
	}
	hash, err := password.Hash(fresh)
	if err != nil {
		return "", err
	}
	if err := h.deps.DB.SetPassword(ctx, user.ID, hash); err != nil {
		return "", err
	}
	if err := h.deps.signIn(ctx, w, r, user); err != nil {
		return "", err
	}
	state, err := h.deps.DB.AccountEmail(ctx, user.ID)
	if err != nil {
		return "", err
	}
	if state.VerifiedAt == nil {
		return "password-changed", nil
	}
	return "password-changed", h.deps.warnPasswordChanged(ctx, loc, user, state.Email)
}

func (h *SettingsHandler) name(r *http.Request, user *db.User) (string, error) {
	ctx := r.Context()
	last, err := h.deps.DB.UsernameChangedAt(ctx, user.ID)
	if err != nil {
		return "", err
	}
	if last != nil && time.Since(*last) < RenameCooldown {
		return "name-cooldown", nil
	}
	raw := wikidot.NormalizeDisplayName(r.PostFormValue("display_name"))
	if wikidot.ValidateDisplayName(raw) != wikidot.DisplayNameOK {
		return "name-invalid", nil
	}
	name := wikidot.CanonicalizeUsername(raw)
	if name == "" || wikidot.ReservedUsername(name) {
		return "name-invalid", nil
	}
	if name != lower(user.Username) {
		taken, err := h.deps.DB.UsernameTaken(ctx, name)
		if err != nil {
			return "", err
		}
		if taken {
			return "name-taken", nil
		}
	}
	display := ""
	if raw != name {
		display = raw
	}
	return "name-changed", h.deps.DB.SetUsername(ctx, user.ID, name, display, time.Now())
}

func (h *SettingsHandler) knowsPassword(ctx context.Context, user *db.User, given string) (bool, error) {
	if given == "" {
		return false, nil
	}
	_, stored, err := h.deps.DB.UserForSession(ctx, user.ID)
	if err != nil {
		return false, err
	}
	if !password.IsUsable(stored) {
		return false, nil
	}
	ok, err := password.Verify(given, stored)
	if err != nil {
		return false, nil
	}
	return ok, nil
}

func aboutUser(user *db.User) password.Attributes {
	return password.Attributes{Username: user.Username, DisplayName: user.DisplayName}
}

func plausibleEmail(value string) bool {
	name, domain, found := strings.Cut(value, "@")
	if !found || name == "" || len(value) > 254 {
		return false
	}
	return strings.Contains(domain, ".") && !strings.ContainsAny(value, " \t\r\n")
}
