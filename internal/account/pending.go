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
	"github.com/WikitTeam/ProjectWikit/internal/shell"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/token"
)

const (
	SignupPendingPath = SignupPath + "/pending"

	pendingCookie  = "pwikit_signup"
	purposePending = "pending"
)

// The tab that signed up is the only one that knew the password, so only its
// cookie may sign the account in once the mailbox has answered.
func (d Deps) awaitVerification(w http.ResponseWriter, r *http.Request, userID int64, email string) {
	value := token.Custom(purposePending, id(userID), lower(email))
	d.setCookie(w, r, pendingCookie, encodeID(userID)+"/"+d.Tokens.Make(value, time.Now()), token.DefaultTimeout)
}

func (h *SignupHandler) awaiting(r *http.Request) (*db.User, db.AccountEmail, error) {
	cookie, err := r.Cookie(pendingCookie)
	if err != nil {
		return nil, db.AccountEmail{}, nil
	}
	uid, secret, _ := strings.Cut(cookie.Value, "/")
	user, state, err := h.deps.userByUID(r.Context(), uid)
	if err != nil || user == nil {
		return nil, db.AccountEmail{}, err
	}
	if !h.deps.Tokens.Check(secret, token.Custom(purposePending, id(user.ID), lower(state.Email)), time.Now()) {
		return nil, db.AccountEmail{}, nil
	}
	return user, state, nil
}

func (h *SignupHandler) pending(w http.ResponseWriter, r *http.Request) {
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
	user, state, err := h.awaiting(r)
	if err != nil {
		h.deps.logger().Error("read pending sign-up", "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if user == nil {
		h.deps.dropCookie(w, r, pendingCookie)
		redirect(w, SignupPath, http.StatusFound)
		return
	}
	verified := state.VerifiedAt != nil

	if r.Method == http.MethodPost {
		if err := csrf.Verify(r, []string{current.Domain}); err != nil {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		if !verified {
			redirect(w, SignupPendingPath, http.StatusSeeOther)
			return
		}
		if _, err := h.deps.carryOn(w, r, user); err != nil {
			h.deps.logger().Error("sign in after verification", "user", user.ID, "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		h.deps.dropCookie(w, r, pendingCookie)
		redirect(w, homePath, http.StatusSeeOther)
		return
	}

	loc := h.deps.Bundle.For(ctx)
	n := shell.Notice{
		Heading:  loc.T("signup.pending-title"),
		Body:     loc.T("signup.pending", "email", state.Email),
		LinkURL:  LoginPath,
		LinkText: loc.T("reset.back-login"),
	}
	if verified {
		n = shell.Notice{
			Heading:    loc.T("signup.pending-done-title"),
			Body:       loc.T("signup.pending-done"),
			ActionURL:  SignupPendingPath,
			ActionText: loc.T("account.continue"),
			CSRF:       csrf.Issue(w, r),
		}
	}
	h.deps.notice(w, r, loc, current, http.StatusOK, n)
}

// A visitor already signed in as someone else keeps that session, so a link
// opened in a shared browser cannot swap the account under them.
func (d Deps) carryOn(w http.ResponseWriter, r *http.Request, user *db.User) (bool, error) {
	viewer := auth.FromContext(r.Context())
	if viewer != nil || !user.ActiveAt(time.Now()) {
		return false, nil
	}
	return true, d.signIn(r.Context(), w, r, user)
}

func (d Deps) notice(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, current *db.Site, status int, n shell.Notice) {
	n.AuthIcon, n.SiteTitle = authIcon(current), current.Title
	content, err := shell.New(loc, d.Assets).Notice(n)
	if err == nil {
		var body string
		body, err = d.page(r, loc, current, n.Heading, content)
		if err == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(status)
			if r.Method != http.MethodHead {
				_, _ = w.Write([]byte(body))
			}
			return
		}
	}
	d.logger().Error("render notice", "err", err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (d Deps) setCookie(w http.ResponseWriter, r *http.Request, name, value string, life time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name: name, Value: value, Path: "/", MaxAge: int(life / time.Second),
		HttpOnly: true, Secure: d.secure(r), SameSite: http.SameSiteLaxMode,
	})
}

func (d Deps) dropCookie(w http.ResponseWriter, r *http.Request, name string) {
	if _, err := r.Cookie(name); err != nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: name, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: d.secure(r), SameSite: http.SameSiteLaxMode,
	})
}

func (d Deps) secure(r *http.Request) bool {
	return d.Trust != nil && d.Trust.Scheme(r) == "https"
}

func encodeID(n int64) string { return base64.RawURLEncoding.EncodeToString([]byte(id(n))) }

func (d Deps) userByUID(ctx context.Context, uid string) (*db.User, db.AccountEmail, error) {
	raw, err := base64.RawURLEncoding.DecodeString(uid)
	if err != nil {
		return nil, db.AccountEmail{}, nil
	}
	parsed, err := strconv.ParseInt(string(raw), 10, 64)
	if err != nil {
		return nil, db.AccountEmail{}, nil
	}
	user, err := d.DB.UserByID(ctx, parsed)
	if errors.Is(err, db.ErrNotFound) {
		return nil, db.AccountEmail{}, nil
	}
	if err != nil {
		return nil, db.AccountEmail{}, err
	}
	state, err := d.DB.AccountEmail(ctx, user.ID)
	if err != nil {
		return nil, db.AccountEmail{}, err
	}
	return user, state, nil
}
