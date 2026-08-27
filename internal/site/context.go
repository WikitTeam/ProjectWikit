package site

import (
	"context"

	"github.com/WikitTeam/ProjectWikit/internal/db"
)

type contextKey struct{}

func WithSite(ctx context.Context, s *db.Site) context.Context {
	return context.WithValue(ctx, contextKey{}, s)
}

// FromContext returns nil when the request never passed through HostRules,
// which is every request in a test that builds its own.
func FromContext(ctx context.Context) *db.Site {
	s, _ := ctx.Value(contextKey{}).(*db.Site)
	return s
}
