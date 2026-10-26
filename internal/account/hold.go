package account

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/site"
	"github.com/WikitTeam/ProjectWikit/internal/token"
)

// A superuser sets an address for people who lost the old mailbox, which is
// also the story an impostor tells, so the real owner gets a week to notice.
const HoldPeriod = 7 * 24 * time.Hour

const (
	purposeHold   = "hold"
	purposeCancel = "cancel"
)

const (
	HoldRequested = "email-request"
	HoldConfirmed = "email-confirm"
	HoldCancelled = "email-cancel"
	HoldApplied   = "email-apply"
	HoldDropped   = "email-drop"
)

const holdScreen = "users"

var (
	ErrHoldInvalid = errors.New("account: held email is not an address")
	ErrHoldTaken   = errors.New("account: held email belongs to another account")
	ErrHoldSame    = errors.New("account: held email is the current one")
)

func holdDays() int { return int(HoldPeriod / (24 * time.Hour)) }

func (d Deps) holdTokens() token.Generator {
	g := d.Tokens
	g.Timeout = HoldPeriod
	return g
}

func (d Deps) HoldEmail(r *http.Request, loc *i18n.Localizer, target *db.User, email string, by *db.User) error {
	ctx := r.Context()
	wanted := lower(email)
	if !plausibleEmail(wanted) {
		return ErrHoldInvalid
	}
	state, err := d.DB.AccountEmail(ctx, target.ID)
	if err != nil {
		return err
	}
	if wanted == lower(state.Email) {
		return ErrHoldSame
	}
	taken, err := d.DB.VerifiedEmailTaken(ctx, wanted, target.ID)
	if err != nil {
		return err
	}
	if taken {
		return ErrHoldTaken
	}
	current := site.FromContext(ctx)
	if err := d.DB.HoldEmail(ctx, target.ID, wanted, current.ID); err != nil {
		return err
	}
	url := d.link(r, purposeHold, target.ID, token.Custom(purposeHold, id(target.ID), wanted))
	if err := d.Mail.Send(ctx, []string{wanted},
		loc.T("email.hold-subject", "site", current.Title),
		loc.T("email.hold-body", "name", target.DisplayLabel(), "link", url, "site", current.Title, "days", holdDays())); err != nil {
		return err
	}
	d.noteHold(ctx, &current.ID, by, HoldRequested, target.ID, target.Username, state.Email, wanted)
	return nil
}

func (d Deps) CancelHold(ctx context.Context, target *db.User, by *db.User) (bool, error) {
	state, err := d.DB.AccountEmail(ctx, target.ID)
	if err != nil || state.Held == "" {
		return false, err
	}
	done, err := d.DB.DropEmailHold(ctx, target.ID, state.Held)
	if err != nil || !done {
		return false, err
	}
	d.noteHold(ctx, state.HeldSite, by, HoldCancelled, target.ID, target.Username, state.Email, state.Held)
	return true, nil
}

// No background job applies a hold. Every place the address matters settles first.
func (d Deps) SettleHolds(ctx context.Context) {
	settled, err := d.DB.SettleEmailHolds(ctx, time.Now())
	if err != nil {
		d.logger().Error("settle held emails", "err", err)
		return
	}
	for _, one := range settled {
		action := HoldApplied
		if one.Taken {
			action = HoldDropped
		}
		d.noteHold(ctx, one.SiteID, nil, action, one.UserID, one.Username, one.From, one.To)
	}
}

func (d Deps) confirmHold(r *http.Request, user *db.User, state db.AccountEmail, secret string, now time.Time) (string, error) {
	ctx := r.Context()
	if state.Held == "" || state.HeldUntil != nil ||
		!d.Tokens.Check(secret, token.Custom(purposeHold, id(user.ID), lower(state.Held)), now) {
		return "dead", nil
	}
	taken, err := d.DB.VerifiedEmailTaken(ctx, state.Held, user.ID)
	if err != nil {
		return "", err
	}
	if taken {
		return "taken", nil
	}
	until := now.Add(HoldPeriod)
	done, err := d.DB.ConfirmEmailHold(ctx, user.ID, state.Held, until)
	if err != nil || !done {
		return "dead", err
	}
	if state.Email != "" && state.VerifiedAt != nil {
		if err := d.warnHold(r, user, state, until); err != nil {
			return "", err
		}
	}
	d.noteHold(ctx, state.HeldSite, user, HoldConfirmed, user.ID, user.Username, state.Email, state.Held)
	return "held", nil
}

func (d Deps) cancelHold(ctx context.Context, user *db.User, state db.AccountEmail, secret string, now time.Time) (string, error) {
	if state.Held == "" ||
		!d.holdTokens().Check(secret, token.Custom(purposeCancel, id(user.ID), lower(state.Held), lower(state.Email)), now) {
		return "dead", nil
	}
	done, err := d.DB.DropEmailHold(ctx, user.ID, state.Held)
	if err != nil || !done {
		return "dead", err
	}
	d.noteHold(ctx, state.HeldSite, user, HoldCancelled, user.ID, user.Username, state.Email, state.Held)
	return "hold-cancelled", nil
}

func (d Deps) warnHold(r *http.Request, user *db.User, state db.AccountEmail, until time.Time) error {
	ctx := r.Context()
	loc := d.Bundle.For(ctx)
	value := token.Custom(purposeCancel, id(user.ID), lower(state.Held), lower(state.Email))
	url := d.linkWith(d.holdTokens(), r, purposeCancel, user.ID, value)
	when := until.In(site.Zone(ctx)).Format("2006-01-02 15:04 MST")
	return d.Mail.Send(ctx, []string{state.Email},
		loc.T("email.hold-warn-subject", "site", d.siteTitle(ctx)),
		loc.T("email.hold-warn-body", "name", user.DisplayLabel(), "email", masked(state.Held),
			"when", when, "link", url, "site", d.siteTitle(ctx)))
}

func (d Deps) noteHold(ctx context.Context, siteID *int64, by *db.User, action string, userID int64, username, from, to string) {
	label := username + " · " + masked(from) + " → " + masked(to)
	attrs := []any{"action", action, "user", userID, "from", masked(from), "to", masked(to)}
	if siteID != nil {
		attrs = append(attrs, "site", *siteID)
	}
	if by != nil {
		attrs = append(attrs, "by", by.ID)
	}
	d.logger().Info("email change", attrs...)
	if siteID == nil {
		return
	}
	note := db.AdminNote{
		Action: action, Screen: holdScreen, Target: strconv.FormatInt(userID, 10),
		Label: label, SiteID: *siteID, At: time.Now().UTC(),
	}
	if by != nil {
		note.UserID = &by.ID
		note.Name = by.Username
	}
	if err := d.DB.WriteAdminNote(ctx, note); err != nil {
		d.logger().Error("record email change", "user", userID, "err", err)
	}
}
