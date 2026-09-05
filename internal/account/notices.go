package account

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/token"
)

func (d Deps) link(ctx context.Context, purpose string, userID int64, value token.Value) string {
	current := site.FromContext(ctx)
	host := ""
	if current != nil {
		host = "https://" + current.Domain
	}
	uid := base64.RawURLEncoding.EncodeToString([]byte(id(userID)))
	return host + EmailPrefix + purpose + "/" + uid + "/" + d.Tokens.Make(value, time.Now())
}

func (d Deps) siteTitle(ctx context.Context) string {
	if current := site.FromContext(ctx); current != nil {
		return current.Title
	}
	return ""
}

func (d Deps) sendVerification(ctx context.Context, loc *i18n.Localizer, user *db.User, state db.AccountEmail) error {
	if state.Email == "" {
		return nil
	}
	url := d.link(ctx, purposeVerify, user.ID, verifyValue(user.ID, state))
	return d.Mail.Send(ctx, []string{state.Email},
		loc.T("email.verify-subject", "site", d.siteTitle(ctx)),
		loc.T("email.verify-body", "name", user.DisplayLabel(), "link", url, "site", d.siteTitle(ctx)))
}

func (d Deps) sendApproval(ctx context.Context, loc *i18n.Localizer, user *db.User, to, pending string) error {
	url := d.link(ctx, purposeApprove, user.ID, token.Custom(purposeApprove, id(user.ID), lower(pending)))
	return d.Mail.Send(ctx, []string{to},
		loc.T("email.approve-subject", "site", d.siteTitle(ctx)),
		loc.T("email.approve-body", "name", user.DisplayLabel(), "email", pending, "link", url, "site", d.siteTitle(ctx)))
}

func (d Deps) sendActivation(ctx context.Context, loc *i18n.Localizer, user *db.User, pending string) error {
	url := d.link(ctx, purposeActivate, user.ID, token.Custom(purposeActivate, id(user.ID), lower(pending)))
	return d.Mail.Send(ctx, []string{pending},
		loc.T("email.activate-subject", "site", d.siteTitle(ctx)),
		loc.T("email.activate-body", "name", user.DisplayLabel(), "link", url, "site", d.siteTitle(ctx)))
}

func (d Deps) warnPrevious(ctx context.Context, loc *i18n.Localizer, user *db.User, previous, now string) error {
	url := d.link(ctx, purposeRevert, user.ID, token.Custom(purposeRevert, id(user.ID), lower(now), lower(previous)))
	return d.Mail.Send(ctx, []string{previous},
		loc.T("email.moved-subject", "site", d.siteTitle(ctx)),
		loc.T("email.moved-body", "name", user.DisplayLabel(), "email", masked(now), "link", url, "site", d.siteTitle(ctx)))
}

func (d Deps) warnPasswordChanged(ctx context.Context, loc *i18n.Localizer, user *db.User, to string) error {
	if to == "" {
		return nil
	}
	return d.Mail.Send(ctx, []string{to},
		loc.T("email.password-subject", "site", d.siteTitle(ctx)),
		loc.T("email.password-body", "name", user.DisplayLabel(), "site", d.siteTitle(ctx), "link", "https://"+d.host(ctx)+ResetPath))
}

func (d Deps) host(ctx context.Context) string {
	if current := site.FromContext(ctx); current != nil {
		return current.Domain
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
