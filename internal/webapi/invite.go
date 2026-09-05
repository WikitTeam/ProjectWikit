package webapi

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/auth"
	"github.com/WikitTeam/ProjectWikit/internal/csrf"
	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/perms"
	"github.com/WikitTeam/ProjectWikit/internal/repo"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/token"
	"github.com/WikitTeam/ProjectWikit/internal/wikijson"
)

const acceptPrefix = "/-/accept/"

const (
	inviteKindRegister = "register"
	inviteByLink       = "link"
)

type inviteRequest struct {
	Email string  `json:"email"`
	Roles []int64 `json:"roles"`
}

func (h *Users) invite(w http.ResponseWriter, r *http.Request, loc *i18n.Localizer) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, field("error", loc.T("api-login-required")))
		return
	}
	current := site.FromContext(ctx)
	if current == nil {
		h.deps.log().Error("invite", "err", errors.New("the request carries no site"))
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	if err := csrf.Verify(r, []string{current.Domain, current.MediaDomain}); err != nil {
		writeJSON(w, http.StatusForbidden, field("error", loc.T("api-csrf-failed")))
		return
	}
	subject, err := repo.NewPerms(ctx, h.deps.DB).Subject(user, time.Now())
	if err != nil {
		h.deps.log().Error("resolve permissions", "user", user.ID, "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	if !perms.Resolve(subject, nil).Has(perms.ManageUsers) {
		writeJSON(w, http.StatusForbidden, field("error", loc.T("api-forbidden")))
		return
	}

	raw, err := readBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-request")))
		return
	}
	var input inviteRequest
	if json.Unmarshal(raw, &input) != nil {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-bad-json")))
		return
	}
	email := strings.TrimSpace(input.Email)
	if email == "" {
		writeJSON(w, http.StatusBadRequest, field("error", loc.T("api-missing-email")))
		return
	}
	if _, err := h.deps.DB.UserByEmail(ctx, email); err == nil {
		writeJSON(w, http.StatusConflict, field("error", loc.T("api-email-taken")))
		return
	} else if !errors.Is(err, db.ErrNotFound) {
		h.deps.log().Error("look up email", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}

	body, err := h.mintInvite(r, current, user, email, input.Roles)
	if err != nil {
		h.deps.log().Error("write invitation", "err", err)
		writeJSON(w, http.StatusInternalServerError, field("error", loc.T("api-internal-error")))
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func (h *Users) mintInvite(r *http.Request, current *db.Site, by *db.User, email string, roles []int64) (string, error) {
	ctx := r.Context()
	now := time.Now()
	id, err := h.deps.DB.CreateInvitedUser(ctx, email, now)
	if err != nil {
		return "", err
	}
	for _, role := range roles {
		if err := h.deps.DB.GrantRole(ctx, id, role); err != nil {
			return "", err
		}
	}
	minted := h.deps.Tokens.Make(token.InviteValue(id, false), now)
	uid := base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(id, 10)))
	link := scheme(r) + "://" + current.Domain + acceptPrefix + uid + "/" + minted

	owner := by.ID
	if _, err := h.deps.DB.CreateInviteLink(ctx, inviteKindRegister, inviteByLink,
		email, "", minted, uid, &owner, id, now); err != nil {
		return "", err
	}
	return wikijson.Marshal(wikijson.Object{
		{Key: "email", Value: email},
		{Key: "invitationUrl", Value: link},
		{Key: "userId", Value: id},
	})
}

func scheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	return "http"
}
