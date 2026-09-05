package account

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/password"
	"github.com/WikitTeam/ProjectWikit/internal/shell"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
)

const (
	SignupPath     = "/-/signup"
	CheckPath      = "/-/signup/check-wikidot"
	SendCodePath   = "/-/signup/send-wikidot-code"
	SignupPrefix   = SignupPath + "/"
	defaultRoleRef = "reader"
)

type SignupHandler struct {
	deps Deps
}

var _ http.Handler = (*SignupHandler)(nil)

func NewSignup(d Deps) *SignupHandler { return &SignupHandler{deps: d} }

func (h *SignupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case CheckPath:
		h.check(w, r)
		return
	case SendCodePath:
		h.sendCode(w, r)
		return
	case SignupPath:
	default:
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
	token := csrf.Issue(w, r)

	form := shell.Signup{
		AuthIcon: authIcon(current), SiteTitle: current.Title,
		Notice: current.SignupNotice, CSRF: token,
	}
	if r.Method == http.MethodPost {
		if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		done, err := h.register(w, r, loc, current, &form)
		if err != nil {
			h.deps.logger().Error("sign up", "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if done {
			redirect(w, homePath, http.StatusFound)
			return
		}
	}

	render := shell.New(loc, h.deps.Assets, h.deps.TimeZone)
	content, err := render.Signup(form)
	if err == nil {
		var body string
		body, err = h.deps.page(r, loc, current, loc.T("signup.title"), content)
		if err == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			if r.Method != http.MethodHead {
				_, _ = w.Write([]byte(body))
			}
			return
		}
	}
	h.deps.logger().Error("render signup", "err", err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (h *SignupHandler) register(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer,
	current *db.Site, form *shell.Signup) (bool, error) {

	ctx := r.Context()
	raw := wikidot.NormalizeDisplayName(r.PostFormValue("username"))
	form.Username = raw
	plain := r.PostFormValue("password")

	if problem := wikidot.ValidateDisplayName(raw); problem != wikidot.DisplayNameOK {
		form.Error = displayNameError(loc, problem)
		return false, nil
	}
	name := wikidot.CanonicalizeUsername(raw)
	display := ""
	if raw != name {
		display = raw
	}
	if wikidot.ReservedUsername(name) {
		form.Error = loc.T("signup.error-name-reserved")
		return false, nil
	}

	claiming, err := h.deps.DB.UserByWikidotName(ctx, name)
	if errors.Is(err, db.ErrNotFound) {
		claiming, err = nil, nil
	}
	if err != nil {
		return false, err
	}
	form.IsWikidot = claiming != nil

	if plain != r.PostFormValue("password_confirm") {
		form.Error = loc.T("signup.error-password-mismatch")
		return false, nil
	}
	about := password.Attributes{Username: name, DisplayName: display}
	if claiming != nil {
		about = password.Attributes{Username: claiming.Username, DisplayName: claiming.DisplayName}
	}
	if err := password.Validate(plain, about); err != nil {
		form.Error = passwordError(loc, err)
		return false, nil
	}
	hash, err := password.Hash(plain)
	if err != nil {
		return false, err
	}

	if claiming != nil {
		return h.claim(w, r, loc, current, form, claiming, raw, name, hash)
	}

	taken, err := h.deps.DB.UsernameTaken(ctx, name)
	if err != nil {
		return false, err
	}
	if taken {
		form.Error = loc.T("signup.error-name-taken", "name", name)
		return false, nil
	}

	id, err := h.deps.DB.CreateUser(ctx, name, display, hash, true, time.Now())
	if err != nil {
		return false, err
	}
	if name == "" {
		if err := h.fallbackName(ctx, id, display); err != nil {
			return false, err
		}
	}
	if err := h.grant(ctx, current.DefaultRoleID, id); err != nil {
		return false, err
	}
	if err := h.deps.welcome(ctx, id); err != nil {
		return false, err
	}
	user, err := h.deps.DB.UserByID(ctx, id)
	if err != nil {
		return false, err
	}
	return true, h.deps.signIn(ctx, w, r, user)
}

func (h *SignupHandler) claim(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer,
	current *db.Site, form *shell.Signup, claiming *db.User, raw, name, hash string) (bool, error) {

	ctx := r.Context()
	code := strings.TrimSpace(r.PostFormValue("verification_code"))
	if code == "" {
		form.Error = loc.T("signup.error-code-empty")
		return false, nil
	}
	message, err := h.deps.Verifier.Verify(ctx, raw, code)
	if err != nil {
		form.Error = verifyError(loc, message, err)
		return false, nil
	}

	display := (*string)(nil)
	if claiming.DisplayName == "" && raw != name {
		display = &raw
	}
	if err := h.deps.DB.ActivateUser(ctx, claiming.ID, name, display, hash); err != nil {
		return false, err
	}
	if err := h.grant(ctx, current.VerifiedRoleID, claiming.ID); err != nil {
		return false, err
	}
	if err := h.deps.welcome(ctx, claiming.ID); err != nil {
		return false, err
	}
	user, err := h.deps.DB.UserByID(ctx, claiming.ID)
	if err != nil {
		return false, err
	}
	return true, h.deps.signIn(ctx, w, r, user)
}

func (h *SignupHandler) grant(ctx context.Context, configured *int64, userID int64) error {
	if configured != nil {
		return h.deps.DB.GrantRole(ctx, userID, *configured)
	}
	id, err := h.deps.DB.RoleIDBySlug(ctx, defaultRoleRef)
	if errors.Is(err, db.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return h.deps.DB.GrantRole(ctx, userID, id)
}

const fallbackAttempts = 1000

func (h *SignupHandler) fallbackName(ctx context.Context, id int64, display string) error {
	for suffix := 1; suffix < fallbackAttempts; suffix++ {
		candidate := wikidot.FallbackUsername(id, suffix)
		taken, err := h.deps.DB.UsernameTaken(ctx, candidate)
		if err != nil {
			return err
		}
		if !taken {
			return h.deps.DB.RenameUser(ctx, id, candidate, display)
		}
	}
	return errors.New("account: no fallback username is free")
}

func writeJSON(w http.ResponseWriter, status int, body map[string]any) {
	raw, err := json.Marshal(body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}

func displayNameError(loc *i18n.Localizer, problem wikidot.DisplayNameProblem) string {
	switch problem {
	case wikidot.DisplayNameEmpty:
		return loc.T("signup.error-name-empty")
	case wikidot.DisplayNameTooLong:
		return loc.T("signup.error-name-too-long")
	case wikidot.DisplayNameOddSpace:
		return loc.T("signup.error-name-space")
	case wikidot.DisplayNameLeadingMark:
		return loc.T("signup.error-name-mark")
	default:
		return loc.T("signup.error-name-invisible")
	}
}

func passwordError(loc *i18n.Localizer, err error) string {
	switch {
	case errors.Is(err, password.ErrTooShort):
		return loc.T("password.too-short")
	case errors.Is(err, password.ErrTooCommon):
		return loc.T("password.too-common")
	case errors.Is(err, password.ErrAllNumeric):
		return loc.T("password.all-numeric")
	default:
		return loc.T("password.too-similar")
	}
}

func (h *SignupHandler) check(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	name := wikidot.CanonicalizeUsername(strings.TrimSpace(r.URL.Query().Get("username")))
	found := false
	if name != "" {
		_, err := h.deps.DB.UserByWikidotName(r.Context(), name)
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			h.deps.logger().Error("check wikidot name", "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		found = err == nil
	}
	writeJSON(w, http.StatusOK, map[string]any{"is_wikidot": found})
}

func (h *SignupHandler) sendCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	current := site.FromContext(ctx)
	if current == nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	loc := h.deps.Bundle.Localizer(i18n.DefaultLanguage)
	if err := r.ParseForm(); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	raw := strings.TrimSpace(r.PostFormValue("username"))
	name := wikidot.CanonicalizeUsername(raw)
	if name == "" {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": loc.T("signup.error-username-empty")})
		return
	}
	_, err := h.deps.DB.UserByWikidotName(ctx, name)
	if errors.Is(err, db.ErrNotFound) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": loc.T("signup.error-not-wikidot")})
		return
	}
	if err != nil {
		h.deps.logger().Error("check wikidot name", "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	message, err := h.deps.Verifier.Send(ctx, raw)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": verifyError(loc, message, err)})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func verifyError(loc *i18n.Localizer, message string, err error) string {
	switch {
	case errors.Is(err, ErrVerifierUnreachable):
		return loc.T("signup.error-verify-unreachable")
	case message != "":
		return message
	default:
		return loc.T("signup.error-code-wrong")
	}
}
