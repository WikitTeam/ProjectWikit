package auth

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/session"
)

const (
	secret   = "test-secret-key"
	password = "pbkdf2_sha256$1000000$abc$def="
	userID   = 42
)

type fakeSessions struct {
	data    map[string]string
	err     error
	deleted []string
}

func (f *fakeSessions) SessionByKey(_ context.Context, key string) (string, time.Time, error) {
	if f.err != nil {
		return "", time.Time{}, f.err
	}
	data, ok := f.data[key]
	if !ok {
		return "", time.Time{}, db.ErrNotFound
	}
	return data, time.Now().Add(time.Hour), nil
}

func (f *fakeSessions) DeleteSession(_ context.Context, key string) error {
	f.deleted = append(f.deleted, key)
	return nil
}

type fakeUsers struct {
	user     *db.User
	password string
	err      error
}

func (f *fakeUsers) UserForSession(_ context.Context, _ int64) (*db.User, string, error) {
	if f.err != nil {
		return nil, "", f.err
	}
	return f.user, f.password, nil
}

func activeUser() *db.User {
	return &db.User{ID: userID, Type: db.UserTypeNormal, Username: "seeduser", IsActive: true}
}

func quietLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func signedSession(t *testing.T, store *session.Store, hash string) string {
	t.Helper()
	data, err := store.Encode(map[string]any{
		session.AuthUserID:      "42",
		session.AuthUserBackend: "django.contrib.auth.backends.ModelBackend",
		session.AuthUserHash:    hash,
	})
	if err != nil {
		t.Fatalf("Encode(...) err = %v, want nil", err)
	}
	return data
}

type fixture struct {
	resolver *Resolver
	sessions *fakeSessions
	users    *fakeUsers
	store    *session.Store
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	store := session.New(secret)
	sessions := &fakeSessions{data: map[string]string{}}
	users := &fakeUsers{user: activeUser(), password: password}
	return &fixture{
		resolver: NewResolver(store, sessions, users, quietLog()),
		sessions: sessions,
		users:    users,
		store:    store,
	}
}

func (f *fixture) request(t *testing.T, key string) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/main", nil)
	if key != "" {
		r.AddCookie(&http.Cookie{Name: session.CookieName, Value: key})
	}
	return r
}

func (f *fixture) resolveWith(t *testing.T, key, hash string) *db.User {
	t.Helper()
	f.sessions.data[key] = signedSession(t, f.store, hash)
	return f.resolver.resolve(f.request(t, key))
}

func TestResolveSignedInUser(t *testing.T) {
	f := newFixture(t)
	got := f.resolveWith(t, "abc", f.store.AuthHash(password))

	if got == nil {
		t.Fatal("resolve(...) = nil, want the session's user")
	}
	if got.ID != userID {
		t.Errorf("resolve(...).ID = %d, want %d", got.ID, userID)
	}
}

func TestResolveNoCookie(t *testing.T) {
	f := newFixture(t)
	if got := f.resolver.resolve(f.request(t, "")); got != nil {
		t.Errorf("resolve(no cookie) = %v, want nil", got)
	}
}

func TestResolveUnknownSessionKey(t *testing.T) {
	f := newFixture(t)
	if got := f.resolver.resolve(f.request(t, "missing")); got != nil {
		t.Errorf("resolve(unknown key) = %v, want nil", got)
	}
}

func TestResolveRejectsForgedSessionData(t *testing.T) {
	f := newFixture(t)
	f.sessions.data["abc"] = signedSession(t, session.New("wrong-secret"), f.store.AuthHash(password))

	if got := f.resolver.resolve(f.request(t, "abc")); got != nil {
		t.Errorf("resolve(session signed with another secret) = %v, want nil", got)
	}
}

func TestResolveRejectsStalePasswordHash(t *testing.T) {
	f := newFixture(t)
	got := f.resolveWith(t, "abc", f.store.AuthHash("pbkdf2_sha256$1000000$abc$old="))

	if got != nil {
		t.Errorf("resolve(session from before a password change) = %v, want nil", got)
	}
	if len(f.sessions.deleted) != 1 || f.sessions.deleted[0] != "abc" {
		t.Errorf("deleted sessions = %v, want [abc]", f.sessions.deleted)
	}
}

func TestResolveRejectsMissingPasswordHash(t *testing.T) {
	f := newFixture(t)
	if got := f.resolveWith(t, "abc", ""); got != nil {
		t.Errorf("resolve(session with no auth hash) = %v, want nil", got)
	}
}

func TestResolveAcceptsFallbackSecret(t *testing.T) {
	rotated := session.New("new-secret", secret)
	f := newFixture(t)
	f.resolver = NewResolver(rotated, f.sessions, f.users, quietLog())
	f.sessions.data["abc"] = signedSession(t, session.New(secret), session.New(secret).AuthHash(password))

	if got := f.resolver.resolve(f.request(t, "abc")); got == nil {
		t.Error("resolve(session from before a key rotation) = nil, want the session's user")
	}
}

func TestResolveRejectsInactiveUser(t *testing.T) {
	f := newFixture(t)
	f.users.user.IsActive = false

	if got := f.resolveWith(t, "abc", f.store.AuthHash(password)); got != nil {
		t.Errorf("resolve(inactive user) = %v, want nil", got)
	}
}

func TestResolveRejectsUserBannedUntilAFutureDate(t *testing.T) {
	f := newFixture(t)
	until := time.Now().Add(time.Hour)
	f.users.user.InactiveUntil = &until

	if got := f.resolveWith(t, "abc", f.store.AuthHash(password)); got != nil {
		t.Errorf("resolve(user banned until a future date) = %v, want nil", got)
	}
}

func TestResolveDeletedUser(t *testing.T) {
	f := newFixture(t)
	f.users.err = db.ErrNotFound

	if got := f.resolveWith(t, "abc", f.store.AuthHash(password)); got != nil {
		t.Errorf("resolve(deleted user) = %v, want nil", got)
	}
}

func TestResolveDatabaseErrorIsAnonymous(t *testing.T) {
	f := newFixture(t)
	f.sessions.err = errors.New("connection refused")

	if got := f.resolver.resolve(f.request(t, "abc")); got != nil {
		t.Errorf("resolve(with a failing database) = %v, want nil", got)
	}
}

func TestMiddlewarePutsUserOnTheContext(t *testing.T) {
	f := newFixture(t)
	f.sessions.data["abc"] = signedSession(t, f.store, f.store.AuthHash(password))

	var seen *db.User
	h := f.resolver.Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = FromContext(r.Context())
	}))
	h.ServeHTTP(httptest.NewRecorder(), f.request(t, "abc"))

	if seen == nil {
		t.Fatal("FromContext(...) = nil, want the session's user")
	}
	if seen.Username != "seeduser" {
		t.Errorf("FromContext(...).Username = %q, want %q", seen.Username, "seeduser")
	}
}

func TestFromContextWithoutMiddleware(t *testing.T) {
	if got := FromContext(context.Background()); got != nil {
		t.Errorf("FromContext(bare context) = %v, want nil", got)
	}
}
