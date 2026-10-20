package account

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/wikidotclient"
)

const (
	claimCodeLife     = 15 * time.Minute
	claimResendPause  = time.Minute
	claimMaxAttempts  = 5
	claimCodeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
)

type claimStore interface {
	WikidotUserID(ctx context.Context, userID int64) (int64, error)
	ReserveClaimCode(ctx context.Context, userID int64, hash string, now, expires, notBefore time.Time) (bool, error)
	DropClaimCode(ctx context.Context, userID int64, hash string) error
	UseClaimCode(ctx context.Context, userID int64, hash string, now time.Time, maxAttempts int) (db.ClaimOutcome, error)
}

type wikidotMessenger interface {
	Username() string
	LookUp(ctx context.Context, name string) (int64, error)
	Send(ctx context.Context, to int64, subject, source string) error
}

type LocalVerifier struct {
	Store   claimStore
	Wikidot wikidotMessenger
	Bundle  *i18n.Bundle
	Log     *slog.Logger
	Now     func() time.Time
	Subject string
	Body    string
}

var _ Verifier = (*LocalVerifier)(nil)

func NewLocalVerifier(store *db.DB, client *wikidotclient.Client, bundle *i18n.Bundle, log *slog.Logger) *LocalVerifier {
	return &LocalVerifier{Store: store, Wikidot: client, Bundle: bundle, Log: log, Now: time.Now}
}

func (v *LocalVerifier) Send(ctx context.Context, target ClaimTarget) error {
	to, err := v.recipient(ctx, target)
	if err != nil {
		return err
	}

	code, err := newClaimCode()
	if err != nil {
		return v.internal("make claim code", err)
	}
	hash := claimHash(target.UserID, code)
	now := v.Now()
	reserved, err := v.Store.ReserveClaimCode(ctx, target.UserID, hash, now, now.Add(claimCodeLife), now.Add(-claimResendPause))
	if err != nil {
		return v.internal("reserve claim code", err)
	}
	if !reserved {
		return refused(CodeRateLimited)
	}

	subject, body := v.message(ctx, target.Site, code)
	if err := v.Wikidot.Send(ctx, to, subject, body); err != nil {
		if dropErr := v.Store.DropClaimCode(context.WithoutCancel(ctx), target.UserID, hash); dropErr != nil {
			v.Log.Error("drop claim code", "user", target.UserID, "err", dropErr)
		}
		return v.wikidotError("send claim code", target, err)
	}
	return nil
}

func (v *LocalVerifier) recipient(ctx context.Context, target ClaimTarget) (int64, error) {
	id, err := v.Store.WikidotUserID(ctx, target.UserID)
	if err != nil {
		return 0, v.internal("read wikidot id", err)
	}
	if id > 0 {
		return id, nil
	}
	id, err = v.Wikidot.LookUp(ctx, target.Name)
	if err != nil {
		return 0, v.wikidotError("look up wikidot user", target, err)
	}
	return id, nil
}

func (v *LocalVerifier) Verify(ctx context.Context, target ClaimTarget, code string) error {
	outcome, err := v.Store.UseClaimCode(ctx, target.UserID, claimHash(target.UserID, code), v.Now(), claimMaxAttempts)
	if err != nil {
		return v.internal("check claim code", err)
	}
	switch outcome {
	case db.ClaimMatched:
		return nil
	case db.ClaimMissing:
		return refused(CodeNoPendingCode)
	case db.ClaimExpired:
		return refused(CodeExpired)
	case db.ClaimLocked:
		return refused(CodeTooManyAttempts)
	default:
		return refused(CodeWrong)
	}
}

func (v *LocalVerifier) wikidotError(what string, target ClaimTarget, err error) error {
	switch {
	case errors.Is(err, wikidotclient.ErrNoUser):
		return refused(CodeUserNotFound)
	case errors.Is(err, wikidotclient.ErrRefused):
		v.Log.Warn(what, "user", target.Name, "err", err)
		return refused(CodeMessagesRefused)
	case errors.Is(err, wikidotclient.ErrPaused):
		return refused(CodeSignInPaused)
	case errors.Is(err, wikidotclient.ErrLogin):
		v.Log.Error("wikidot refused to sign in the account set in pwikit.toml", "account", v.Wikidot.Username())
		return refused(CodeLoginFailed)
	case errors.Is(err, wikidotclient.ErrUnavailable):
		v.Log.Warn(what, "user", target.Name, "err", err)
		return refused(CodeWikidotUnavailable)
	default:
		return v.internal(what, err)
	}
}

func (v *LocalVerifier) internal(what string, err error) error {
	v.Log.Error(what, "err", err)
	return refused(CodeInternal)
}

func newClaimCode() (string, error) {
	var b strings.Builder
	limit := big.NewInt(int64(len(claimCodeAlphabet)))
	for i := 0; i < 8; i++ {
		if i == 4 {
			b.WriteByte('-')
		}
		n, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", err
		}
		b.WriteByte(claimCodeAlphabet[n.Int64()])
	}
	return b.String(), nil
}

func claimHash(userID int64, code string) string {
	clean := strings.Map(func(r rune) rune {
		if r == '-' || r == ' ' {
			return -1
		}
		return r
	}, strings.ToUpper(strings.TrimSpace(code)))
	sum := sha256.Sum256([]byte(strconv.FormatInt(userID, 10) + ":" + clean))
	return hex.EncodeToString(sum[:])
}

func (v *LocalVerifier) message(ctx context.Context, site, code string) (string, string) {
	minutes := strconv.Itoa(int(claimCodeLife / time.Minute))
	loc := v.Bundle.For(ctx)
	subject := loc.T("claim.message-subject", "site", site)
	body := loc.T("claim.message-body", "code", code, "site", site, "minutes", minutes)
	fill := strings.NewReplacer("{site}", site, "{code}", code, "{minutes}", minutes)
	if v.Subject != "" {
		subject = fill.Replace(v.Subject)
	}
	if v.Body != "" {
		body = fill.Replace(strings.TrimSpace(v.Body))
	}
	return subject, body
}
