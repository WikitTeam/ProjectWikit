// Package auth resolves the signed-in user from the session cookie.
package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/session"
)

type Sessions interface {
	SessionByKey(ctx context.Context, key string) (string, time.Time, error)
	DeleteSession(ctx context.Context, key string) error
}

type Users interface {
	UserForSession(ctx context.Context, id int64) (*db.User, string, error)
}

type Resolver struct {
	store    *session.Store
	sessions Sessions
	users    Users
	log      *slog.Logger
}

func NewResolver(store *session.Store, sessions Sessions, users Users, log *slog.Logger) *Resolver {
	return &Resolver{store: store, sessions: sessions, users: users, log: log}
}

type contextKey struct{}

// Middleware puts the signed-in user on the request context. Anonymous is not
// an error and not a separate branch: FromContext returns nil for it.
func (r *Resolver) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		user := r.resolve(req)
		next.ServeHTTP(w, req.WithContext(context.WithValue(req.Context(), contextKey{}, user)))
	})
}

// FromContext returns nil when nobody is signed in.
func FromContext(ctx context.Context) *db.User {
	user, _ := ctx.Value(contextKey{}).(*db.User)
	return user
}

func (r *Resolver) resolve(req *http.Request) *db.User {
	cookie, err := req.Cookie(session.CookieName)
	if err != nil || cookie.Value == "" {
		return nil
	}
	ctx := req.Context()

	data, _, err := r.sessions.SessionByKey(ctx, cookie.Value)
	if err != nil {
		if !errors.Is(err, db.ErrNotFound) {
			r.log.Error("read session", "err", err)
		}
		return nil
	}
	decoded, err := r.store.Decode(data)
	if err != nil {
		return nil
	}

	raw, ok := session.UserID(decoded)
	if !ok {
		return nil
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil
	}

	user, password, err := r.users.UserForSession(ctx, id)
	if err != nil {
		if !errors.Is(err, db.ErrNotFound) {
			r.log.Error("read session user", "id", id, "err", err)
		}
		return nil
	}
	if !user.ActiveAt(time.Now()) {
		return nil
	}

	hash, _ := decoded[session.AuthUserHash].(string)
	if !r.store.AuthHashMatches(password, hash) {
		// The password changed under this session. Leaving the row would let the same
		// dead cookie keep costing a query.
		if err := r.sessions.DeleteSession(ctx, cookie.Value); err != nil {
			r.log.Error("drop stale session", "err", err)
		}
		return nil
	}
	return user
}
