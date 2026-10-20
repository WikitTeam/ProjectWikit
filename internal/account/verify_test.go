package account

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/i18n"
	"github.com/WikitTeam/ProjectWikit/internal/wikidotclient"
)

func keyBundle(t *testing.T) *i18n.Bundle {
	t.Helper()
	b, err := i18n.LoadKeys()
	if err != nil {
		t.Fatalf("LoadKeys() err = %v, want nil", err)
	}
	return b
}

func TestWikitVerifierReadsTheAnswer(t *testing.T) {
	tests := []struct {
		name   string
		answer string
		want   error
	}{
		{"success", `{"status":"success"}`, nil},
		{"coded", `{"status":"error","code":"messages_refused","message":"The user does not accept messages"}`,
			&RefusedError{Code: CodeMessagesRefused, Message: "The user does not accept messages"}},
		{"old service", `{"status":"error","message":"an old message"}`, &RefusedError{Message: "an old message"}},
		{"garbage", `<html>`, ErrVerifierUnreachable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var seen string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seen = r.PostFormValue("user")
				_, _ = io.WriteString(w, tt.answer)
			}))
			defer server.Close()
			v := &WikitVerifier{Base: server.URL, Client: server.Client()}

			err := v.Send(context.Background(), ClaimTarget{UserID: 1, Name: "AshlyS_M"})
			if seen != "AshlyS_M" {
				t.Errorf("Send() posted user %q, want %q", seen, "AshlyS_M")
			}
			var got, want *RefusedError
			if errors.As(tt.want, &want) {
				if !errors.As(err, &got) || *got != *want {
					t.Errorf("Send() err = %v, want %v", err, tt.want)
				}
				return
			}
			if !errors.Is(err, tt.want) {
				t.Errorf("Send() err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestVerifyError(t *testing.T) {
	loc := keyBundle(t).Localizer("")
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"unreachable", ErrVerifierUnreachable, loc.T("signup.error-verify-unreachable")},
		{"refused", refused(CodeMessagesRefused), loc.T("signup.verify-messages-refused")},
		{"wrong", refused(CodeWrong), loc.T("signup.error-code-wrong")},
		{"login failed", refused(CodeLoginFailed), loc.T("signup.verify-unavailable")},
		{"unknown code", refused("brand_new"), loc.T("signup.verify-failed")},
		{"internal", refused(CodeInternal), loc.T("signup.verify-failed")},
		{"old service", &RefusedError{Message: "an old message"}, "an old message"},
		{"other", errors.New("boom"), loc.T("signup.verify-failed")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := verifyError(loc, tt.err); got != tt.want {
				t.Errorf("verifyError(%v) = %q, want %q", tt.err, got, tt.want)
			}
		})
	}
}

func TestClaimTarget(t *testing.T) {
	tests := []struct {
		display, typed, want string
	}{
		{"AshlyS_M", "ashlys_m", "AshlyS_M"},
		{"", "zjm20071212", "zjm20071212"},
	}
	for _, tt := range tests {
		got := claimTarget(&db.User{ID: 3, DisplayName: tt.display}, tt.typed, "Wiki")
		if got.Name != tt.want {
			t.Errorf("claimTarget(%q, %q).Name = %q, want %q", tt.display, tt.typed, got.Name, tt.want)
		}
	}
}

func TestClaimHashIgnoresLayout(t *testing.T) {
	want := claimHash(5, "ABCD-EFGH")
	for _, typed := range []string{"abcd-efgh", " ABCDEFGH ", "ABCD EFGH"} {
		if got := claimHash(5, typed); got != want {
			t.Errorf("claimHash(5, %q) = %q, want %q", typed, got, want)
		}
	}
	if claimHash(6, "ABCD-EFGH") == want {
		t.Errorf("claimHash(6, ...) = claimHash(5, ...), want them to differ")
	}
}

func TestNewClaimCode(t *testing.T) {
	code, err := newClaimCode()
	if err != nil {
		t.Fatalf("newClaimCode() err = %v, want nil", err)
	}
	if len(code) != 9 || code[4] != '-' {
		t.Errorf("newClaimCode() = %q, want XXXX-XXXX", code)
	}
	for i, c := range code {
		if i != 4 && !strings.ContainsRune(claimCodeAlphabet, c) {
			t.Errorf("newClaimCode() = %q, want only %q", code, claimCodeAlphabet)
		}
	}
}

type fakeClaimStore struct {
	wikidotID int64
	hash      string
	sentAt    time.Time
	dropped   bool
	outcome   db.ClaimOutcome
}

func (f *fakeClaimStore) WikidotUserID(context.Context, int64) (int64, error) {
	return f.wikidotID, nil
}

func (f *fakeClaimStore) ReserveClaimCode(_ context.Context, _ int64, hash string, now, _, notBefore time.Time) (bool, error) {
	if !f.sentAt.IsZero() && !f.sentAt.Before(notBefore) {
		return false, nil
	}
	f.hash, f.sentAt = hash, now
	return true, nil
}

func (f *fakeClaimStore) DropClaimCode(_ context.Context, _ int64, hash string) error {
	if f.hash == hash {
		f.dropped = true
	}
	return nil
}

func (f *fakeClaimStore) UseClaimCode(context.Context, int64, string, time.Time, int) (db.ClaimOutcome, error) {
	return f.outcome, nil
}

type fakeMessenger struct {
	lookedUp string
	to       int64
	subject  string
	body     string
	err      error
}

func (f *fakeMessenger) Username() string { return "bot" }

func (f *fakeMessenger) LookUp(_ context.Context, name string) (int64, error) {
	f.lookedUp = name
	return 99, nil
}

func (f *fakeMessenger) Send(_ context.Context, to int64, subject, body string) error {
	f.to, f.subject, f.body = to, subject, body
	return f.err
}

func newLocal(t *testing.T, store *fakeClaimStore, wd *fakeMessenger) *LocalVerifier {
	t.Helper()
	b, err := i18n.Load("")
	if err != nil {
		t.Fatalf("Load() err = %v, want nil", err)
	}
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	return &LocalVerifier{Store: store, Wikidot: wd, Bundle: b,
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Now: func() time.Time { return now }}
}

func TestLocalSendUsesTheStoredWikidotID(t *testing.T) {
	store := &fakeClaimStore{wikidotID: 8305255}
	wd := &fakeMessenger{}
	v := newLocal(t, store, wd)
	if err := v.Send(context.Background(), ClaimTarget{UserID: 1, Name: "AceAttackAssassin"}); err != nil {
		t.Fatalf("Send() err = %v, want nil", err)
	}
	if wd.to != 8305255 {
		t.Errorf("Send() to = %d, want 8305255", wd.to)
	}
	if wd.lookedUp != "" {
		t.Errorf("Send() looked up %q, want no lookup", wd.lookedUp)
	}
	if store.hash == "" {
		t.Errorf("Send() stored no hash, want one")
	}
}

func TestLocalSendLooksUpWithoutAStoredID(t *testing.T) {
	wd := &fakeMessenger{}
	v := newLocal(t, &fakeClaimStore{}, wd)
	if err := v.Send(context.Background(), ClaimTarget{UserID: 1, Name: "AshlyS_M"}); err != nil {
		t.Fatalf("Send() err = %v, want nil", err)
	}
	if wd.lookedUp != "AshlyS_M" || wd.to != 99 {
		t.Errorf("Send() looked up %q and sent to %d, want AshlyS_M and 99", wd.lookedUp, wd.to)
	}
}

func TestLocalSendTooSoon(t *testing.T) {
	store := &fakeClaimStore{wikidotID: 1}
	v := newLocal(t, store, &fakeMessenger{})
	target := ClaimTarget{UserID: 1, Name: "x"}
	if err := v.Send(context.Background(), target); err != nil {
		t.Fatalf("Send() err = %v, want nil", err)
	}
	var got *RefusedError
	if err := v.Send(context.Background(), target); !errors.As(err, &got) || got.Code != CodeRateLimited {
		t.Errorf("second Send() err = %v, want %s", err, CodeRateLimited)
	}
}

func TestLocalSendFailureMapping(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{wikidotclient.ErrRefused, CodeMessagesRefused},
		{wikidotclient.ErrLogin, CodeLoginFailed},
		{fmt.Errorf("%w: %w", wikidotclient.ErrPaused, wikidotclient.ErrLogin), CodeSignInPaused},
		{wikidotclient.ErrUnavailable, CodeWikidotUnavailable},
		{errors.New("boom"), CodeInternal},
	}
	for _, tt := range tests {
		store := &fakeClaimStore{wikidotID: 1}
		v := newLocal(t, store, &fakeMessenger{err: tt.err})
		var got *RefusedError
		if err := v.Send(context.Background(), ClaimTarget{UserID: 1}); !errors.As(err, &got) || got.Code != tt.want {
			t.Errorf("Send() with %v err = %v, want %s", tt.err, err, tt.want)
		}
		if !store.dropped {
			t.Errorf("Send() with %v kept its code, want it dropped", tt.err)
		}
	}
}

func TestLocalVerifyOutcomes(t *testing.T) {
	tests := []struct {
		outcome db.ClaimOutcome
		want    string
	}{
		{db.ClaimMatched, ""},
		{db.ClaimMissing, CodeNoPendingCode},
		{db.ClaimExpired, CodeExpired},
		{db.ClaimWrong, CodeWrong},
		{db.ClaimLocked, CodeTooManyAttempts},
	}
	for _, tt := range tests {
		v := newLocal(t, &fakeClaimStore{outcome: tt.outcome}, &fakeMessenger{})
		err := v.Verify(context.Background(), ClaimTarget{UserID: 1}, "ABCD-EFGH")
		got := ""
		var refusal *RefusedError
		if errors.As(err, &refusal) {
			got = refusal.Code
		}
		if got != tt.want {
			t.Errorf("Verify() with outcome %d err = %v, want %q", tt.outcome, err, tt.want)
		}
	}
}

func TestLocalSendWritesTheCodeIntoTheMessage(t *testing.T) {
	store := &fakeClaimStore{wikidotID: 1}
	wd := &fakeMessenger{}
	v := newLocal(t, store, wd)
	if err := v.Send(context.Background(), ClaimTarget{UserID: 4, Site: "Probe Wiki"}); err != nil {
		t.Fatalf("Send() err = %v, want nil", err)
	}
	found := boldCode.FindStringSubmatch(wd.body)
	if found == nil {
		t.Fatalf("Send() body = %q, want a bold code", wd.body)
	}
	if claimHash(4, found[1]) != store.hash {
		t.Errorf("claimHash(4, %q) = %q, want the stored %q", found[1], claimHash(4, found[1]), store.hash)
	}
}

var boldCode = regexp.MustCompile(`\*\*([0-9A-Z]{4}-[0-9A-Z]{4})\*\*`)

func TestLocalSendCustomMessage(t *testing.T) {
	store := &fakeClaimStore{wikidotID: 1}
	wd := &fakeMessenger{}
	v := newLocal(t, store, wd)
	v.Subject = "{site} code"
	v.Body = "\n**{code}** for {site}, {minutes} min\n"
	if err := v.Send(context.Background(), ClaimTarget{UserID: 4, Site: "Probe Wiki"}); err != nil {
		t.Fatalf("Send() err = %v, want nil", err)
	}
	if wd.subject != "Probe Wiki code" {
		t.Errorf("Send() subject = %q, want %q", wd.subject, "Probe Wiki code")
	}
	found := boldCode.FindStringSubmatch(wd.body)
	if found == nil {
		t.Fatalf("Send() body = %q, want a bold code", wd.body)
	}
	if want := "**" + found[1] + "** for Probe Wiki, 15 min"; wd.body != want {
		t.Errorf("Send() body = %q, want %q", wd.body, want)
	}
}
