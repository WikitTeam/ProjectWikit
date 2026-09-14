package site

import (
	"context"
	"time"

	"github.com/WikitTeam/ProjectWikit/internal/db"
	"github.com/WikitTeam/ProjectWikit/internal/timezone"
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

func Zone(ctx context.Context) *time.Location {
	s := FromContext(ctx)
	if s == nil {
		return time.UTC
	}
	return timezone.Load(s.TimeZone)
}
