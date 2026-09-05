package account

import (
	"context"
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
	"github.com/WikitTeam/ProjectWikit/internal/page"
	"github.com/WikitTeam/ProjectWikit/internal/pagerender"
	"github.com/WikitTeam/ProjectWikit/internal/password"
	"github.com/WikitTeam/ProjectWikit/internal/renderer"
	"github.com/WikitTeam/ProjectWikit/internal/shell"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/token"
)

const (
	ResetPath         = "/-/password_reset"
	ResetDonePath     = "/-/password_reset/done"
	ResetConfirmPath  = "/-/reset/"
	ResetCompletePath = "/-/reset/done"
	ResetHelpPath     = "/-/password_reset/help"
	ResetPrefix       = ResetPath + "/"
)

const (
	stageAsk  = "ask"
	stageSent = "sent"
	stageSet  = "set"
	stageDead = "dead"
	stageHelp = "help"
	stageDone = "done"
)

type ResetHandler struct {
	deps Deps
}

var _ http.Handler = (*ResetHandler)(nil)

func NewReset(d Deps) *ResetHandler { return &ResetHandler{deps: d} }

func (h *ResetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	loc := h.deps.Bundle.Localizer(i18n.DefaultLanguage)
	form := shell.Reset{
		AuthIcon: authIcon(current), SiteTitle: current.Title, CSRF: csrf.Issue(w, r),
	}
	title := ""

	switch {
	case r.URL.Path == ResetHelpPath:
		form.Stage, title = stageHelp, loc.T("reset.help-title")
		help, err := h.help(r, current)
		if err != nil {
			h.deps.logger().Error("render password help", "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		form.Help = help
	case r.URL.Path == ResetPath:
		form.Stage, title = stageAsk, loc.T("reset.title")
		if r.Method == http.MethodPost {
			if !h.checkToken(w, r, current) {
				return
			}
			if err := h.request(r, loc, current); err != nil {
				h.deps.logger().Error("request password reset", "err", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			redirect(w, ResetDonePath, http.StatusFound)
			return
		}
	case r.URL.Path == ResetDonePath:
		form.Stage, title = stageSent, loc.T("reset.sent-title")
	case r.URL.Path == ResetCompletePath:
		form.Stage, title = stageDone, loc.T("reset.done-title")
	case strings.HasPrefix(r.URL.Path, ResetConfirmPath):
		done, err := h.confirm(w, r, loc, current, &form)
		if err != nil {
			h.deps.logger().Error("finish password reset", "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if done {
			return
		}
		title = loc.T("reset.set-title")
		if form.Stage == stageDead {
			title = loc.T("reset.dead-title")
		}
	default:
		notFound(w)
		return
	}

	render := shell.New(loc, h.deps.Assets, h.deps.TimeZone)
	content, err := render.Reset(form)
	if err == nil {
		var body string
		body, err = h.deps.page(r, loc, current, title, content)
		if err == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			if r.Method != http.MethodHead {
				_, _ = w.Write([]byte(body))
			}
			return
		}
	}
	h.deps.logger().Error("render password reset", "err", err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

// The text a site writes here is wikitext, but it is rendered in the mode that
// turns modules off, so a setting cannot reach into the module set.
func (h *ResetHandler) help(r *http.Request, current *db.Site) (string, error) {
	source := strings.TrimSpace(current.PasswordHelp)
	if source == "" {
		return "", nil
	}
	ctx := r.Context()
	loc := h.deps.Bundle.Localizer(i18n.DefaultLanguage)
	env := pagerender.Deps{DB: h.deps.DB, Engine: h.deps.Engine, Icons: h.deps.Icons}.
		Env(ctx, loc, current, auth.FromContext(ctx))

	info, err := env.PageInfo(nil)
	if err != nil {
		return "", err
	}
	pc := page.NewContext(nil, nil, nil, auth.FromContext(ctx))
	html, err := env.HTML(source, info, env.Callbacks(nil, pc), renderer.ModeSystem)
	if err != nil {
		return "", err
	}
	return html.Body, nil
}

func (h *ResetHandler) checkToken(w http.ResponseWriter, r *http.Request, current *db.Site) bool {
	if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return false
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return false
	}
	return true
}

func (h *ResetHandler) request(r *http.Request, loc *i18n.Localizer, current *db.Site) error {
	ctx := r.Context()
	email := strings.TrimSpace(r.PostFormValue("email"))
	if email == "" {
		return nil
	}
	user, err := h.deps.DB.UserByEmail(ctx, email)
	if errors.Is(err, db.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if !user.IsActive {
		return nil
	}
	value, err := h.resetValue(ctx, user)
	if err != nil {
		return err
	}
	minted := h.deps.Tokens.Make(value, time.Now())
	uid := base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(user.ID, 10)))
	home := siteURL(r, current)
	link := home + ResetConfirmPath + uid + "/" + minted

	subject := loc.T("reset.mail-subject", "site", current.Title)
	body := loc.T("reset.mail-body", "link", link, "home", home, "site", current.Title)
	return h.deps.Mail.Send(ctx, []string{email}, subject, body)
}

func (h *ResetHandler) confirm(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer,
	current *db.Site, form *shell.Reset) (bool, error) {

	ctx := r.Context()
	rest := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, ResetConfirmPath), "/")
	uid, secret, found := strings.Cut(rest, "/")
	if !found || strings.Contains(secret, "/") {
		form.Stage = stageDead
		return false, nil
	}
	user, value, err := h.resetTarget(r, uid)
	if err != nil {
		return false, err
	}
	if user == nil || !h.deps.Tokens.Check(secret, value, time.Now()) {
		form.Stage = stageDead
		return false, nil
	}
	form.Stage = stageSet

	if r.Method != http.MethodPost {
		return false, nil
	}
	if !h.checkToken(w, r, current) {
		return true, nil
	}
	first := r.PostFormValue("new_password1")
	if first != r.PostFormValue("new_password2") {
		form.Error = loc.T("reset.error-mismatch")
		return false, nil
	}
	about := password.Attributes{Username: user.Username, DisplayName: user.DisplayName}
	if err := password.Validate(first, about); err != nil {
		form.Error = passwordError(loc, err)
		return false, nil
	}
	hash, err := password.Hash(first)
	if err != nil {
		return false, err
	}
	if err := h.deps.DB.SetPassword(ctx, user.ID, hash); err != nil {
		return false, err
	}
	redirect(w, ResetCompletePath, http.StatusFound)
	return true, nil
}

func (h *ResetHandler) resetTarget(r *http.Request, uid string) (*db.User, token.Value, error) {
	raw, err := base64.RawURLEncoding.DecodeString(uid)
	if err != nil {
		return nil, nil, nil
	}
	id, err := strconv.ParseInt(string(raw), 10, 64)
	if err != nil {
		return nil, nil, nil
	}
	user, err := h.deps.DB.UserByID(r.Context(), id)
	if errors.Is(err, db.ErrNotFound) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	value, err := h.resetValue(r.Context(), user)
	if err != nil {
		return nil, nil, err
	}
	return user, value, nil
}

func (h *ResetHandler) resetValue(ctx context.Context, user *db.User) (token.Value, error) {
	hash, last, email, err := h.deps.DB.ResetFields(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return token.ResetValue(user.ID, hash, last, email), nil
}

func siteURL(r *http.Request, current *db.Site) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + current.Domain
}
