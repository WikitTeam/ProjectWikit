package account

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/password"
	"github.com/WikitTeam/ProjectWikit/internal/session"
	"github.com/WikitTeam/ProjectWikit/internal/shell"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
)

const modelBackend = "django.contrib.auth.backends.ModelBackend"

type LoginHandler struct {
	deps Deps
}

var _ http.Handler = (*LoginHandler)(nil)

func NewLogin(d Deps) *LoginHandler { return &LoginHandler{deps: d} }

func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != LoginPath {
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
		redirect(w, destination(r), http.StatusFound)
		return
	}
	loc := h.deps.Bundle.Localizer(i18n.DefaultLanguage)
	token := csrf.Issue(w, r)

	form := shell.Login{AuthIcon: authIcon(current), SiteTitle: current.Title, CSRF: token}
	if r.Method == http.MethodPost {
		if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		form.Username = r.PostFormValue("username")

		user, err := h.authenticate(ctx, form.Username, r.PostFormValue("password"))
		if err != nil {
			h.deps.logger().Error("authenticate", "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if user == nil {
			form.Error = loc.T("login.error-credentials")
		} else {
			if err := h.deps.signIn(ctx, w, r, user); err != nil {
				h.deps.logger().Error("sign in", "user", user.ID, "err", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			redirect(w, destination(r), http.StatusFound)
			return
		}
	}

	render := shell.New(loc, h.deps.Assets, h.deps.TimeZone)
	content, err := render.Login(form)
	if err == nil {
		var body string
		body, err = h.deps.page(r, loc, current, loc.T("login.title"), content)
		if err == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			if r.Method != http.MethodHead {
				_, _ = w.Write([]byte(body))
			}
			return
		}
	}
	h.deps.logger().Error("render login", "err", err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (h *LoginHandler) authenticate(ctx context.Context, name, plain string) (*db.User, error) {
	user, hash, err := h.lookup(ctx, name)
	if err != nil {
		return nil, err
	}
	if user == nil {
		canonical := wikidot.CanonicalizeUsername(name)
		if canonical == "" || canonical == name {
			return nil, nil
		}
		if user, hash, err = h.lookup(ctx, canonical); err != nil || user == nil {
			return nil, err
		}
	}

	ok, err := password.Verify(plain, hash)
	if err != nil || !ok {
		return nil, nil
	}
	if !user.IsActive {
		return nil, nil
	}
	if password.NeedsRehash(hash) {
		fresh, err := password.Hash(plain)
		if err != nil {
			return nil, err
		}
		if err := h.deps.DB.SetPassword(ctx, user.ID, fresh); err != nil {
			return nil, err
		}
	}
	return user, nil
}

func (h *LoginHandler) lookup(ctx context.Context, name string) (*db.User, string, error) {
	user, hash, err := h.deps.DB.UserForLogin(ctx, name)
	if errors.Is(err, db.ErrNotFound) {
		return nil, "", nil
	}
	return user, hash, err
}

func (d Deps) signIn(ctx context.Context, w http.ResponseWriter, r *http.Request, user *db.User) error {
	if old, err := r.Cookie(session.CookieName); err == nil && old.Value != "" {
		if err := d.DB.DeleteSession(ctx, old.Value); err != nil {
			return err
		}
	}
	_, hash, err := d.DB.UserForSession(ctx, user.ID)
	if err != nil {
		return err
	}
	key, err := session.NewKey()
	if err != nil {
		return err
	}
	data, err := d.Sessions.Encode(map[string]any{
		session.AuthUserID:      strconv.FormatInt(user.ID, 10),
		session.AuthUserBackend: modelBackend,
		session.AuthUserHash:    d.Sessions.AuthHash(hash),
	})
	if err != nil {
		return err
	}
	now := time.Now()
	if err := d.DB.SaveSession(ctx, key, data, now.Add(session.CookieAge)); err != nil {
		return err
	}
	if err := d.DB.SetLastLogin(ctx, user.ID, now); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name: session.CookieName, Value: key, Path: "/",
		Expires: now.Add(session.CookieAge), MaxAge: int(session.CookieAge / time.Second),
		HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteLaxMode,
	})
	return nil
}

type LogoutHandler struct {
	deps Deps
}

var _ http.Handler = (*LogoutHandler)(nil)

func NewLogout(d Deps) *LogoutHandler { return &LogoutHandler{deps: d} }

func (h *LogoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != LogoutPath {
		notFound(w)
		return
	}
	if cookie, err := r.Cookie(session.CookieName); err == nil && cookie.Value != "" {
		if err := h.deps.DB.DeleteSession(r.Context(), cookie.Value); err != nil {
			h.deps.logger().Error("sign out", "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name: session.CookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteLaxMode,
	})
	redirect(w, destination(r), http.StatusFound)
}

const notFoundBody = "Not found"

func notFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(notFoundBody)))
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(notFoundBody))
}
