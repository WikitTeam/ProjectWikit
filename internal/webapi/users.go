package webapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/wikidot"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

const (
	UsersPath   = "/pw-api/users"
	UsersPrefix = "/pw-api/users/"
)

type Users struct {
	deps     Deps
	upstream http.Handler
}

var _ http.Handler = (*Users)(nil)

func NewUsers(d Deps, upstream http.Handler) *Users {
	return &Users{deps: d, upstream: upstream}
}

func (h *Users) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	loc := h.deps.Bundle.Localizer(i18n.DefaultLanguage)
	if r.URL.Path == UsersPath && r.Method == http.MethodGet {
		h.all(w, r, loc)
		return
	}
	rest, ok := strings.CutPrefix(r.URL.Path, UsersPrefix)
	if !ok {
		h.upstream.ServeHTTP(w, r)
		return
	}
	head, tail, _ := strings.Cut(rest, "/")
	switch {
	case head == "lookup" && tail == "" && r.Method == http.MethodGet:
		h.lookup(w, r, loc)
	case head == "generate-invite" && tail == "" && r.Method == http.MethodPost:
		h.invite(w, r, loc)
	case tail == "block" && (r.Method == http.MethodPost || r.Method == http.MethodDelete):
		h.block(w, r, loc, head)
	default:
		h.upstream.ServeHTTP(w, r)
	}
}

func (h *Users) all(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	ctx := r.Context()
	users, err := h.deps.DB.AllUsers(ctx)
	if err != nil {
		h.deps.log().Error("list users", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	rendered, err := repo.UsersJSON(ctx, h.deps.DB, users, time.Now())
	if err != nil {
		h.deps.log().Error("render users", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	body, err := wikijson.Marshal(rendered)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func (h *Users) lookup(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	ctx := r.Context()
	name := strings.TrimSpace(r.URL.Query().Get("username"))
	if name == "" {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-missing-username")))
		return
	}
	found, err := h.deps.DB.UserByAnyName(ctx, wikidot.CanonicalizeUsername(name), name)
	if errors.Is(err, db.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, field("error", loc.T("api-user-not-found")))
		return
	}
	if err != nil {
		h.deps.log().Error("look up user", "name", name, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	rendered, err := repo.UserJSON(ctx, h.deps.DB, found)
	if err != nil {
		h.deps.log().Error("render user", "id", found.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	body, err := wikijson.Marshal(rendered)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func (h *Users) block(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer, raw string) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, field("error", loc.T("api-login-required")))
		return
	}
	current := site.FromContext(ctx)
	if current == nil {
		h.deps.log().Error("block", "err", errors.New("the request carries no site"))
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
		writeJSON(w, http.StatusForbidden, field("error", loc.T("api-csrf-failed")))
		return
	}

	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusNotFound, field("error", loc.T("api-user-not-found")))
		return
	}
	target, err := h.deps.DB.UserByID(ctx, id)
	if errors.Is(err, db.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, field("error", loc.T("api-user-not-found")))
		return
	}
	if err != nil {
		h.deps.log().Error("read user", "id", id, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}

	blocked := r.Method == http.MethodPost
	if blocked {
		if target.ID == user.ID {
			writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-cannot-block-self")))
			return
		}
		_, err = h.deps.DB.BlockUser(ctx, user.ID, target.ID, time.Now())
	} else {
		_, err = h.deps.DB.UnblockUser(ctx, user.ID, target.ID)
	}
	if err != nil {
		h.deps.log().Error("change block", "user", user.ID, "target", target.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	body, err := wikijson.Marshal(wikijson.Object{
		{Key: "status", Value: "ok"},
		{Key: "blocked", Value: blocked},
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}
