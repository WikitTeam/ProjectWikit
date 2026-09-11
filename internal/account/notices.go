package account

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/token"
)

func (d Deps) link(r *http.Request, purpose string, userID int64, value token.Value) string {
	host := ""
	if current := site.FromContext(r.Context()); current != nil {
		host = d.origin(r, current)
	}
	uid := base64.RawURLEncoding.EncodeToString([]byte(id(userID)))
	return host + EmailPrefix + purpose + "/" + uid + "/" + d.Tokens.Make(value, time.Now())
}

func (d Deps) origin(r *http.Request, current *db.Site) string {
	return d.Trust.Scheme(r) + "://" + current.Domain
}

func (d Deps) siteTitle(ctx context.Context) string {
	if current := site.FromContext(ctx); current != nil {
		return current.Title
	}
	return ""
}

func (d Deps) sendVerification(r *http.Request, loc *i18n.Localizer, user *db.User, state db.AccountEmail) error {
	if state.Email == "" {
		return nil
	}
	url := d.link(r, purposeVerify, user.ID, verifyValue(user.ID, state))
	return d.Mail.Send(r.Context(), []string{state.Email},
		loc.T("email.verify-subject", "site", d.siteTitle(r.Context())),
		loc.T("email.verify-body", "name", user.DisplayLabel(), "link", url, "site", d.siteTitle(r.Context())))
}

func (d Deps) sendApproval(r *http.Request, loc *i18n.Localizer, user *db.User, to, pending string) error {
	url := d.link(r, purposeApprove, user.ID, token.Custom(purposeApprove, id(user.ID), lower(pending)))
	return d.Mail.Send(r.Context(), []string{to},
		loc.T("email.approve-subject", "site", d.siteTitle(r.Context())),
		loc.T("email.approve-body", "name", user.DisplayLabel(), "email", pending, "link", url, "site", d.siteTitle(r.Context())))
}

func (d Deps) sendActivation(r *http.Request, loc *i18n.Localizer, user *db.User, pending string) error {
	url := d.link(r, purposeActivate, user.ID, token.Custom(purposeActivate, id(user.ID), lower(pending)))
	return d.Mail.Send(r.Context(), []string{pending},
		loc.T("email.activate-subject", "site", d.siteTitle(r.Context())),
		loc.T("email.activate-body", "name", user.DisplayLabel(), "link", url, "site", d.siteTitle(r.Context())))
}

func (d Deps) warnPrevious(r *http.Request, loc *i18n.Localizer, user *db.User, previous, now string) error {
	url := d.link(r, purposeRevert, user.ID, token.Custom(purposeRevert, id(user.ID), lower(now), lower(previous)))
	return d.Mail.Send(r.Context(), []string{previous},
		loc.T("email.moved-subject", "site", d.siteTitle(r.Context())),
		loc.T("email.moved-body", "name", user.DisplayLabel(), "email", masked(now), "link", url, "site", d.siteTitle(r.Context())))
}

func (d Deps) warnPasswordChanged(r *http.Request, loc *i18n.Localizer, user *db.User, to string) error {
	if to == "" {
		return nil
	}
	return d.Mail.Send(r.Context(), []string{to},
		loc.T("email.password-subject", "site", d.siteTitle(r.Context())),
		loc.T("email.password-body", "name", user.DisplayLabel(), "site", d.siteTitle(r.Context()), "link", d.home(r)+ResetPath))
}

func (d Deps) home(r *http.Request) string {
	if current := site.FromContext(r.Context()); current != nil {
		return d.origin(r, current)
	}
	return ""
}

func masked(email string) string {
	name, domain, found := strings.Cut(email, "@")
	if !found || name == "" {
		return email
	}
	return name[:1] + "***@" + domain
}

func (d Deps) retirePassword(ctx context.Context, userID int64) error {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return err
	}
	return d.DB.SetPassword(ctx, userID, "!"+hex.EncodeToString(buf))
}
